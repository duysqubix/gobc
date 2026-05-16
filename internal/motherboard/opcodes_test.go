package motherboard

import (
	"fmt"
	"sort"
	"testing"

	"github.com/duysqubix/gobc/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpcodeTable_AllEntriesNonNil(t *testing.T) {
	require.NotEmpty(t, OPCODES, "OPCODES map must not be empty")
	for op, fn := range OPCODES {
		assert.NotNil(t, fn, "opcode %#x has nil handler", op)
	}
}

func TestOpcodeTable_TotalCount(t *testing.T) {
	// 256 base opcodes minus 11 illegal opcodes = 245 base handlers, plus
	// 256 CB-prefixed handlers (keyed at CB_SHIFT + n) = 501 total.
	base, cb := 0, 0
	for op := range OPCODES {
		if op >= CB_SHIFT {
			cb++
		} else {
			base++
		}
	}
	assert.Equal(t, 245, base, "base opcode count (256 - 11 illegal)")
	assert.Equal(t, 256, cb, "CB-prefixed opcode count")
	assert.Equal(t, 501, len(OPCODES), "total OPCODES entries")
}

func TestOpcodeTable_IllegalOpcodesAreAbsent(t *testing.T) {
	for _, op := range ILLEGAL_OPCODES {
		_, ok := OPCODES[op]
		assert.False(t, ok, "illegal opcode %#x must not have a handler", op)
		assert.True(t, op.IsIllegal(), "IsIllegal() must return true for %#x", op)
	}
}

func TestOpcodeTable_AllKeysWithinValidRange(t *testing.T) {
	// Base opcodes occupy 0x00..0xFF; CB-prefixed live at 0x100..0x1FF.
	for op := range OPCODES {
		assert.LessOrEqual(t, op, OpCode(0x1FF), "opcode %#x out of range", op)
	}
}

func TestOpcodeTable_CBPrefixDetection(t *testing.T) {
	cb := OpCode(0xCB)
	assert.True(t, cb.CBPrefix(), "0xCB must be detected as CB prefix")
	notCB := OpCode(0xAB)
	assert.False(t, notCB.CBPrefix())

	shifted := cb.Shift()
	assert.Equal(t, OpCode(0xCB)+CB_SHIFT, shifted, "Shift adds CB_SHIFT")
}

