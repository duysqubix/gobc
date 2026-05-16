package motherboard

import (
	"testing"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/bootrom"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------
//
// The motherboard package does not (today) expose a clean way to construct a
// CPU in isolation: `NewMotherboard` requires a real cartridge file on disk
// and the alternative "dummy ROM" path inside `cartridge.NewCartridge` panics
// because the synthesised header has SRAM_SIZE_ADDR == 0xFF, which falls into
// the `default` branch of the RAM-size switch and calls `logger.Panicf`.
//
// To keep these tests fast and hermetic we build a *Motherboard* manually,
// wiring up only the components the CPU under test actually touches:
//
//   * Memory      – needed for HRAM-backed stack ops (PUSH/POP/CALL/RET).
//   * Timer       – referenced from a handful of opcodes via DIV/TAC.
//   * Input       – referenced from the joypad opcode path.
//   * BootRom     – queried by GetItem when PC < 0x100; we disable it so
//                   instruction fetch from WRAM works without a cartridge.
//   * Palettes    – created so any CGB palette code is non-nil.
//   * Breakpoints – CPU.ExecuteInstruction consults Mb.Breakpoints.Enabled.
//
// Crucially we do NOT create a Cartridge. Any opcode that dereferences
// mb.Cartridge would NPE; the test set below sticks to opcodes that operate
// purely on registers, HRAM, or WRAM.

func newTestCPU(t *testing.T) (*CPU, *Motherboard) {
	t.Helper()

	// Silence noisy logrus output triggered by some opcode paths.
	internal.Logger.SetLevel(log.PanicLevel)

	mb := &Motherboard{
		Cgb:           false,
		CpuFreq:       internal.DMG_CLOCK_SPEED,
		Decouple:      false,
		Breakpoints:   &Breakpoints{Enabled: false},
		Timer:         NewTimer(),
		BGPalette:     NewPalette(),
		SpritePalette: NewPalette(),
	}
	mb.Input = NewInput(mb)
	mb.Cpu = NewCpu(mb)
	mb.Memory = NewInternalRAM(mb, false)
	mb.BootRom = bootrom.NewBootRom(false)
	mb.BootRom.Disable() // tests fetch instructions from WRAM/HRAM, not boot ROM

	cpu := mb.Cpu
	cpu.Registers.A = 0
	cpu.Registers.B = 0
	cpu.Registers.C = 0
	cpu.Registers.D = 0
	cpu.Registers.E = 0
	cpu.Registers.F = 0
	cpu.Registers.H = 0
	cpu.Registers.L = 0
	cpu.Registers.SP = 0xFFFE
	cpu.Registers.PC = 0xC000

	return cpu, mb
}

// ---------------------------------------------------------------------------
// Register accessor tests
// ---------------------------------------------------------------------------

func TestCPU_Registers_SingleReadWrite(t *testing.T) {
	cpu, _ := newTestCPU(t)

	cases := []struct {
		name string
		set  func(uint8)
		get  func() uint8
	}{
		{"A", func(v uint8) { cpu.Registers.A = v }, func() uint8 { return cpu.Registers.A }},
		{"B", func(v uint8) { cpu.Registers.B = v }, func() uint8 { return cpu.Registers.B }},
		{"C", func(v uint8) { cpu.Registers.C = v }, func() uint8 { return cpu.Registers.C }},
		{"D", func(v uint8) { cpu.Registers.D = v }, func() uint8 { return cpu.Registers.D }},
		{"E", func(v uint8) { cpu.Registers.E = v }, func() uint8 { return cpu.Registers.E }},
		{"H", func(v uint8) { cpu.Registers.H = v }, func() uint8 { return cpu.Registers.H }},
		{"L", func(v uint8) { cpu.Registers.L = v }, func() uint8 { return cpu.Registers.L }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, v := range []uint8{0x00, 0x01, 0x7F, 0x80, 0xAA, 0x55, 0xFF} {
				c.set(v)
				assert.Equal(t, v, c.get(), "register %s round-trip for %#02x", c.name, v)
			}
		})
	}
}

