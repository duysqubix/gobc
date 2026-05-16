// Coverage map for the opcodes exercised in this file (kept as a quick
// reference because the dispatcher table is keyed by opaque hex literals
// and there is no other place a reviewer can confirm completeness):
//
//	0x01 0x11 0x21 0x31  LD rr, d16
//	0x08                 LD (a16), SP
//	0xF8                 LD HL, SP+r8
//	0xF9                 LD SP, HL
//	0xC5 0xD5 0xE5 0xF5  PUSH BC/DE/HL/AF
//	0xC1 0xD1 0xE1 0xF1  POP  BC/DE/HL/AF
//	0x09 0x19 0x29 0x39  ADD HL, BC/DE/HL/SP
//	0xE8                 ADD SP, r8
//	0x03 0x13 0x23 0x33  INC BC/DE/HL/SP
//	0x0B 0x1B 0x2B 0x3B  DEC BC/DE/HL/SP
package motherboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// LD rr, d16  (0x01, 0x11, 0x21, 0x31)
// ---------------------------------------------------------------------------

func TestLD16_Immediate(t *testing.T) {
	cases := []struct {
		name   string
		opcode OpCode
		value  uint16
		// Verify the destination via the appropriate accessor.
		read func(*CPU) uint16
	}{
		{"LD_BC_d16_0x01", 0x01, 0x1234, func(c *CPU) uint16 { return c.BC() }},
		{"LD_DE_d16_0x11", 0x11, 0xBEEF, func(c *CPU) uint16 { return c.DE() }},
		{"LD_HL_d16_0x21", 0x21, 0xC0DE, func(c *CPU) uint16 { return c.HL() }},
		{"LD_SP_d16_0x31", 0x31, 0xFFF0, func(c *CPU) uint16 { return c.Registers.SP }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			startPC := cpu.Registers.PC
			startF := cpu.Registers.F

			cycles := OPCODES[tc.opcode](mb, tc.value)

			assert.Equal(t, OpCycles(12), cycles, "%s cycles", tc.name)
			assert.Equal(t, startPC+3, cpu.Registers.PC, "%s PC advance", tc.name)
			assert.Equal(t, tc.value, tc.read(cpu), "%s loaded value", tc.name)
			assert.Equal(t, startF, cpu.Registers.F, "%s flags untouched", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// LD (a16), SP  (0x08)
// ---------------------------------------------------------------------------

func TestLD16_StoreSP_at_a16(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.SP = 0xBEEF
	startPC := cpu.Registers.PC
	startF := cpu.Registers.F
	const addr uint16 = 0xC100 // WRAM bank 0, comfortably writable

	cycles := OPCODES[0x08](mb, addr)

	assert.Equal(t, OpCycles(20), cycles, "LD (a16),SP cycles")
	assert.Equal(t, startPC+3, cpu.Registers.PC, "LD (a16),SP PC advance")
	// SP is split little-endian: low byte first, high byte at addr+1.
	assert.Equal(t, uint8(0xEF), mb.GetItem(addr), "low byte at addr")
	assert.Equal(t, uint8(0xBE), mb.GetItem(addr+1), "high byte at addr+1")
	assert.Equal(t, uint16(0xBEEF), cpu.Registers.SP, "SP unchanged by store")
	assert.Equal(t, startF, cpu.Registers.F, "LD (a16),SP flags untouched")
}

// ---------------------------------------------------------------------------
// LD HL, SP+r8  (0xF8)
//
// Flag math (per impl + Pan Docs):
//   Z = 0
//   N = 0
//   H = 1 iff (SP & 0x0F) + (r8 & 0x0F) > 0x0F
//   C = 1 iff (SP & 0xFF) + (r8 & 0xFF) > 0xFF   (r8 treated as unsigned byte)
// ---------------------------------------------------------------------------

func TestLD16_HL_SP_plus_r8(t *testing.T) {
	cases := []struct {
		name   string
		sp     uint16
		r8     uint16 // raw byte the dispatcher would feed in (0..0xFF)
		wantHL uint16
		wantH  bool
		wantC  bool
	}{
		{
			name: "positive_no_carries",
			sp:   0x1000, r8: 0x10,
			wantHL: 0x1010, wantH: false, wantC: false,
		},
		{
			name: "positive_half_carry_only",
			sp:   0x0008, r8: 0x08,
			wantHL: 0x0010, wantH: true, wantC: false,
		},
		{
			name: "positive_full_carry_only",
			sp:   0x00F0, r8: 0x70,
			wantHL: 0x0160, wantH: false, wantC: true,
		},
		{
			name: "positive_both_half_and_full_carry",
			sp:   0x00FF, r8: 0x01,
			wantHL: 0x0100, wantH: true, wantC: true,
		},
		{
			name: "zero_offset_still_clears_Z_and_N",
			sp:   0x1234, r8: 0x00,
			wantHL: 0x1234, wantH: false, wantC: false,
		},
		{
			name: "negative_minus_one_with_both_carries",
			sp:   0x0001, r8: 0xFF, // 0xFF == -1
			wantHL: 0x0000, wantH: true, wantC: true,
		},
		{
			name: "negative_minus_one_no_carries",
			sp:   0x1000, r8: 0xFF, // -1, low byte 0x00 + 0xFF = 0xFF, no overflow
			wantHL: 0x0FFF, wantH: false, wantC: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.SP = tc.sp
			// Pre-set Z and N to 1 so we can prove the opcode clears them.
			cpu.SetFlagZ()
			cpu.SetFlagN()
			startPC := cpu.Registers.PC

			cycles := OPCODES[0xF8](mb, tc.r8)

			assert.Equal(t, OpCycles(12), cycles, "cycles")
			assert.Equal(t, startPC+2, cpu.Registers.PC, "PC advance")
			assert.Equal(t, tc.sp, cpu.Registers.SP, "SP unchanged")
			assert.Equal(t, tc.wantHL, cpu.HL(), "HL = SP + r8 (signed)")
			assert.False(t, cpu.IsFlagZSet(), "Z must be cleared")
			assert.False(t, cpu.IsFlagNSet(), "N must be cleared")
			assert.Equal(t, tc.wantH, cpu.IsFlagHSet(), "H flag")
			assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C flag")
		})
	}
}

// ---------------------------------------------------------------------------
// LD SP, HL  (0xF9)
// ---------------------------------------------------------------------------

func TestLD16_SP_HL(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.SetHL(0xCAFE)
	cpu.Registers.SP = 0x0000
	startPC := cpu.Registers.PC
	startF := cpu.Registers.F

	cycles := OPCODES[0xF9](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD SP,HL cycles")
	assert.Equal(t, startPC+1, cpu.Registers.PC, "LD SP,HL PC advance")
	assert.Equal(t, uint16(0xCAFE), cpu.Registers.SP, "SP gets HL")
	assert.Equal(t, uint16(0xCAFE), cpu.HL(), "HL preserved")
	assert.Equal(t, startF, cpu.Registers.F, "LD SP,HL flags untouched")
}

// ---------------------------------------------------------------------------
// PUSH rr  (0xC5, 0xD5, 0xE5, 0xF5)
// ---------------------------------------------------------------------------

func TestPushPop_Push(t *testing.T) {
	cases := []struct {
		name      string
		opcode    OpCode
		setupPair func(*CPU) // load the source pair
		wantHigh  uint8      // high byte expected at SP-1
		wantLow   uint8      // low byte  expected at SP-2
	}{
		{
			name: "PUSH_BC_0xC5", opcode: 0xC5,
			setupPair: func(c *CPU) { c.SetBC(0xABCD) },
			wantHigh:  0xAB, wantLow: 0xCD,
		},
		{
			name: "PUSH_DE_0xD5", opcode: 0xD5,
			setupPair: func(c *CPU) { c.SetDE(0x1234) },
			wantHigh:  0x12, wantLow: 0x34,
		},
		{
			name: "PUSH_HL_0xE5", opcode: 0xE5,
			setupPair: func(c *CPU) { c.SetHL(0xBEEF) },
			wantHigh:  0xBE, wantLow: 0xEF,
		},
		{
			// AF: low byte is F. With F = 0xF0 the lower nibble is already
			// zero so we can compare to the fixed expected byte. The
			// dedicated nibble-quirk test below asserts the post-instruction
			// memory image when F has its lower nibble set.
			name: "PUSH_AF_0xF5", opcode: 0xF5,
			setupPair: func(c *CPU) { c.Registers.A = 0xCA; c.Registers.F = 0xF0 },
			wantHigh:  0xCA, wantLow: 0xF0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			tc.setupPair(cpu)
			startPC := cpu.Registers.PC
			startSP := cpu.Registers.SP // 0xFFFE per harness

			cycles := OPCODES[tc.opcode](mb, 0)

			assert.Equal(t, OpCycles(16), cycles, "%s cycles", tc.name)
			assert.Equal(t, startPC+1, cpu.Registers.PC, "%s PC advance", tc.name)
			assert.Equal(t, startSP-2, cpu.Registers.SP, "%s SP -= 2", tc.name)
			assert.Equal(t, tc.wantHigh, mb.GetItem(startSP-1), "%s high byte at SP-1", tc.name)
			assert.Equal(t, tc.wantLow, mb.GetItem(startSP-2), "%s low byte at SP-2", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// POP rr  (0xC1, 0xD1, 0xE1, 0xF1)
// ---------------------------------------------------------------------------

func TestPushPop_Pop(t *testing.T) {
	cases := []struct {
		name     string
		opcode   OpCode
		highByte uint8 // memory at SP+1 (becomes high register)
		lowByte  uint8 // memory at SP   (becomes low  register)
		want     uint16
		readPair func(*CPU) uint16
		// AF has a hardware quirk: the low nibble of F is wired to ground,
		// so POP AF must mask it on read. Set this to true to assert the
		// mask AND verify F's lower nibble is zero.
		isAF bool
	}{
		{
			name: "POP_BC_0xC1", opcode: 0xC1,
			highByte: 0xAB, lowByte: 0xCD, want: 0xABCD,
			readPair: func(c *CPU) uint16 { return c.BC() },
		},
		{
			name: "POP_DE_0xD1", opcode: 0xD1,
			highByte: 0x12, lowByte: 0x34, want: 0x1234,
			readPair: func(c *CPU) uint16 { return c.DE() },
		},
		{
			name: "POP_HL_0xE1", opcode: 0xE1,
			highByte: 0xBE, lowByte: 0xEF, want: 0xBEEF,
			readPair: func(c *CPU) uint16 { return c.HL() },
		},
		{
			// 0xFF in the low byte exercises the F-nibble masking quirk:
			// the in-memory lower nibble (0xF) must NOT survive into F.
			name: "POP_AF_0xF1_masks_low_nibble", opcode: 0xF1,
			highByte: 0xCA, lowByte: 0xFF, want: 0xCAF0,
			readPair: func(c *CPU) uint16 {
				return (uint16(c.Registers.A) << 8) | uint16(c.Registers.F)
			},
			isAF: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			startPC := cpu.Registers.PC
			startSP := cpu.Registers.SP
			// Plant the bytes the POP will consume. SetItem takes uint16
			// (it asserts value < 0x100), so widen explicitly.
			mb.SetItem(startSP, uint16(tc.lowByte))
			mb.SetItem(startSP+1, uint16(tc.highByte))

			cycles := OPCODES[tc.opcode](mb, 0)

			assert.Equal(t, OpCycles(12), cycles, "%s cycles", tc.name)
			assert.Equal(t, startPC+1, cpu.Registers.PC, "%s PC advance", tc.name)
			assert.Equal(t, startSP+2, cpu.Registers.SP, "%s SP += 2", tc.name)
			assert.Equal(t, tc.want, tc.readPair(cpu), "%s register pair", tc.name)
			if tc.isAF {
				assert.Equal(t, uint8(0), cpu.Registers.F&0x0F,
					"POP AF must clear low nibble of F (Game Boy quirk)")
			}
		})
	}
}

// PUSH AF / POP AF round-trip: the byte that PUSH writes to the stack
// preserves whatever was in the F register at the moment of the push
// (nothing in PUSH AF masks the lower nibble), but POP AF masks the lower
// nibble of F to zero on the way back into the register file. This
// round-trip pins both halves of that contract.
func TestPushPop_AF_NibblePreservation(t *testing.T) {
	cpu, mb := newTestCPU(t)

	cpu.Registers.A = 0xAB
	cpu.Registers.F = 0xFF // every flag bit + every garbage bit set
	startSP := cpu.Registers.SP

	// PUSH AF
	cycles := OPCODES[0xF5](mb, 0)
	require.Equal(t, OpCycles(16), cycles, "PUSH AF cycles")
	require.Equal(t, startSP-2, cpu.Registers.SP, "PUSH AF moves SP by 2")

	// The byte at the new SP is the post-instruction memory image of F.
	// PUSH AF writes F as-is, so 0xFF survives unchanged in memory.
	assert.Equal(t, uint8(0xFF), mb.GetItem(cpu.Registers.SP),
		"PUSH AF writes the raw F byte (low nibble preserved in stack image)")
	assert.Equal(t, uint8(0xAB), mb.GetItem(cpu.Registers.SP+1),
		"PUSH AF writes the raw A byte at SP+1")

	// Now overwrite the registers so we can prove POP repopulates them.
	cpu.Registers.A = 0
	cpu.Registers.F = 0

	// POP AF
	cycles = OPCODES[0xF1](mb, 0)
	require.Equal(t, OpCycles(12), cycles, "POP AF cycles")
	require.Equal(t, startSP, cpu.Registers.SP, "POP AF restores SP")

	assert.Equal(t, uint8(0xAB), cpu.Registers.A, "POP AF restores A")
	assert.Equal(t, uint8(0xF0), cpu.Registers.F,
		"POP AF masks F low nibble to zero (Game Boy quirk)")
	assert.Equal(t, uint8(0), cpu.Registers.F&0x0F,
		"low nibble of F is forced to zero on POP")
}

// ---------------------------------------------------------------------------
// ADD HL, rr  (0x09, 0x19, 0x29, 0x39)
//
// Flag math (impl: cpu.AddSetFlags16):
//   Z UNCHANGED
//   N = 0
//   H = 1 iff carry out of bit 11 (i.e. bit 12 of (a^b^sum))
//   C = 1 iff carry out of bit 15 (i.e. bit 16 of (a+b))
// ---------------------------------------------------------------------------

func TestADDHL_Pairs(t *testing.T) {
	cases := []struct {
		name   string
		opcode OpCode
		// Setup: the test loads HL and the source pair to specific values.
		hl     uint16
		src    uint16
		setSrc func(*CPU, uint16)
		// Expectations.
		wantHL uint16
		wantH  bool
		wantC  bool
	}{
		{
			name: "ADD_HL_BC_no_carries_0x09", opcode: 0x09,
			hl: 0x1000, src: 0x0FFF,
			setSrc: func(c *CPU, v uint16) { c.SetBC(v) },
			wantHL: 0x1FFF, wantH: false, wantC: false,
		},
		{
			name: "ADD_HL_DE_half_carry_only_0x19", opcode: 0x19,
			hl: 0x0FFF, src: 0x0001,
			setSrc: func(c *CPU, v uint16) { c.SetDE(v) },
			wantHL: 0x1000, wantH: true, wantC: false,
		},
		{
			// HL+HL self-add. With HL=0x8000 the result wraps to 0 with
			// carry out of bit 15 but no carry out of bit 11 (low 12 bits
			// are zero on both sides).
			name: "ADD_HL_HL_carry_only_0x29", opcode: 0x29,
			hl: 0x8000, src: 0x8000, // src is ignored -- HL+HL uses HL twice
			setSrc: func(c *CPU, v uint16) { /* no-op */ },
			wantHL: 0x0000, wantH: false, wantC: true,
		},
		{
			name: "ADD_HL_SP_both_carries_0x39", opcode: 0x39,
			hl: 0xFFFF, src: 0x0001,
			setSrc: func(c *CPU, v uint16) { c.Registers.SP = v },
			wantHL: 0x0000, wantH: true, wantC: true,
		},
	}

	// Run each row twice: once with Z preset to 1, once with Z preset to 0.
	// ADD HL,rr must leave Z untouched.
	for _, tc := range cases {
		for _, presetZ := range []bool{true, false} {
			label := tc.name
			if presetZ {
				label += "_Zpreset_1"
			} else {
				label += "_Zpreset_0"
			}
			t.Run(label, func(t *testing.T) {
				cpu, mb := newTestCPU(t)
				cpu.SetHL(tc.hl)
				tc.setSrc(cpu, tc.src)
				if presetZ {
					cpu.SetFlagZ()
				} else {
					cpu.ResetFlagZ()
				}
				// Pre-set N to 1 so we can prove the opcode clears it.
				cpu.SetFlagN()
				startPC := cpu.Registers.PC

				cycles := OPCODES[tc.opcode](mb, 0)

				assert.Equal(t, OpCycles(8), cycles, "cycles")
				assert.Equal(t, startPC+1, cpu.Registers.PC, "PC advance")
				assert.Equal(t, tc.wantHL, cpu.HL(), "HL result")
				assert.Equal(t, presetZ, cpu.IsFlagZSet(), "Z must be unchanged")
				assert.False(t, cpu.IsFlagNSet(), "N must be cleared")
				assert.Equal(t, tc.wantH, cpu.IsFlagHSet(), "H flag")
				assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C flag")
			})
		}
	}
}

// ---------------------------------------------------------------------------
// ADD SP, r8  (0xE8)
//
// Same low-byte flag math as LD HL,SP+r8:
//   Z = 0, N = 0
//   H = 1 iff (SP & 0x0F) + (r8 & 0x0F) > 0x0F
//   C = 1 iff (SP & 0xFF) + (r8 & 0xFF) > 0xFF  (r8 unsigned for flag math)
// SP is mutated; HL is not.
// ---------------------------------------------------------------------------

func TestADDSP_r8(t *testing.T) {
	cases := []struct {
		name   string
		sp     uint16
		r8     uint16 // raw byte
		wantSP uint16
		wantH  bool
		wantC  bool
	}{
		{
			name: "positive_no_carries",
			sp:   0x1000, r8: 0x10,
			wantSP: 0x1010, wantH: false, wantC: false,
		},
		{
			name: "positive_half_carry_only",
			sp:   0x0008, r8: 0x08,
			wantSP: 0x0010, wantH: true, wantC: false,
		},
		{
			name: "positive_full_carry_only",
			sp:   0x00F0, r8: 0x70,
			wantSP: 0x0160, wantH: false, wantC: true,
		},
		{
			name: "positive_both_carries",
			sp:   0x00FF, r8: 0x01,
			wantSP: 0x0100, wantH: true, wantC: true,
		},
		{
			name: "zero_offset_still_clears_Z_and_N",
			sp:   0x1234, r8: 0x00,
			wantSP: 0x1234, wantH: false, wantC: false,
		},
		{
			name: "negative_minus_one_both_carries",
			sp:   0x0001, r8: 0xFF,
			wantSP: 0x0000, wantH: true, wantC: true,
		},
		{
			name: "negative_minus_one_no_carries",
			sp:   0x1000, r8: 0xFF,
			wantSP: 0x0FFF, wantH: false, wantC: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.SP = tc.sp
			cpu.SetHL(0xDEAD) // must remain unchanged
			// Pre-set Z and N to 1 to prove the opcode clears them.
			cpu.SetFlagZ()
			cpu.SetFlagN()
			startPC := cpu.Registers.PC

			cycles := OPCODES[0xE8](mb, tc.r8)

			assert.Equal(t, OpCycles(16), cycles, "cycles")
			assert.Equal(t, startPC+2, cpu.Registers.PC, "PC advance")
			assert.Equal(t, tc.wantSP, cpu.Registers.SP, "SP = SP + r8 (signed)")
			assert.Equal(t, uint16(0xDEAD), cpu.HL(), "HL untouched")
			assert.False(t, cpu.IsFlagZSet(), "Z must be cleared")
			assert.False(t, cpu.IsFlagNSet(), "N must be cleared")
			assert.Equal(t, tc.wantH, cpu.IsFlagHSet(), "H flag")
			assert.Equal(t, tc.wantC, cpu.IsFlagCSet(), "C flag")
		})
	}
}

// ---------------------------------------------------------------------------
// INC rr  (0x03, 0x13, 0x23, 0x33)  -- NO flag changes
// ---------------------------------------------------------------------------

func TestIncDec16_INC(t *testing.T) {
	cases := []struct {
		name   string
		opcode OpCode
		setup  func(*CPU)
		read   func(*CPU) uint16
	}{
		{
			name: "INC_BC_0x03", opcode: 0x03,
			setup: func(c *CPU) { c.SetBC(0x12FF) },
			read:  func(c *CPU) uint16 { return c.BC() },
		},
		{
			name: "INC_DE_0x13", opcode: 0x13,
			setup: func(c *CPU) { c.SetDE(0x00FF) },
			read:  func(c *CPU) uint16 { return c.DE() },
		},
		{
			name: "INC_HL_0x23", opcode: 0x23,
			setup: func(c *CPU) { c.SetHL(0xFFFF) }, // wrap to 0x0000
			read:  func(c *CPU) uint16 { return c.HL() },
		},
		{
			name: "INC_SP_0x33", opcode: 0x33,
			setup: func(c *CPU) { c.Registers.SP = 0xABCD },
			read:  func(c *CPU) uint16 { return c.Registers.SP },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			tc.setup(cpu)
			before := tc.read(cpu)
			startPC := cpu.Registers.PC
			// Set every flag bit to demonstrate INC rr does not touch F.
			cpu.Registers.F = 0xF0
			startF := cpu.Registers.F

			cycles := OPCODES[tc.opcode](mb, 0)

			assert.Equal(t, OpCycles(8), cycles, "cycles")
			assert.Equal(t, startPC+1, cpu.Registers.PC, "PC advance")
			assert.Equal(t, before+1, tc.read(cpu), "register pair += 1 (with wrap)")
			assert.Equal(t, startF, cpu.Registers.F, "INC rr must not touch flags")
		})
	}
}

// ---------------------------------------------------------------------------
// DEC rr  (0x0B, 0x1B, 0x2B, 0x3B)  -- NO flag changes
// ---------------------------------------------------------------------------

func TestIncDec16_DEC(t *testing.T) {
	cases := []struct {
		name   string
		opcode OpCode
		setup  func(*CPU)
		read   func(*CPU) uint16
	}{
		{
			name: "DEC_BC_0x0B", opcode: 0x0B,
			setup: func(c *CPU) { c.SetBC(0x1000) },
			read:  func(c *CPU) uint16 { return c.BC() },
		},
		{
			name: "DEC_DE_0x1B", opcode: 0x1B,
			setup: func(c *CPU) { c.SetDE(0x0001) }, // -> 0x0000
			read:  func(c *CPU) uint16 { return c.DE() },
		},
		{
			name: "DEC_HL_0x2B", opcode: 0x2B,
			setup: func(c *CPU) { c.SetHL(0x0000) }, // wrap to 0xFFFF
			read:  func(c *CPU) uint16 { return c.HL() },
		},
		{
			name: "DEC_SP_0x3B", opcode: 0x3B,
			setup: func(c *CPU) { c.Registers.SP = 0xC000 },
			read:  func(c *CPU) uint16 { return c.Registers.SP },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			tc.setup(cpu)
			before := tc.read(cpu)
			startPC := cpu.Registers.PC
			cpu.Registers.F = 0xF0
			startF := cpu.Registers.F

			cycles := OPCODES[tc.opcode](mb, 0)

			assert.Equal(t, OpCycles(8), cycles, "cycles")
			assert.Equal(t, startPC+1, cpu.Registers.PC, "PC advance")
			assert.Equal(t, before-1, tc.read(cpu), "register pair -= 1 (with wrap)")
			assert.Equal(t, startF, cpu.Registers.F, "DEC rr must not touch flags")
		})
	}
}
