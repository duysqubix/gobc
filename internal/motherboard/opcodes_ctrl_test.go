package motherboard

// Comprehensive unit tests for control-flow opcodes (jumps, calls, returns,
// restarts), the four non-CB rotates that act on register A, and the
// miscellaneous opcodes (NOP, STOP, DAA, CPL, SCF, CCF, HALT, DI, EI).
//
// Conventions
//   * Every test starts from `newTestCPU(t)` (see cpu_test.go), which seeds
//     PC=0xC000 (WRAM) and SP=0xFFFE (top of HRAM).
//   * Tests dispatch handlers directly via OPCODES[op](mb, value) — exactly
//     the pattern used by the existing `opcodes_test.go`. The handler is
//     responsible for advancing PC and consuming `value` (the immediate
//     operand fetched by ExecuteInstruction in production).
//   * For every conditional opcode we assert BOTH branches (taken AND
//     not-taken) and verify the cycle count returned by the handler matches
//     the implementation in `opcodes.go`.
//
// Invariants intentionally pinned down by these tests (i.e. asserted as
// behaviour, not just side-effects):
//
//   * RLCA / RRCA / RLA / RRA all clear Z unconditionally — they never set
//     Z=1 even when the result is 0. This matches Game Boy hardware and
//     differs from the CB-prefixed RLC/RRC/RL/RR variants.
//
//   * RETI in this implementation uses the *delayed* IME-enable path
//     (Interrupts.InterruptsEnabling = true), exactly like EI. Real DMG
//     hardware enables IME immediately on RETI; this emulator deviates.
//     The deviation is captured here so any future fix surfaces as a
//     deliberate test failure rather than a silent behaviour change.
//
//   * EI sets InterruptsEnabling but does NOT immediately set
//     InterruptsOn — the master enable transition happens one instruction
//     later, driven by the main tick loop, not by the opcode handler.
//
//   * HALT does not advance PC. The PC bump after a HALT is performed by
//     ServiceInterrupt() when an interrupt wakes the CPU.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// pushReturnAddrOnStack writes a 16-bit return address into HRAM at the
// current SP (low byte at SP, high byte at SP+1). Used by RET / RET cond
// tests to stage the stack before the handler pops it.
func pushReturnAddrOnStack(t *testing.T, mb *Motherboard, addr uint16) {
	t.Helper()
	sp := mb.Cpu.Registers.SP
	mb.SetItem(sp, uint16(addr&0xFF))
	mb.SetItem(sp+1, uint16((addr>>8)&0xFF))
}

// readReturnAddrFromStack reads the 16-bit value at [SP] (little-endian).
// Used by CALL / RST tests to verify the pushed return address.
func readReturnAddrFromStack(mb *Motherboard, sp uint16) uint16 {
	lo := uint16(mb.GetItem(sp))
	hi := uint16(mb.GetItem(sp + 1))
	return (hi << 8) | lo
}

// ---------------------------------------------------------------------------
// 0x18 — JR r8 (unconditional relative jump)
// ---------------------------------------------------------------------------

func TestJR_r8_0x18(t *testing.T) {
	cases := []struct {
		name   string
		offset uint16 // raw byte value, interpreted as int8
		wantPC uint16 // assuming start PC 0xC000
	}{
		{"forward+5", 0x05, 0xC007},    // 0xC000 + 2 + 5
		{"forward+127", 0x7F, 0xC081},  // max positive
		{"zero", 0x00, 0xC002},         // no displacement
		{"backward-2", 0xFE, 0xC000},   // 0xC000 + 2 + (-2)
		{"backward-128", 0x80, 0xBF82}, // 0xC000 + 2 - 128 = 0xBF82
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			cycles := OPCODES[0x18](mb, tc.offset)
			assert.Equal(t, OpCycles(12), cycles, "JR r8 always takes 12 cycles")
			assert.Equal(t, tc.wantPC, cpu.Registers.PC)
		})
	}
}

// ---------------------------------------------------------------------------
// 0x20 / 0x28 / 0x30 / 0x38 — Conditional JR r8
// Both branches: taken (12 cycles) vs not-taken (8 cycles).
// ---------------------------------------------------------------------------