func TestCPU_Registers_FDirectAcceptsLowNibble(t *testing.T) {
	// Direct writes to F today do NOT mask the low nibble. The real Game Boy
	// hardware keeps F bits 0..3 wired to ground, so reading F always
	// returns a value with the lower nibble cleared. The flag-mutation
	// helpers (SetFlagZ etc.) preserve that invariant naturally because
	// they only ever touch bits 4..7. This test characterises the *current*
	// behaviour so that a future fix (masking on assignment / SetAF) will
	// surface here intentionally.
	cpu, _ := newTestCPU(t)
	cpu.Registers.F = 0xFF
	assert.Equal(t, uint8(0xFF), cpu.Registers.F,
		"raw F register write is currently un-masked (Game Boy quirk not enforced)")
}

// ---------------------------------------------------------------------------
// 16-bit register-pair accessor tests
// ---------------------------------------------------------------------------

func TestCPU_Pairs_BC_DE_HL_AF(t *testing.T) {
	cpu, _ := newTestCPU(t)

	const value uint16 = 0xABCD

	cpu.SetBC(value)
	assert.Equal(t, uint8(0xAB), cpu.Registers.B, "BC high byte -> B")
	assert.Equal(t, uint8(0xCD), cpu.Registers.C, "BC low byte -> C")
	assert.Equal(t, value, cpu.BC(), "BC() round-trip")

	cpu.SetDE(value)
	assert.Equal(t, uint8(0xAB), cpu.Registers.D)
	assert.Equal(t, uint8(0xCD), cpu.Registers.E)
	assert.Equal(t, value, cpu.DE())

	cpu.SetHL(value)
	assert.Equal(t, uint8(0xAB), cpu.Registers.H)
	assert.Equal(t, uint8(0xCD), cpu.Registers.L)
	assert.Equal(t, value, cpu.HL())

	// AF is special-cased so that the lower nibble of F should remain zero.
	// Today SetAF mirrors the raw F register without masking, which is a
	// minor deviation from real hardware. We test the documented behaviour
	// here and call it out as a finding in the test summary.
	cpu.SetAF(0xAB12)
	assert.Equal(t, uint8(0xAB), cpu.Registers.A, "AF high byte -> A")
	assert.Equal(t, uint8(0x12), cpu.Registers.F,
		"AF low byte -> F (no masking applied today; real hardware would mask 0x12->0x10)")
}

func TestCPU_Pairs_FlagHelpersPreserveLowerNibble(t *testing.T) {
	// Verify the Set/Reset/Toggle flag API never disturbs the lower nibble.
	cpu, _ := newTestCPU(t)
	cpu.Registers.F = 0
	cpu.SetFlagZ()
	cpu.SetFlagN()
	cpu.SetFlagH()
	cpu.SetFlagC()
	assert.Equal(t, uint8(0xF0), cpu.Registers.F, "all four flag bits set, low nibble untouched")
	cpu.ResetAllFlags()
	assert.Equal(t, uint8(0x00), cpu.Registers.F, "ResetAllFlags clears entire F")
}

// ---------------------------------------------------------------------------
// PC and SP accessor tests
// ---------------------------------------------------------------------------

func TestCPU_PCSP_ReadWrite(t *testing.T) {
	cpu, _ := newTestCPU(t)

	for _, pc := range []uint16{0x0000, 0x0100, 0x4000, 0x8000, 0xC000, 0xFFFF} {
		cpu.Registers.PC = pc
		assert.Equal(t, pc, cpu.Registers.PC, "PC round-trip %#04x", pc)
	}
	for _, sp := range []uint16{0x0000, 0x00FF, 0xC000, 0xFFFE, 0xFFFF} {
		cpu.Registers.SP = sp
		assert.Equal(t, sp, cpu.Registers.SP, "SP round-trip %#04x", sp)
	}
}