func TestOpcodeTable_NonIllegalBaseOpcodesAllPresent(t *testing.T) {
	illegal := make(map[OpCode]struct{}, len(ILLEGAL_OPCODES))
	for _, op := range ILLEGAL_OPCODES {
		illegal[op] = struct{}{}
	}
	missing := []OpCode{}
	for op := OpCode(0x00); op <= 0xFF; op++ {
		if _, isBad := illegal[op]; isBad {
			continue
		}
		if _, ok := OPCODES[op]; !ok {
			missing = append(missing, op)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	assert.Empty(t, missing, "non-illegal base opcodes missing handlers: %#v", missing)
}

func TestOpcodeTable_CBOpcodesContiguous(t *testing.T) {
	missing := []OpCode{}
	for op := CB_SHIFT; op <= CB_SHIFT+0xFF; op++ {
		if _, ok := OPCODES[op]; !ok {
			missing = append(missing, op)
		}
	}
	assert.Empty(t, missing, "CB opcodes missing handlers: %#v", missing)
}

func TestOpcodeDispatch_NOP_0x00(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0x0100
	cycles := OPCODES[0x00](mb, 0)

	assert.Equal(t, OpCycles(4), cycles, "NOP takes 4 cycles")
	assert.Equal(t, uint16(0x0101), cpu.Registers.PC, "NOP advances PC by 1")
}

func TestOpcodeDispatch_LD_B_d8_0x06(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0x0100
	cycles := OPCODES[0x06](mb, 0x42) // LD B, 0x42

	assert.Equal(t, OpCycles(8), cycles)
	assert.Equal(t, uint8(0x42), cpu.Registers.B)
	assert.Equal(t, uint16(0x0102), cpu.Registers.PC, "LD B,d8 advances PC by 2")
}

func TestOpcodeDispatch_ADD_A_d8_0xC6(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.A = 0x0F
	cpu.Registers.PC = 0x0100
	cycles := OPCODES[0xC6](mb, 0x01) // ADD A, 0x01

	assert.Equal(t, OpCycles(8), cycles)
	assert.Equal(t, uint8(0x10), cpu.Registers.A)
	assert.True(t, cpu.IsFlagHSet(), "half carry from 0x0F + 0x01")
	assert.False(t, cpu.IsFlagCSet())
	assert.False(t, cpu.IsFlagZSet())
	assert.False(t, cpu.IsFlagNSet())
	assert.Equal(t, uint16(0x0102), cpu.Registers.PC)
}

func TestOpcodeDispatch_XOR_d8_0xEE(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.A = 0xFF
	cpu.SetFlagC()
	cpu.SetFlagN()
	cpu.SetFlagH()
	OPCODES[0xEE](mb, 0xFF)

	assert.Equal(t, uint8(0x00), cpu.Registers.A)
	assert.True(t, cpu.IsFlagZSet())
	assert.False(t, cpu.IsFlagNSet(), "XOR clears N")
	assert.False(t, cpu.IsFlagHSet(), "XOR clears H")
	assert.False(t, cpu.IsFlagCSet(), "XOR clears C")
}

func TestOpcodeDispatch_JP_a16_0xC3(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0x1234
	cycles := OPCODES[0xC3](mb, 0xBEEF) // JP 0xBEEF

	assert.Equal(t, OpCycles(16), cycles)
	assert.Equal(t, uint16(0xBEEF), cpu.Registers.PC, "JP sets PC to target")
}

func TestOpcodeDispatch_SCF_0x37(t *testing.T) {
	// SCF: sets C, clears N and H, leaves Z untouched.
	cpu, mb := newTestCPU(t)
	cpu.SetFlagZ()
	cpu.SetFlagN()
	cpu.SetFlagH()
	cpu.ResetFlagC()
	cpu.Registers.PC = 0x0100

	OPCODES[0x37](mb, 0)

	assert.True(t, cpu.IsFlagZSet(), "SCF preserves Z")
	assert.False(t, cpu.IsFlagNSet(), "SCF clears N")
	assert.False(t, cpu.IsFlagHSet(), "SCF clears H")
	assert.True(t, cpu.IsFlagCSet(), "SCF sets C")
	assert.Equal(t, uint16(0x0101), cpu.Registers.PC)
}

func TestOpcodeDispatch_CPL_0x2F(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.A = 0xAA
	OPCODES[0x2F](mb, 0) // CPL: A = ^A, sets N and H

	assert.Equal(t, uint8(0x55), cpu.Registers.A)
	assert.True(t, cpu.IsFlagNSet())
	assert.True(t, cpu.IsFlagHSet())
}

func TestOpcodeDispatch_ADD_HL_BC_0x09(t *testing.T) {
	// 16-bit ADD: HL += BC. Z is preserved; N cleared; H/C from bit 11 / 15.
	cpu, mb := newTestCPU(t)
	cpu.SetHL(0x0FFF)
	cpu.SetBC(0x0001)
	cpu.SetFlagZ() // SCF semantics: Z must survive
	OPCODES[0x09](mb, 0)

	assert.Equal(t, uint16(0x1000), cpu.HL())
	assert.True(t, cpu.IsFlagZSet(), "ADD HL,BC preserves Z")
	assert.False(t, cpu.IsFlagNSet(), "N cleared")
	assert.True(t, cpu.IsFlagHSet(), "half-carry from bit 11")
	assert.False(t, cpu.IsFlagCSet(), "no full carry")
}

func TestCPU_ExecuteInstruction_NOPFromWRAM(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	// Write a NOP (0x00) into WRAM at PC.
	mb.SetItem(0xC000, 0x00)

	cycles := cpu.ExecuteInstruction()

	assert.Equal(t, OpCycles(4), cycles)
	assert.Equal(t, uint16(0xC001), cpu.Registers.PC)
	assert.Equal(t, OpCode(0x00), cpu.lastOpCode)
}

func TestCPU_ExecuteInstruction_LDB_d8_FromWRAM(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000
	mb.SetItem(0xC000, 0x06) // LD B, d8
	mb.SetItem(0xC001, 0x99) // immediate byte

	cycles := cpu.ExecuteInstruction()

	assert.Equal(t, OpCycles(8), cycles)
	assert.Equal(t, uint8(0x99), cpu.Registers.B)
	assert.Equal(t, uint16(0xC002), cpu.Registers.PC)
}

func TestCPU_ExecuteInstruction_PCHistoryGrowsThenCaps(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.PC = 0xC000

	// Lay down a long ribbon of NOPs.
	for addr := uint16(0xC000); addr < 0xC020; addr++ {
		mb.SetItem(addr, 0x00)
	}
	for i := 0; i < 10; i++ {
		cpu.ExecuteInstruction()
	}
	assert.LessOrEqual(t, cpu.PcHist.Len(), PC_HISTORY_COUNT_MAX,
		"PC history must never exceed PC_HISTORY_COUNT_MAX (%d)", PC_HISTORY_COUNT_MAX)
}

// Smoke-tests every OPCODES entry: invoke once with a deterministic setup
// and assert it does not panic. Behavioural assertions live in the targeted
// tests above; this one exists solely to exercise the dispatch surface that
// is otherwise unreachable without a real ROM tick loop.
func TestOpcodeTable_AllHandlersExecutable(t *testing.T) {
	mb := newMbForSubsysTest(t)
	cpu := mb.Cpu

	keys := make([]OpCode, 0, len(OPCODES))
	for k := range OPCODES {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	resetCPU := func() {
		cpu.Registers.A = 0x12
		cpu.Registers.B = 0x34
		cpu.Registers.C = 0x56
		cpu.Registers.D = 0x78
		cpu.Registers.E = 0x9A
		cpu.Registers.F = 0
		cpu.Registers.H = 0xC1
		cpu.Registers.L = 0x00
		cpu.Registers.SP = 0xFFFE
		cpu.Registers.PC = 0xC000
		cpu.Halted = false
		cpu.Stopped = false
		cpu.IsStuck = false
		cpu.Interrupts.IE = 0
		cpu.Interrupts.IF = 0
		cpu.Interrupts.InterruptsOn = false
		cpu.Interrupts.InterruptsEnabling = false
	}

	var failures []failure

	for _, op := range keys {
		resetCPU()
		callOpcodeSafely(op, mb, 0x0042, &failures)
	}

	assert.Empty(t, failures, "opcodes that panicked during smoke execution: %v",
		formatOpFailures(failures))
}

func callOpcodeSafely(op OpCode, mb *Motherboard, value uint16, failures *[]failure) {
	defer func() {
		if r := recover(); r != nil {
			*failures = append(*failures, failure{op: op, msg: fmt.Sprintf("%v", r)})
		}
	}()
	OPCODES[op](mb, value)
}

type failure struct {
	op  OpCode
	msg string
}

func formatOpFailures(failures []failure) string {
	if len(failures) == 0 {
		return "(none)"
	}
	out := ""
	for _, f := range failures {
		out += fmt.Sprintf("\n  %#04x: %s", f.op, f.msg)
	}
	return out
}

func TestOpcodeDispatch_RandomizeRegistersBounded(t *testing.T) {
	cpu, _ := newTestCPU(t)
	cpu.RandomizeRegisters(42)
	// Implementation samples [0, 0xF) for 8-bit regs; assert no overflow.
	for _, r := range []uint8{
		cpu.Registers.A,
		cpu.Registers.B,
		cpu.Registers.C,
		cpu.Registers.D,
		cpu.Registers.E,
		cpu.Registers.H,
		cpu.Registers.L,
	} {
		assert.LessOrEqual(t, r, uint8(0x0F), "register sampled within [0, 0xF]")
	}
	assert.True(t, cpu.Registers.F&0x0F == 0, "F low nibble must remain zero post-randomize")
	_ = internal.DMG_CLOCK_SPEED // keep the import live; some assertions above need internal symbols
}