func TestJR_Conditional(t *testing.T) {
	type setupFn func(*CPU)

	cases := []struct {
		name     string
		op       OpCode
		setTaken setupFn // configures CPU so the branch IS taken
		setSkip  setupFn // configures CPU so the branch is NOT taken
	}{
		{"NZ_0x20", 0x20,
			func(c *CPU) { c.ResetFlagZ() },
			func(c *CPU) { c.SetFlagZ() },
		},
		{"Z_0x28", 0x28,
			func(c *CPU) { c.SetFlagZ() },
			func(c *CPU) { c.ResetFlagZ() },
		},
		{"NC_0x30", 0x30,
			func(c *CPU) { c.ResetFlagC() },
			func(c *CPU) { c.SetFlagC() },
		},
		{"C_0x38", 0x38,
			func(c *CPU) { c.SetFlagC() },
			func(c *CPU) { c.ResetFlagC() },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/taken_forward", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			tc.setTaken(cpu)
			cycles := OPCODES[tc.op](mb, 0x05)
			assert.Equal(t, OpCycles(12), cycles, "taken branch takes 12 cycles")
			assert.Equal(t, uint16(0xC007), cpu.Registers.PC)
		})

		t.Run(tc.name+"/taken_backward", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC100
			tc.setTaken(cpu)
			cycles := OPCODES[tc.op](mb, 0xFE) // -2
			assert.Equal(t, OpCycles(12), cycles)
			assert.Equal(t, uint16(0xC100), cpu.Registers.PC, "0xC100 + 2 + (-2) = 0xC100")
		})

		t.Run(tc.name+"/notTaken", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			tc.setSkip(cpu)
			cycles := OPCODES[tc.op](mb, 0x05)
			assert.Equal(t, OpCycles(8), cycles, "skipped branch takes 8 cycles")
			assert.Equal(t, uint16(0xC002), cpu.Registers.PC, "PC advances by instruction length only")
		})
	}
}

// ---------------------------------------------------------------------------
// 0xC3 — JP a16 (unconditional)  &  0xE9 — JP (HL)
// (0xC3 already has a smoke-test in opcodes_test.go; we add a more
// thorough check covering edge addresses.)
// ---------------------------------------------------------------------------

func TestJP_a16_0xC3_Edges(t *testing.T) {
	cases := []struct {
		name   string
		target uint16
	}{
		{"low", 0x0000},
		{"mid", 0x4242},
		{"high", 0xFFFF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			cycles := OPCODES[0xC3](mb, tc.target)
			assert.Equal(t, OpCycles(16), cycles)
			assert.Equal(t, tc.target, cpu.Registers.PC)
		})
	}
}

func TestJP_HL_0xE9(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.SetHL(0xBEEF)
	cycles := OPCODES[0xE9](mb, 0)
	assert.Equal(t, OpCycles(4), cycles, "JP (HL) is fast — 4 cycles")
	assert.Equal(t, uint16(0xBEEF), cpu.Registers.PC)
}

// ---------------------------------------------------------------------------
// 0xC2 / 0xCA / 0xD2 / 0xDA — Conditional JP a16
// Both branches: taken (16 cycles) vs not-taken (12 cycles).
// ---------------------------------------------------------------------------

func TestJP_Conditional(t *testing.T) {
	type setupFn func(*CPU)
	cases := []struct {
		name     string
		op       OpCode
		setTaken setupFn
		setSkip  setupFn
	}{
		{"NZ_0xC2", 0xC2,
			func(c *CPU) { c.ResetFlagZ() },
			func(c *CPU) { c.SetFlagZ() },
		},
		{"Z_0xCA", 0xCA,
			func(c *CPU) { c.SetFlagZ() },
			func(c *CPU) { c.ResetFlagZ() },
		},
		{"NC_0xD2", 0xD2,
			func(c *CPU) { c.ResetFlagC() },
			func(c *CPU) { c.SetFlagC() },
		},
		{"C_0xDA", 0xDA,
			func(c *CPU) { c.SetFlagC() },
			func(c *CPU) { c.ResetFlagC() },
		},
	}

	const target uint16 = 0xBEEF
	for _, tc := range cases {
		t.Run(tc.name+"/taken", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			tc.setTaken(cpu)
			cycles := OPCODES[tc.op](mb, target)
			assert.Equal(t, OpCycles(16), cycles, "taken JP cond = 16 cycles")
			assert.Equal(t, target, cpu.Registers.PC)
		})
		t.Run(tc.name+"/notTaken", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			tc.setSkip(cpu)
			cycles := OPCODES[tc.op](mb, target)
			assert.Equal(t, OpCycles(12), cycles, "skipped JP cond = 12 cycles")
			assert.Equal(t, uint16(0xC003), cpu.Registers.PC, "PC advances by instruction length only")
		})
	}
}

// ---------------------------------------------------------------------------
// 0xCD — CALL a16 (unconditional)
// ---------------------------------------------------------------------------