// ---------------------------------------------------------------------------
// Flag bit position + Set / Reset / Toggle tests
// ---------------------------------------------------------------------------

func TestCPU_Flags_BitPositions(t *testing.T) {
	// Real Game Boy F register layout: Z=bit7, N=bit6, H=bit5, C=bit4.
	cpu, _ := newTestCPU(t)
	cpu.Registers.F = 0
	cpu.SetFlagZ()
	assert.Equal(t, uint8(1<<7), cpu.Registers.F, "Z flag = bit 7 (0x80)")
	cpu.Registers.F = 0
	cpu.SetFlagN()
	assert.Equal(t, uint8(1<<6), cpu.Registers.F, "N flag = bit 6 (0x40)")
	cpu.Registers.F = 0
	cpu.SetFlagH()
	assert.Equal(t, uint8(1<<5), cpu.Registers.F, "H flag = bit 5 (0x20)")
	cpu.Registers.F = 0
	cpu.SetFlagC()
	assert.Equal(t, uint8(1<<4), cpu.Registers.F, "C flag = bit 4 (0x10)")
}

func TestCPU_Flags_IndividualSetResetToggle(t *testing.T) {
	cpu, _ := newTestCPU(t)

	type op struct {
		name    string
		set     func()
		reset   func()
		toggle  func()
		isSet   func() bool
		bitMask uint8
	}
	ops := []op{
		{"Z", cpu.SetFlagZ, cpu.ResetFlagZ, cpu.ToggleFlagZ, cpu.IsFlagZSet, 1 << 7},
		{"N", cpu.SetFlagN, cpu.ResetFlagN, cpu.ToggleFlagN, cpu.IsFlagNSet, 1 << 6},
		{"H", cpu.SetFlagH, cpu.ResetFlagH, cpu.ToggleFlagH, cpu.IsFlagHSet, 1 << 5},
		{"C", cpu.SetFlagC, cpu.ResetFlagC, cpu.ToggleFlagC, cpu.IsFlagCSet, 1 << 4},
	}
	for _, o := range ops {
		t.Run(o.name, func(t *testing.T) {
			cpu.Registers.F = 0
			require.False(t, o.isSet())

			o.set()
			assert.True(t, o.isSet())
			assert.Equal(t, o.bitMask, cpu.Registers.F&o.bitMask)

			o.reset()
			assert.False(t, o.isSet())
			assert.Equal(t, uint8(0), cpu.Registers.F&o.bitMask)

			o.toggle()
			assert.True(t, o.isSet())
			o.toggle()
			assert.False(t, o.isSet())
		})
	}
}

func TestCPU_Flags_AllSixteenCombinationsRoundTrip(t *testing.T) {
	cpu, _ := newTestCPU(t)

	for combo := 0; combo < 16; combo++ {
		cpu.Registers.F = 0
		z := combo&0b1000 != 0
		n := combo&0b0100 != 0
		h := combo&0b0010 != 0
		c := combo&0b0001 != 0

		if z {
			cpu.SetFlagZ()
		}
		if n {
			cpu.SetFlagN()
		}
		if h {
			cpu.SetFlagH()
		}
		if c {
			cpu.SetFlagC()
		}

		assert.Equal(t, z, cpu.IsFlagZSet(), "combo %#x Z", combo)
		assert.Equal(t, n, cpu.IsFlagNSet(), "combo %#x N", combo)
		assert.Equal(t, h, cpu.IsFlagHSet(), "combo %#x H", combo)
		assert.Equal(t, c, cpu.IsFlagCSet(), "combo %#x C", combo)
		assert.Equal(t, uint8(0), cpu.Registers.F&0x0F,
			"combo %#x: lower nibble must stay zero after flag-helper writes", combo)
	}
}

func TestCPU_Flags_ClearAllFlags(t *testing.T) {
	cpu, _ := newTestCPU(t)
	cpu.Registers.F = 0xFF
	cpu.ClearAllFlags()
	assert.Equal(t, uint8(0), cpu.Registers.F)
}

// ---------------------------------------------------------------------------
// Stack operations: PUSH BC (0xC5) and POP BC (0xC1)
// ---------------------------------------------------------------------------

func TestCPU_Stack_PushPopRoundTrip(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.SP = 0xFFFE
	cpu.SetBC(0xABCD)

	OPCODES[0xC5](mb, 0) // PUSH BC
	assert.Equal(t, uint16(0xFFFC), cpu.Registers.SP, "SP decremented by 2 on PUSH")

	// Wipe BC, then POP back.
	cpu.SetBC(0x0000)
	OPCODES[0xC1](mb, 0) // POP BC
	assert.Equal(t, uint16(0xABCD), cpu.BC(), "POP restores pushed value")
	assert.Equal(t, uint16(0xFFFE), cpu.Registers.SP, "SP incremented by 2 on POP")
}

func TestCPU_Stack_LIFOOrder(t *testing.T) {
	cpu, mb := newTestCPU(t)
	cpu.Registers.SP = 0xFFFE

	cpu.SetBC(0x1122)
	OPCODES[0xC5](mb, 0) // push 0x1122
	cpu.SetBC(0x3344)
	OPCODES[0xC5](mb, 0) // push 0x3344
	assert.Equal(t, uint16(0xFFFA), cpu.Registers.SP)

	cpu.SetBC(0x0000)
	OPCODES[0xC1](mb, 0) // pop -> 0x3344
	assert.Equal(t, uint16(0x3344), cpu.BC(), "LIFO: last pushed comes out first")

	cpu.SetBC(0x0000)
	OPCODES[0xC1](mb, 0) // pop -> 0x1122
	assert.Equal(t, uint16(0x1122), cpu.BC())
	assert.Equal(t, uint16(0xFFFE), cpu.Registers.SP)
}

func TestCPU_Stack_PushAFMasksLowNibbleOnPop(t *testing.T) {
	// PUSH AF stores F as-is; POP AF masks the low nibble. So pushing F=0x12
	// and popping should give F=0x10.
	cpu, mb := newTestCPU(t)
	cpu.Registers.SP = 0xFFFE
	cpu.Registers.A = 0xDE
	cpu.Registers.F = 0x12 // intentionally dirty low nibble

	OPCODES[0xF5](mb, 0) // PUSH AF
	cpu.Registers.A = 0
	cpu.Registers.F = 0
	OPCODES[0xF1](mb, 0) // POP AF

	assert.Equal(t, uint8(0xDE), cpu.Registers.A, "A restored exactly")
	assert.Equal(t, uint8(0x10), cpu.Registers.F,
		"POP AF must clear the lower nibble of F (real-hw quirk enforced here)")
}

// ---------------------------------------------------------------------------
// Arithmetic instruction tests (table-driven).
//
// We exercise the CPU helpers directly (AddSetFlags8 etc.). That is exactly
// what the OPCODES table delegates to for ADD A,r / SUB A,r / etc., and it
// lets us cover all flag-edge combinations cheaply without setting up
// register-source variants for each row.
// ---------------------------------------------------------------------------

type aluRow struct {
	name       string
	a, b       uint8
	carryIn    bool // only consulted by ADC / SBC tests
	want       uint8
	z, n, h, c bool
}