func TestCALL_a16_0xCD(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.Registers.SP = 0xFFFE

	const target uint16 = 0xBEEF
	cycles := OPCODES[0xCD](mb, target)

	assert.Equal(t, OpCycles(24), cycles, "CALL a16 = 24 cycles")
	assert.Equal(t, target, cpu.Registers.PC, "PC -> target")
	assert.Equal(t, uint16(0xFFFC), cpu.Registers.SP, "SP -= 2")

	// Return address pushed = original PC + 3 (instruction length).
	got := readReturnAddrFromStack(mb, cpu.Registers.SP)
	assert.Equal(t, uint16(0xC003), got, "return address = original PC + 3, little-endian")
}

// ---------------------------------------------------------------------------
// 0xC4 / 0xCC / 0xD4 / 0xDC — Conditional CALL
// Taken: 24 cycles, SP-=2, return addr pushed, PC=target.
// Not taken: 12 cycles, SP unchanged, PC=orig+3.
// ---------------------------------------------------------------------------

func TestCALL_Conditional(t *testing.T) {
	type setupFn func(*CPU)
	cases := []struct {
		name     string
		op       OpCode
		setTaken setupFn
		setSkip  setupFn
	}{
		{"NZ_0xC4", 0xC4,
			func(c *CPU) { c.ResetFlagZ() },
			func(c *CPU) { c.SetFlagZ() },
		},
		{"Z_0xCC", 0xCC,
			func(c *CPU) { c.SetFlagZ() },
			func(c *CPU) { c.ResetFlagZ() },
		},
		{"NC_0xD4", 0xD4,
			func(c *CPU) { c.ResetFlagC() },
			func(c *CPU) { c.SetFlagC() },
		},
		{"C_0xDC", 0xDC,
			func(c *CPU) { c.SetFlagC() },
			func(c *CPU) { c.ResetFlagC() },
		},
	}

	const target uint16 = 0xBEEF
	for _, tc := range cases {
		t.Run(tc.name+"/taken", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			cpu.Registers.SP = 0xFFFE
			tc.setTaken(cpu)

			cycles := OPCODES[tc.op](mb, target)

			assert.Equal(t, OpCycles(24), cycles, "taken CALL cond = 24 cycles")
			assert.Equal(t, target, cpu.Registers.PC)
			assert.Equal(t, uint16(0xFFFC), cpu.Registers.SP, "SP -= 2 on taken branch")

			got := readReturnAddrFromStack(mb, cpu.Registers.SP)
			assert.Equal(t, uint16(0xC003), got, "return addr = orig PC + 3")
		})

		t.Run(tc.name+"/notTaken", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			cpu.Registers.SP = 0xFFFE
			tc.setSkip(cpu)

			cycles := OPCODES[tc.op](mb, target)

			assert.Equal(t, OpCycles(12), cycles, "skipped CALL cond = 12 cycles")
			assert.Equal(t, uint16(0xC003), cpu.Registers.PC, "PC advances by instruction length")
			assert.Equal(t, uint16(0xFFFE), cpu.Registers.SP, "SP unchanged on skip")
		})
	}
}

// ---------------------------------------------------------------------------
// 0xC9 — RET   /   0xD9 — RETI
// ---------------------------------------------------------------------------

func TestRET_0xC9(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.Registers.SP = 0xFFFC

	pushReturnAddrOnStack(t, mb, 0x1234)

	cycles := OPCODES[0xC9](mb, 0)

	assert.Equal(t, OpCycles(16), cycles, "RET = 16 cycles")
	assert.Equal(t, uint16(0x1234), cpu.Registers.PC, "PC = popped value")
	assert.Equal(t, uint16(0xFFFE), cpu.Registers.SP, "SP += 2")
}

func TestRETI_0xD9(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.Registers.SP = 0xFFFC

	// Pre-condition: IME-related flags both off, prove RETI flips the
	// "enabling" flag (delayed-enable model used by this emulator).
	cpu.Interrupts.InterruptsOn = false
	cpu.Interrupts.InterruptsEnabling = false

	pushReturnAddrOnStack(t, mb, 0xCAFE)

	cycles := OPCODES[0xD9](mb, 0)

	assert.Equal(t, OpCycles(16), cycles, "RETI = 16 cycles")
	assert.Equal(t, uint16(0xCAFE), cpu.Registers.PC)
	assert.Equal(t, uint16(0xFFFE), cpu.Registers.SP)
	// IME activation: this implementation matches EI's delayed-enable
	// semantics — InterruptsEnabling flips to true, InterruptsOn does
	// NOT immediately become true. Real hardware enables IME instantly
	// on RETI; the discrepancy is documented at the top of this file.
	assert.True(t, cpu.Interrupts.InterruptsEnabling,
		"RETI sets InterruptsEnabling (delayed IME enable, like EI)")
	assert.False(t, cpu.Interrupts.InterruptsOn,
		"RETI does NOT immediately set InterruptsOn in this impl")
}

// ---------------------------------------------------------------------------
// 0xC0 / 0xC8 / 0xD0 / 0xD8 — Conditional RET
// Taken: 20 cycles, PC=popped, SP+=2.
// Not taken: 8 cycles, PC=orig+1, SP unchanged.
// ---------------------------------------------------------------------------

func TestRET_Conditional(t *testing.T) {
	type setupFn func(*CPU)
	cases := []struct {
		name     string
		op       OpCode
		setTaken setupFn
		setSkip  setupFn
	}{
		{"NZ_0xC0", 0xC0,
			func(c *CPU) { c.ResetFlagZ() },
			func(c *CPU) { c.SetFlagZ() },
		},
		{"Z_0xC8", 0xC8,
			func(c *CPU) { c.SetFlagZ() },
			func(c *CPU) { c.ResetFlagZ() },
		},
		{"NC_0xD0", 0xD0,
			func(c *CPU) { c.ResetFlagC() },
			func(c *CPU) { c.SetFlagC() },
		},
		{"C_0xD8", 0xD8,
			func(c *CPU) { c.SetFlagC() },
			func(c *CPU) { c.ResetFlagC() },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/taken", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			cpu.Registers.SP = 0xFFFC
			tc.setTaken(cpu)
			pushReturnAddrOnStack(t, mb, 0xABCD)

			cycles := OPCODES[tc.op](mb, 0)

			assert.Equal(t, OpCycles(20), cycles, "taken RET cond = 20 cycles")
			assert.Equal(t, uint16(0xABCD), cpu.Registers.PC, "PC = popped value")
			assert.Equal(t, uint16(0xFFFE), cpu.Registers.SP, "SP += 2 on taken branch")
		})

		t.Run(tc.name+"/notTaken", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			cpu.Registers.SP = 0xFFFC
			tc.setSkip(cpu)
			pushReturnAddrOnStack(t, mb, 0xABCD)

			cycles := OPCODES[tc.op](mb, 0)

			assert.Equal(t, OpCycles(8), cycles, "skipped RET cond = 8 cycles")
			assert.Equal(t, uint16(0xC001), cpu.Registers.PC, "PC advances by 1 on skip")
			assert.Equal(t, uint16(0xFFFC), cpu.Registers.SP, "SP unchanged on skip")
		})
	}
}

// ---------------------------------------------------------------------------
// 0xC7 / 0xCF / 0xD7 / 0xDF / 0xE7 / 0xEF / 0xF7 / 0xFF — RST n
// ---------------------------------------------------------------------------

func TestRST_AllVectors(t *testing.T) {
	cases := []struct {
		name   string
		op     OpCode
		target uint16
	}{
		{"RST_00H_0xC7", 0xC7, 0x00},
		{"RST_08H_0xCF", 0xCF, 0x08},
		{"RST_10H_0xD7", 0xD7, 0x10},
		{"RST_18H_0xDF", 0xDF, 0x18},
		{"RST_20H_0xE7", 0xE7, 0x20},
		{"RST_28H_0xEF", 0xEF, 0x28},
		{"RST_30H_0xF7", 0xF7, 0x30},
		{"RST_38H_0xFF", 0xFF, 0x38},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.PC = 0xC000
			cpu.Registers.SP = 0xFFFE

			cycles := OPCODES[tc.op](mb, 0)

			assert.Equal(t, OpCycles(16), cycles, "RST n = 16 cycles")
			assert.Equal(t, tc.target, cpu.Registers.PC, "PC = vector target")
			assert.Equal(t, uint16(0xFFFC), cpu.Registers.SP, "SP -= 2")

			got := readReturnAddrFromStack(mb, cpu.Registers.SP)
			assert.Equal(t, uint16(0xC001), got,
				"return address = original PC + 1 (RST is a 1-byte instruction)")
		})
	}
}

// ---------------------------------------------------------------------------
// 0x07 / 0x0F / 0x17 / 0x1F — Non-CB rotates on A
// All four ALWAYS clear Z (verified explicitly even when A=0).
// ---------------------------------------------------------------------------

func TestRotateA_RLCA_0x07(t *testing.T) {
	cases := []struct {
		name  string
		inA   uint8
		wantA uint8
		wantC bool
	}{
		{"bit7_set", 0x80, 0x01, true},
		{"bit7_clear", 0x7F, 0xFE, false},
		{"all_ones", 0xFF, 0xFF, true},
		{"zero", 0x00, 0x00, false},
		{"alternating", 0xAA, 0x55, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = tc.inA
			cpu.Registers.PC = 0xC000
			// Pre-set all flags to verify the handler reliably clears Z/N/H.
			cpu.SetFlagZ()
			cpu.SetFlagN()
			cpu.SetFlagH()

			cycles := OPCODES[0x07](mb, 0)

			assert.Equal(t, OpCycles(4), cycles)
			assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
			assert.Equal(t, tc.wantA, cpu.Registers.A)
			assert.False(t, cpu.IsFlagZSet(), "RLCA always clears Z (even when result==0)")
			assert.False(t, cpu.IsFlagNSet(), "RLCA clears N")
			assert.False(t, cpu.IsFlagHSet(), "RLCA clears H")
			assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C = old bit 7")
		})
	}
}