func TestCPU_ADD_A(t *testing.T) {
	rows := []aluRow{
		{"0+0=0, Z set", 0x00, 0x00, false, 0x00, true, false, false, false},
		{"1+1=2, no flags", 0x01, 0x01, false, 0x02, false, false, false, false},
		{"0x0F+0x01=0x10, H only", 0x0F, 0x01, false, 0x10, false, false, true, false},
		{"0x10+0x10=0x20, no half", 0x10, 0x10, false, 0x20, false, false, false, false},
		{"0xF0+0x10=0x00, Z+C", 0xF0, 0x10, false, 0x00, true, false, false, true},
		{"0xFF+0x01=0x00, Z+H+C", 0xFF, 0x01, false, 0x00, true, false, true, true},
		{"0x80+0x80=0x00, Z+C", 0x80, 0x80, false, 0x00, true, false, false, true},
		{"0x3A+0xC6=0x00, Z+H+C", 0x3A, 0xC6, false, 0x00, true, false, true, true},
		{"0x08+0x08=0x10, H only", 0x08, 0x08, false, 0x10, false, false, true, false},
		{"0xC0+0x40=0x00, Z+C", 0xC0, 0x40, false, 0x00, true, false, false, true},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			got := cpu.AddSetFlags8(r.a, r.b)
			assertALU(t, cpu, r, got)
		})
	}
}

func TestCPU_SUB_A(t *testing.T) {
	rows := []aluRow{
		{"0-0=0, Z+N", 0x00, 0x00, false, 0x00, true, true, false, false},
		{"5-1=4, N only", 0x05, 0x01, false, 0x04, false, true, false, false},
		{"0x10-0x01=0x0F, N+H", 0x10, 0x01, false, 0x0F, false, true, true, false},
		{"0x00-0x01=0xFF, N+H+C (borrow)", 0x00, 0x01, false, 0xFF, false, true, true, true},
		{"0x10-0x20=0xF0, N+C", 0x10, 0x20, false, 0xF0, false, true, false, true},
		{"0x3E-0x3E=0x00, Z+N", 0x3E, 0x3E, false, 0x00, true, true, false, false},
		{"0x80-0x01=0x7F, N+H", 0x80, 0x01, false, 0x7F, false, true, true, false},
		{"0xFF-0x00=0xFF, N only", 0xFF, 0x00, false, 0xFF, false, true, false, false},
		{"0x10-0x0F=0x01, N+H", 0x10, 0x0F, false, 0x01, false, true, true, false},
		{"0x01-0x02=0xFF, N+H+C", 0x01, 0x02, false, 0xFF, false, true, true, true},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			got := cpu.SubSetFlags8(r.a, r.b)
			assertALU(t, cpu, r, got)
		})
	}
}

func TestCPU_AND_A(t *testing.T) {
	rows := []aluRow{
		{"0xFF & 0xFF = 0xFF", 0xFF, 0xFF, false, 0xFF, false, false, true, false},
		{"0x0F & 0xF0 = 0, Z+H", 0x0F, 0xF0, false, 0x00, true, false, true, false},
		{"0xAA & 0x55 = 0, Z+H", 0xAA, 0x55, false, 0x00, true, false, true, false},
		{"0xAA & 0xFF = 0xAA, H", 0xAA, 0xFF, false, 0xAA, false, false, true, false},
		{"0x00 & 0x00 = 0, Z+H", 0x00, 0x00, false, 0x00, true, false, true, false},
		{"0x12 & 0x34 = 0x10, H", 0x12, 0x34, false, 0x10, false, false, true, false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			// AND should clear carry even if it was set going in.
			cpu.SetFlagC()
			cpu.SetFlagN()
			got := cpu.AndSetFlags(r.a, r.b)
			assertALU(t, cpu, r, got)
		})
	}
}

func TestCPU_OR_A(t *testing.T) {
	rows := []aluRow{
		{"0|0=0, Z", 0x00, 0x00, false, 0x00, true, false, false, false},
		{"0xF0|0x0F=0xFF", 0xF0, 0x0F, false, 0xFF, false, false, false, false},
		{"0xAA|0x55=0xFF", 0xAA, 0x55, false, 0xFF, false, false, false, false},
		{"0x12|0x80=0x92", 0x12, 0x80, false, 0x92, false, false, false, false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			cpu.SetFlagC()
			cpu.SetFlagH()
			cpu.SetFlagN()
			got := cpu.OrSetFlags(r.a, r.b)
			assertALU(t, cpu, r, got)
		})
	}
}