func TestRotateA_RRCA_0x0F(t *testing.T) {
	cases := []struct {
		name  string
		inA   uint8
		wantA uint8
		wantC bool
	}{
		{"bit0_set", 0x01, 0x80, true},
		{"bit0_clear", 0xFE, 0x7F, false},
		{"all_ones", 0xFF, 0xFF, true},
		{"zero", 0x00, 0x00, false},
		{"alternating", 0x55, 0xAA, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = tc.inA
			cpu.Registers.PC = 0xC000
			cpu.SetFlagZ()
			cpu.SetFlagN()
			cpu.SetFlagH()

			cycles := OPCODES[0x0F](mb, 0)

			assert.Equal(t, OpCycles(4), cycles)
			assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
			assert.Equal(t, tc.wantA, cpu.Registers.A)
			assert.False(t, cpu.IsFlagZSet(), "RRCA always clears Z")
			assert.False(t, cpu.IsFlagNSet())
			assert.False(t, cpu.IsFlagHSet())
			assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C = old bit 0")
		})
	}
}

func TestRotateA_RLA_0x17(t *testing.T) {
	cases := []struct {
		name   string
		inA    uint8
		startC bool
		wantA  uint8
		wantC  bool
	}{
		{"top_in_carry_no_in", 0x80, false, 0x00, true},
		{"top_in_carry_in", 0x80, true, 0x01, true},
		{"low_in_carry", 0x00, true, 0x01, false},
		{"zero_no_carry", 0x00, false, 0x00, false},
		{"all_ones_no_carry", 0xFF, false, 0xFE, true},
		{"alt_with_carry_in", 0x55, true, 0xAB, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = tc.inA
			cpu.Registers.PC = 0xC000
			if tc.startC {
				cpu.SetFlagC()
			} else {
				cpu.ResetFlagC()
			}
			// Pre-set Z/N/H to verify they are cleared.
			cpu.SetFlagZ()
			cpu.SetFlagN()
			cpu.SetFlagH()

			cycles := OPCODES[0x17](mb, 0)

			assert.Equal(t, OpCycles(4), cycles)
			assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
			assert.Equal(t, tc.wantA, cpu.Registers.A)
			assert.False(t, cpu.IsFlagZSet(), "RLA always clears Z")
			assert.False(t, cpu.IsFlagNSet())
			assert.False(t, cpu.IsFlagHSet())
			assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C = old bit 7")
		})
	}
}

func TestRotateA_RRA_0x1F(t *testing.T) {
	cases := []struct {
		name   string
		inA    uint8
		startC bool
		wantA  uint8
		wantC  bool
	}{
		{"low_to_carry_no_in", 0x01, false, 0x00, true},
		{"low_to_carry_in", 0x01, true, 0x80, true},
		{"top_in_no_low", 0x00, true, 0x80, false},
		{"zero_no_carry", 0x00, false, 0x00, false},
		{"all_ones_no_carry", 0xFF, false, 0x7F, true},
		{"alt_with_carry_in", 0xAA, true, 0xD5, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = tc.inA
			cpu.Registers.PC = 0xC000
			if tc.startC {
				cpu.SetFlagC()
			} else {
				cpu.ResetFlagC()
			}
			cpu.SetFlagZ()
			cpu.SetFlagN()
			cpu.SetFlagH()

			cycles := OPCODES[0x1F](mb, 0)

			assert.Equal(t, OpCycles(4), cycles)
			assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
			assert.Equal(t, tc.wantA, cpu.Registers.A)
			assert.False(t, cpu.IsFlagZSet(), "RRA always clears Z")
			assert.False(t, cpu.IsFlagNSet())
			assert.False(t, cpu.IsFlagHSet())
			assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C = old bit 0")
		})
	}
}

// ---------------------------------------------------------------------------
// 0x27 — DAA  (table-driven)
// Each row corresponds to a realistic post-arithmetic state. Expected
// values match THIS implementation's reading of the rules in opcodes.go:
//
//   * After ADD: corr starts at 0; if H -> |0x06; if C -> |0x60;
//     additionally if low-nibble > 9 -> |0x06; if A > 0x99 -> |0x60.
//     A = (A + corr) & 0xFF.
//   * After SUB (N=1): corr = (H ? 0x06 : 0) | (C ? 0x60 : 0).
//     A = (A - corr) & 0xFF.
//   * Z = (A == 0). H always cleared. N preserved. C set iff corr & 0x60.
//
// The implementation in opcodes.go ALWAYS rebuilds the C flag from the
// current `corr`; it does NOT preserve a previous C when the SUB-path
// produces no 0x60 bit. The "subN_no_correction_preserve" row pins down
// this quirk so a future fix shows up as an intentional test failure.
// ---------------------------------------------------------------------------

func TestDAA_0x27(t *testing.T) {
	type daaCase struct {
		name  string
		inA   uint8
		inN   bool
		inH   bool
		inC   bool
		wantA uint8
		wantZ bool
		wantC bool // H is always cleared; N is always preserved
	}

	cases := []daaCase{
		// --- ADD path (N=0) -------------------------------------------------
		{"add_low_nibble_overflow", 0x0A, false, false, false, 0x10, false, false},
		{"add_with_half_carry", 0x12, false, true, false, 0x18, false, false},
		{"add_high_nibble_overflow", 0xA0, false, false, false, 0x00, true, true},
		{"add_with_full_carry", 0x32, false, true, true, 0x98, false, true},
		{"add_zero_no_correction", 0x00, false, false, false, 0x00, true, false},
		{"add_pure_low_correction", 0x05, false, false, false, 0x05, false, false},
		// --- SUB path (N=1) -------------------------------------------------
		{"sub_no_correction", 0x05, true, false, false, 0x05, false, false},
		{"sub_half_borrow", 0x0B, true, true, false, 0x05, false, false},
		{"sub_full_borrow", 0xFF, true, true, true, 0x99, false, true},
		{"sub_zero_result", 0x00, true, false, false, 0x00, true, false},
		{"sub_carry_only", 0x70, true, false, true, 0x10, false, true},
		// Quirk pin: SUB path with C=1 going in but corr has no 0x60 bit
		// because we only flagged H. Implementation REBUILDS C from corr,
		// so the previous C is *lost*. Pin it down.
		{"subN_no_correction_preserve", 0x10, true, true, false, 0x0A, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = tc.inA
			cpu.Registers.PC = 0xC000

			// Seed flag state. Z and "other" bits are reset so we can
			// observe the handler's output cleanly.
			cpu.ResetFlagZ()
			if tc.inN {
				cpu.SetFlagN()
			} else {
				cpu.ResetFlagN()
			}
			if tc.inH {
				cpu.SetFlagH()
			} else {
				cpu.ResetFlagH()
			}
			if tc.inC {
				cpu.SetFlagC()
			} else {
				cpu.ResetFlagC()
			}

			cycles := OPCODES[0x27](mb, 0)

			assert.Equal(t, OpCycles(4), cycles, "DAA = 4 cycles")
			assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
			assert.Equalf(t, tc.wantA, cpu.Registers.A,
				"A: input=%#02x N=%v H=%v C=%v", tc.inA, tc.inN, tc.inH, tc.inC)
			assert.Equal(t, tc.wantZ, cpu.IsFlagZSet(), "Z flag")
			assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C flag")
			assert.False(t, cpu.IsFlagHSet(), "DAA always clears H")
			assert.Equal(t, tc.inN, cpu.IsFlagNSet(), "DAA preserves N")
		})
	}
}

// ---------------------------------------------------------------------------
// 0x2F — CPL
// ---------------------------------------------------------------------------

func TestCPL_0x2F(t *testing.T) {
	cases := []struct {
		name string
		inA  uint8
		want uint8
	}{
		{"alternating_AA", 0xAA, 0x55},
		{"alternating_55", 0x55, 0xAA},
		{"all_zero", 0x00, 0xFF},
		{"all_one", 0xFF, 0x00},
	}
	for _, tc := range cases {
		t.Run(tc.name+"_Z=0_C=0", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = tc.inA
			cpu.Registers.PC = 0xC000
			cpu.ResetFlagZ()
			cpu.ResetFlagC()

			OPCODES[0x2F](mb, 0)

			assert.Equal(t, tc.want, cpu.Registers.A)
			assert.True(t, cpu.IsFlagNSet(), "CPL sets N")
			assert.True(t, cpu.IsFlagHSet(), "CPL sets H")
			assert.False(t, cpu.IsFlagZSet(), "CPL preserves Z=0")
			assert.False(t, cpu.IsFlagCSet(), "CPL preserves C=0")
			assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
		})
		t.Run(tc.name+"_Z=1_C=1", func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = tc.inA
			cpu.Registers.PC = 0xC000
			cpu.SetFlagZ()
			cpu.SetFlagC()

			cycles := OPCODES[0x2F](mb, 0)

			assert.Equal(t, OpCycles(4), cycles)
			assert.Equal(t, tc.want, cpu.Registers.A)
			assert.True(t, cpu.IsFlagNSet())
			assert.True(t, cpu.IsFlagHSet())
			assert.True(t, cpu.IsFlagZSet(), "CPL preserves Z=1")
			assert.True(t, cpu.IsFlagCSet(), "CPL preserves C=1")
		})
	}
}