func TestCPU_XOR_A(t *testing.T) {
	rows := []aluRow{
		{"0xFF^0xFF=0, Z", 0xFF, 0xFF, false, 0x00, true, false, false, false},
		{"0x0F^0xF0=0xFF", 0x0F, 0xF0, false, 0xFF, false, false, false, false},
		{"0xAA^0x55=0xFF", 0xAA, 0x55, false, 0xFF, false, false, false, false},
		{"0x12^0x34=0x26", 0x12, 0x34, false, 0x26, false, false, false, false},
		{"0x00^0x00=0, Z", 0x00, 0x00, false, 0x00, true, false, false, false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			cpu.SetFlagC()
			cpu.SetFlagH()
			cpu.SetFlagN()
			got := cpu.XorSetFlags(r.a, r.b)
			assertALU(t, cpu, r, got)
		})
	}
}

func TestCPU_CP_A(t *testing.T) {
	// CP is SUB without writing the result back. We use OPCODES[0xFE] (CP d8)
	// so we also confirm A is left untouched by the dispatcher.
	rows := []aluRow{
		{"A=5,B=5 -> Z+N", 0x05, 0x05, false, 0x05 /* A unchanged */, true, true, false, false},
		{"A=5,B=1 -> N", 0x05, 0x01, false, 0x05, false, true, false, false},
		{"A=0x10,B=0x01 -> N+H", 0x10, 0x01, false, 0x10, false, true, true, false},
		{"A=0x00,B=0x01 -> N+H+C", 0x00, 0x01, false, 0x00, false, true, true, true},
		{"A=0x10,B=0x20 -> N+C", 0x10, 0x20, false, 0x10, false, true, false, true},
		{"A=0xFF,B=0xFE -> N", 0xFF, 0xFE, false, 0xFF, false, true, false, false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			cpu.Registers.A = r.a
			OPCODES[0xFE](mb, uint16(r.b))
			assert.Equal(t, r.want, cpu.Registers.A, "CP must not modify A")
			assertFlagsOnly(t, cpu, r)
		})
	}
}

func TestCPU_ADC_A(t *testing.T) {
	rows := []aluRow{
		{"0+0+0=0, Z", 0x00, 0x00, false, 0x00, true, false, false, false},
		{"0+0+1=1", 0x00, 0x00, true, 0x01, false, false, false, false},
		{"0x0F+0+1=0x10, H", 0x0F, 0x00, true, 0x10, false, false, true, false},
		{"0xFF+0+1=0x00, Z+H+C", 0xFF, 0x00, true, 0x00, true, false, true, true},
		{"0xFF+0x01+0=0x00, Z+H+C", 0xFF, 0x01, false, 0x00, true, false, true, true},
		{"0xFF+0xFF+1=0xFF, H+C", 0xFF, 0xFF, true, 0xFF, false, false, true, true},
		{"0x10+0x20+1=0x31", 0x10, 0x20, true, 0x31, false, false, false, false},
		{"0x0F+0x00+0=0x0F, no flags", 0x0F, 0x00, false, 0x0F, false, false, false, false},
		{"0x80+0x80+0=0x00, Z+C", 0x80, 0x80, false, 0x00, true, false, false, true},
		{"0xF0+0x10+0=0x00, Z+C (no half)", 0xF0, 0x10, false, 0x00, true, false, false, true},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			if r.carryIn {
				cpu.SetFlagC()
			}
			got := cpu.AdcSetFlags8(r.a, r.b)
			assertALU(t, cpu, r, got)
		})
	}
}