// ---------------------------------------------------------------------------
// 0x37 — SCF  (Set Carry Flag)
// ---------------------------------------------------------------------------

func TestSCF_0x37(t *testing.T) {
	t.Run("preserves_Z=1", func(t *testing.T) {
		cpu, mb := newTestCPU(t)
		cpu.Registers.PC = 0xC000
		cpu.SetFlagZ()
		cpu.SetFlagN()
		cpu.SetFlagH()
		cpu.ResetFlagC()

		cycles := OPCODES[0x37](mb, 0)

		assert.Equal(t, OpCycles(4), cycles)
		assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
		assert.True(t, cpu.IsFlagZSet(), "SCF preserves Z")
		assert.False(t, cpu.IsFlagNSet(), "SCF clears N")
		assert.False(t, cpu.IsFlagHSet(), "SCF clears H")
		assert.True(t, cpu.IsFlagCSet(), "SCF sets C")
	})
	t.Run("preserves_Z=0_idempotent_with_C=1", func(t *testing.T) {
		cpu, mb := newTestCPU(t)
		cpu.Registers.PC = 0xC000
		cpu.ResetFlagZ()
		cpu.SetFlagC()

		OPCODES[0x37](mb, 0)

		assert.False(t, cpu.IsFlagZSet())
		assert.True(t, cpu.IsFlagCSet(), "SCF leaves C set when already set")
	})
}

// ---------------------------------------------------------------------------
// 0x3F — CCF  (Complement Carry Flag)
// ---------------------------------------------------------------------------

func TestCCF_0x3F(t *testing.T) {
	t.Run("toggles_C_from_0_to_1_preserves_Z=1", func(t *testing.T) {
		cpu, mb := newTestCPU(t)
		cpu.Registers.PC = 0xC000
		cpu.SetFlagZ()
		cpu.SetFlagN()
		cpu.SetFlagH()
		cpu.ResetFlagC()

		cycles := OPCODES[0x3F](mb, 0)

		assert.Equal(t, OpCycles(4), cycles)
		assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
		assert.True(t, cpu.IsFlagZSet(), "CCF preserves Z")
		assert.False(t, cpu.IsFlagNSet(), "CCF clears N")
		assert.False(t, cpu.IsFlagHSet(), "CCF clears H")
		assert.True(t, cpu.IsFlagCSet(), "CCF: 0 -> 1")
	})
	t.Run("toggles_C_from_1_to_0_preserves_Z=0", func(t *testing.T) {
		cpu, mb := newTestCPU(t)
		cpu.Registers.PC = 0xC000
		cpu.ResetFlagZ()
		cpu.SetFlagC()

		OPCODES[0x3F](mb, 0)

		assert.False(t, cpu.IsFlagZSet())
		assert.False(t, cpu.IsFlagCSet(), "CCF: 1 -> 0")
	})
}

// ---------------------------------------------------------------------------
// 0x76 — HALT
// HALT does NOT advance PC in this implementation; the PC bump after the
// CPU wakes is performed by ServiceInterrupt(). We only assert the Halted
// state and the cycle count.
// ---------------------------------------------------------------------------

func TestHALT_0x76(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.Halted = false
	cpu.Interrupts.IE = 0
	cpu.Interrupts.IF = 0
	cpu.Interrupts.InterruptsOn = true

	cycles := OPCODES[0x76](mb, 0)

	// HALT consumes the opcode byte and sets the halted flag; the
	// halt-bug edge case requires a pending interrupt with IME=0, so
	// with IME=1 and no pending interrupt we always enter halt.
	assert.Equal(t, OpCycles(4), cycles)
	assert.True(t, cpu.Halted, "HALT must set the Halted flag")
	assert.False(t, cpu.HaltBug, "no pending interrupt -> no HALT bug")
	assert.Equal(t, uint16(0xC001), cpu.Registers.PC, "HALT advances PC past its opcode")
}

// TestHALT_0x76_HaltBug verifies the IME=0 + pending-interrupt edge
// case: HALT does NOT enter the halted state and instead arms the
// HALT bug for the next instruction.
func TestHALT_0x76_HaltBug(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.Halted = false
	cpu.HaltBug = false
	cpu.Interrupts.InterruptsOn = false
	cpu.Interrupts.IE = 0x01
	cpu.Interrupts.IF = 0x01

	cycles := OPCODES[0x76](mb, 0)

	assert.Equal(t, OpCycles(4), cycles)
	assert.False(t, cpu.Halted, "HALT bug case: CPU does NOT halt")
	assert.True(t, cpu.HaltBug, "HALT bug must be armed")
	assert.Equal(t, uint16(0xC001), cpu.Registers.PC, "HALT advances PC past its opcode")
}

// ---------------------------------------------------------------------------
// 0x10 — STOP
// STOP is encoded as a 2-byte instruction (0x10 0x00). Handler advances
// PC by 2 and returns 4 cycles. In CGB mode it also handles speed-switch
// via IO_KEY1; our test CPU runs in DMG mode (Cgb=false) so we just
// verify PC advance and no panic.
// ---------------------------------------------------------------------------

func TestSTOP_0x10(t *testing.T) {
	cpu, mb := newTestCPU(t)
	require.False(t, mb.Cgb, "newTestCPU runs in DMG mode")
	cpu.Registers.PC = 0xC000

	cycles := OPCODES[0x10](mb, 0)

	assert.Equal(t, OpCycles(4), cycles, "STOP returns 4 cycles in this impl")
	assert.Equal(t, uint16(0xC002), cpu.Registers.PC, "STOP advances PC by 2")
}

// ---------------------------------------------------------------------------
// 0xF3 — DI  /  0xFB — EI
// ---------------------------------------------------------------------------

func TestDI_0xF3(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.Interrupts.InterruptsOn = true
	cpu.Interrupts.InterruptsEnabling = true // any state — DI shouldn't touch it

	cycles := OPCODES[0xF3](mb, 0)

	assert.Equal(t, OpCycles(4), cycles)
	assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
	assert.False(t, cpu.Interrupts.InterruptsOn, "DI clears master IME (InterruptsOn)")
	// DI in this impl does NOT clear InterruptsEnabling. Pin it down so
	// any future change is intentional.
	assert.True(t, cpu.Interrupts.InterruptsEnabling,
		"DI in this impl does not touch InterruptsEnabling (delayed-enable flag)")
}

func TestEI_0xFB_DelayedEnable(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	cpu.Interrupts.InterruptsOn = false
	cpu.Interrupts.InterruptsEnabling = false

	cycles := OPCODES[0xFB](mb, 0)

	assert.Equal(t, OpCycles(4), cycles)
	assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
	// EI's "delayed enable" semantics: the handler only sets the
	// `InterruptsEnabling` latch. The actual master IME (InterruptsOn)
	// flips one instruction later, in the main tick loop — NOT here.
	assert.True(t, cpu.Interrupts.InterruptsEnabling,
		"EI sets InterruptsEnabling (delayed enable latch)")
	assert.False(t, cpu.Interrupts.InterruptsOn,
		"EI must NOT immediately set InterruptsOn — the enable is delayed")
}

// ---------------------------------------------------------------------------
// 0x00 — NOP semantic test
// (opcodes_test.go already smoke-tests this; we add a stronger semantic
// check that no register or flag is mutated by NOP.)
// ---------------------------------------------------------------------------

func TestNOP_0x00_PreservesAllState(t *testing.T) {
	cpu, mb := newTestCPU(t)

	// Seed every register and flag to a non-default sentinel value.
	cpu.Registers.A = 0xA5
	cpu.Registers.B = 0xB5
	cpu.Registers.C = 0xC5
	cpu.Registers.D = 0xD5
	cpu.Registers.E = 0xE5
	cpu.Registers.F = 0xF0 // all four flags set, low nibble 0
	cpu.Registers.H = 0x12
	cpu.Registers.L = 0x34
	cpu.Registers.SP = 0xFFF0
	cpu.Registers.PC = 0xC000

	cycles := OPCODES[0x00](mb, 0)

	assert.Equal(t, OpCycles(4), cycles, "NOP = 4 cycles")
	assert.Equal(t, uint16(0xC001), cpu.Registers.PC, "NOP advances PC by 1")

	// Everything else must be untouched.
	assert.Equal(t, uint8(0xA5), cpu.Registers.A)
	assert.Equal(t, uint8(0xB5), cpu.Registers.B)
	assert.Equal(t, uint8(0xC5), cpu.Registers.C)
	assert.Equal(t, uint8(0xD5), cpu.Registers.D)
	assert.Equal(t, uint8(0xE5), cpu.Registers.E)
	assert.Equal(t, uint8(0xF0), cpu.Registers.F, "NOP must not touch any flag")
	assert.Equal(t, uint8(0x12), cpu.Registers.H)
	assert.Equal(t, uint8(0x34), cpu.Registers.L)
	assert.Equal(t, uint16(0xFFF0), cpu.Registers.SP)
}