func TestCPU_SBC_A(t *testing.T) {
	rows := []aluRow{
		{"0-0-0=0, Z+N", 0x00, 0x00, false, 0x00, true, true, false, false},
		{"0-0-1=0xFF, N+H+C", 0x00, 0x00, true, 0xFF, false, true, true, true},
		{"0x10-0x01-0=0x0F, N+H", 0x10, 0x01, false, 0x0F, false, true, true, false},
		{"0x10-0x00-1=0x0F, N+H", 0x10, 0x00, true, 0x0F, false, true, true, false},
		{"0x05-0x05-0=0x00, Z+N", 0x05, 0x05, false, 0x00, true, true, false, false},
		{"0x05-0x05-1=0xFF, N+H+C", 0x05, 0x05, true, 0xFF, false, true, true, true},
		{"0x80-0x01-0=0x7F, N+H", 0x80, 0x01, false, 0x7F, false, true, true, false},
		{"0x10-0x20-0=0xF0, N+C", 0x10, 0x20, false, 0xF0, false, true, false, true},
		{"0x10-0x20-1=0xEF, N+H+C", 0x10, 0x20, true, 0xEF, false, true, true, true},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			if r.carryIn {
				cpu.SetFlagC()
			}
			got := cpu.SbcSetFlags8(r.a, r.b)
			assertALU(t, cpu, r, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Inc / Dec helpers (covers the 0x04/0x05/0x0C/0x0D/etc. opcode family).
// ---------------------------------------------------------------------------

func TestCPU_Inc(t *testing.T) {
	cases := []struct {
		name        string
		in          uint8
		want        uint8
		z, h, prevC bool
	}{
		{"0x00 -> 0x01", 0x00, 0x01, false, false, false},
		{"0x0F -> 0x10 sets H", 0x0F, 0x10, false, true, false},
		{"0xFF -> 0x00 wraps, sets Z and H, preserves C=1", 0xFF, 0x00, true, true, true},
		{"0x7F -> 0x80 sets H only", 0x7F, 0x80, false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			if c.prevC {
				cpu.SetFlagC()
			}
			got := cpu.Inc(c.in)
			assert.Equal(t, c.want, got)
			assert.Equal(t, c.z, cpu.IsFlagZSet(), "Z")
			assert.False(t, cpu.IsFlagNSet(), "INC clears N")
			assert.Equal(t, c.h, cpu.IsFlagHSet(), "H")
			assert.Equal(t, c.prevC, cpu.IsFlagCSet(), "C preserved across INC")
		})
	}
}

func TestCPU_Dec(t *testing.T) {
	cases := []struct {
		name        string
		in          uint8
		want        uint8
		z, h, prevC bool
	}{
		{"0x01 -> 0x00 sets Z", 0x01, 0x00, true, false, false},
		{"0x10 -> 0x0F sets H", 0x10, 0x0F, false, true, false},
		{"0x00 -> 0xFF wraps, sets H, preserves C=1", 0x00, 0xFF, false, true, true},
		{"0x80 -> 0x7F sets H", 0x80, 0x7F, false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, _ := newTestCPU(t)
			if c.prevC {
				cpu.SetFlagC()
			}
			got := cpu.Dec(c.in)
			assert.Equal(t, c.want, got)
			assert.Equal(t, c.z, cpu.IsFlagZSet(), "Z")
			assert.True(t, cpu.IsFlagNSet(), "DEC sets N")
			assert.Equal(t, c.h, cpu.IsFlagHSet(), "H")
			assert.Equal(t, c.prevC, cpu.IsFlagCSet(), "C preserved across DEC")
		})
	}
}

// ---------------------------------------------------------------------------
// Shared assertion helpers
// ---------------------------------------------------------------------------

func assertALU(t *testing.T, cpu *CPU, r aluRow, got uint8) {
	t.Helper()
	assert.Equal(t, r.want, got, "result")
	assertFlagsOnly(t, cpu, r)
}

func assertFlagsOnly(t *testing.T, cpu *CPU, r aluRow) {
	t.Helper()
	assert.Equal(t, r.z, cpu.IsFlagZSet(), "Z flag")
	assert.Equal(t, r.n, cpu.IsFlagNSet(), "N flag")
	assert.Equal(t, r.h, cpu.IsFlagHSet(), "H flag")
	assert.Equal(t, r.c, cpu.IsFlagCSet(), "C flag")
}
