// CB-prefixed opcode tests.
//
// The CB-prefixed family (256 opcodes) is laid out by group, with each group
// applying a single bitwise operation to one of eight operands:
//
//	index 0..7 -> B, C, D, E, H, L, (HL), A
//
// Group layout (offset within the CB page):
//
//	0x00..0x07  RLC r
//	0x08..0x0F  RRC r
//	0x10..0x17  RL r
//	0x18..0x1F  RR r
//	0x20..0x27  SLA r
//	0x28..0x2F  SRA r
//	0x30..0x37  SWAP r
//	0x38..0x3F  SRL r
//	0x40..0x7F  BIT b,r   (8 bits x 8 regs = 64 opcodes)
//	0x80..0xBF  RES b,r   (64 opcodes)
//	0xC0..0xFF  SET b,r   (64 opcodes)
//
// The OPCODES map keys CB-prefixed opcodes at 0x100 + offset (CB_SHIFT moves
// them into a non-overlapping high half of the table).  All CB instructions
// consume two PC bytes (0xCB + opcode) and therefore advance PC by 2 inside
// the handler.  Cycle costs:
//
//	register variants:                                 8
//	(HL) read/modify/write variants (RLC..SRL,RES,SET): 16
//	(HL) read-only variant (BIT):                       12
//
// Every CB opcode is exercised: the per-mnemonic test functions iterate the
// eight register variants (and, for BIT/RES/SET, the eight bit positions),
// marking the OpCode key on each invocation.  TestCB_zCoverage_AllOpcodes
// then asserts that exactly 256 unique CB opcodes were driven.
//
// All assertions are derived from the actual handler code in opcodes.go,
// not a textbook.  Subtle deviations (none observed at time of writing) are
// matched by the assertions and called out in comments.
package motherboard

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cbHLOperandAddr is the WRAM scratch address used as the (HL) operand
// target for memory-touching CB opcodes.  It sits inside Work RAM bank 0
// (0xC000..0xCFFF) which the test harness backs without a cartridge.
const cbHLOperandAddr uint16 = 0xC000

// cbRegisterOrder is the canonical CB operand index order.
var cbRegisterOrder = [...]string{"B", "C", "D", "E", "H", "L", "(HL)", "A"}

// cbExercised records every CB OpCode key driven by a test in this file.
// The trailing TestCB_zCoverage_AllOpcodes test asserts exactly 256 unique
// opcodes were touched.
var (
	cbExercisedMu sync.Mutex
	cbExercised   = map[OpCode]struct{}{}
)

func cbMark(op OpCode) {
	cbExercisedMu.Lock()
	cbExercised[op] = struct{}{}
	cbExercisedMu.Unlock()
}

// cbExpectedCycles returns the documented cycle count for a CB op based on
// the operand and whether the op is a BIT (BIT (HL) costs 12, all other
// (HL) variants 16, all register variants 8).
func cbExpectedCycles(name string, isBit bool) OpCycles {
	if name != "(HL)" {
		return 8
	}
	if isBit {
		return 12
	}
	return 16
}

// cbOperand wires read/write access for one of the eight CB operands.
// For the (HL) variant get/set route through Motherboard.GetItem/SetItem
// at cbHLOperandAddr; for register variants they touch the register
// directly.
type cbOperand struct {
	name  string
	isMem bool
	get   func() uint8
	set   func(uint8)
}

func cbBindOperand(t *testing.T, mb *Motherboard, name string) cbOperand {
	t.Helper()
	cpu := mb.Cpu
	op := cbOperand{name: name}
	switch name {
	case "B":
		op.get = func() uint8 { return cpu.Registers.B }
		op.set = func(v uint8) { cpu.Registers.B = v }
	case "C":
		op.get = func() uint8 { return cpu.Registers.C }
		op.set = func(v uint8) { cpu.Registers.C = v }
	case "D":
		op.get = func() uint8 { return cpu.Registers.D }
		op.set = func(v uint8) { cpu.Registers.D = v }
	case "E":
		op.get = func() uint8 { return cpu.Registers.E }
		op.set = func(v uint8) { cpu.Registers.E = v }
	case "H":
		op.get = func() uint8 { return cpu.Registers.H }
		op.set = func(v uint8) { cpu.Registers.H = v }
	case "L":
		op.get = func() uint8 { return cpu.Registers.L }
		op.set = func(v uint8) { cpu.Registers.L = v }
	case "A":
		op.get = func() uint8 { return cpu.Registers.A }
		op.set = func(v uint8) { cpu.Registers.A = v }
	case "(HL)":
		op.isMem = true
		cpu.SetHL(cbHLOperandAddr)
		op.get = func() uint8 { return mb.GetItem(cbHLOperandAddr) }
		op.set = func(v uint8) { mb.SetItem(cbHLOperandAddr, uint16(v)) }
	default:
		t.Fatalf("cbBindOperand: unknown operand %q", name)
	}
	return op
}

// cbResetForOp clears all registers, points HL at the (HL) scratch address
// (so that even register-variant tests start from a deterministic HL),
// seeds PC to a known WRAM offset, and returns the bound operand together
// with the seeded PC value.
func cbResetForOp(t *testing.T, mb *Motherboard, name string) (cbOperand, uint16) {
	t.Helper()
	cpu := mb.Cpu
	cpu.Registers.A = 0
	cpu.Registers.B = 0
	cpu.Registers.C = 0
	cpu.Registers.D = 0
	cpu.Registers.E = 0
	cpu.Registers.F = 0
	cpu.Registers.H = 0
	cpu.Registers.L = 0
	cpu.Registers.PC = 0xC050
	op := cbBindOperand(t, mb, name)
	return op, cpu.Registers.PC
}

// cbExpectedFlags packs (Z, N, H, C) into the same bit positions as the F
// register so we can compare the whole byte in a single assertion.
//
//	Z = bit 7 (0x80), N = bit 6 (0x40), H = bit 5 (0x20), C = bit 4 (0x10)
//
// The lower nibble of F is always zero after any flag-helper write, so an
// equality check on F doubles as a check that the lower nibble was not
// disturbed.
func cbExpectedFlags(z, n, h, c bool) uint8 {
	var f uint8
	if z {
		f |= 1 << 7
	}
	if n {
		f |= 1 << 6
	}
	if h {
		f |= 1 << 5
	}
	if c {
		f |= 1 << 4
	}
	return f
}

// ---------------------------------------------------------------------------
// RLC r — rotate left, MSB ejected to C and wrapped into bit 0.
// Z = (result == 0); N = 0; H = 0; C = old MSB.
// Opcode keys: 0x100 (B) .. 0x107 (A).
// ---------------------------------------------------------------------------

func TestCB_RLC(t *testing.T) {
	type row struct {
		in   uint8
		want uint8
		z, c bool
	}
	rows := []row{
		{in: 0x00, want: 0x00, z: true, c: false},  // all-zero
		{in: 0x01, want: 0x02, z: false, c: false}, // LSB only -> shifts left
		{in: 0x80, want: 0x01, z: false, c: true},  // MSB only -> wraps to LSB, C=1
		{in: 0xFF, want: 0xFF, z: false, c: true},  // all ones -> still all ones, C=1
		{in: 0xAA, want: 0x55, z: false, c: true},  // 1010_1010 -> 0101_0101
		{in: 0x55, want: 0xAA, z: false, c: false}, // 0101_0101 -> 1010_1010
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x", name, r.in), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(r.in)

				key := OpCode(0x100 + regIdx)
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				assert.Equalf(t, r.want, op.get(),
					"RLC %s in=%#02x: result", name, r.in)
				assert.Equalf(t, cbExpectedFlags(r.z, false, false, r.c),
					mb.Cpu.Registers.F, "RLC %s in=%#02x: flags", name, r.in)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"RLC %s in=%#02x: PC", name, r.in)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"RLC %s in=%#02x: cycles", name, r.in)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// RRC r — rotate right, LSB ejected to C and wrapped into bit 7.
// Z = (result == 0); N = 0; H = 0; C = old LSB.
// Opcode keys: 0x108 (B) .. 0x10F (A).
// ---------------------------------------------------------------------------

func TestCB_RRC(t *testing.T) {
	type row struct {
		in   uint8
		want uint8
		z, c bool
	}
	rows := []row{
		{in: 0x00, want: 0x00, z: true, c: false},
		{in: 0x01, want: 0x80, z: false, c: true},
		{in: 0x80, want: 0x40, z: false, c: false},
		{in: 0xFF, want: 0xFF, z: false, c: true},
		{in: 0xAA, want: 0x55, z: false, c: false},
		{in: 0x55, want: 0xAA, z: false, c: true},
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x", name, r.in), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(r.in)

				key := OpCode(0x108 + regIdx)
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				assert.Equalf(t, r.want, op.get(),
					"RRC %s in=%#02x: result", name, r.in)
				assert.Equalf(t, cbExpectedFlags(r.z, false, false, r.c),
					mb.Cpu.Registers.F, "RRC %s in=%#02x: flags", name, r.in)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"RRC %s in=%#02x: PC", name, r.in)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"RRC %s in=%#02x: cycles", name, r.in)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// RL r — rotate left through carry.
// new bit 0 = old C; new C = old MSB; Z = (result == 0); N = 0; H = 0.
// Opcode keys: 0x110 (B) .. 0x117 (A).
// ---------------------------------------------------------------------------

func TestCB_RL(t *testing.T) {
	type row struct {
		in      uint8
		carryIn bool
		want    uint8
		z, c    bool
	}
	rows := []row{
		// all-zero, carry-in 0 -> 0, Z=1
		{in: 0x00, carryIn: false, want: 0x00, z: true, c: false},
		// all-zero, carry-in 1 -> 0x01 (carry rotated in), Z=0, C=0
		{in: 0x00, carryIn: true, want: 0x01, z: false, c: false},
		// MSB only -> 0 with C=1, Z=1
		{in: 0x80, carryIn: false, want: 0x00, z: true, c: true},
		// MSB only with carry-in -> 0x01, C=1, Z=0
		{in: 0x80, carryIn: true, want: 0x01, z: false, c: true},
		// LSB only, no carry -> shift to 0x02
		{in: 0x01, carryIn: false, want: 0x02, z: false, c: false},
		// All ones, no carry -> 0xFE, C=1
		{in: 0xFF, carryIn: false, want: 0xFE, z: false, c: true},
		// All ones, carry -> 0xFF, C=1
		{in: 0xFF, carryIn: true, want: 0xFF, z: false, c: true},
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x/cin=%t", name, r.in, r.carryIn),
				func(t *testing.T) {
					_, mb := newTestCPU(t)
					op, oldPC := cbResetForOp(t, mb, name)
					op.set(r.in)
					if r.carryIn {
						mb.Cpu.SetFlagC()
					}

					key := OpCode(0x110 + regIdx)
					cycles := OPCODES[key](mb, 0)
					cbMark(key)

					assert.Equalf(t, r.want, op.get(),
						"RL %s in=%#02x cin=%t: result", name, r.in, r.carryIn)
					assert.Equalf(t, cbExpectedFlags(r.z, false, false, r.c),
						mb.Cpu.Registers.F,
						"RL %s in=%#02x cin=%t: flags", name, r.in, r.carryIn)
					assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
						"RL %s in=%#02x cin=%t: PC", name, r.in, r.carryIn)
					assert.Equalf(t, cbExpectedCycles(name, false), cycles,
						"RL %s in=%#02x cin=%t: cycles", name, r.in, r.carryIn)
				})
		}
	}
}

// ---------------------------------------------------------------------------
// RR r — rotate right through carry.
// new bit 7 = old C; new C = old LSB; Z = (result == 0); N = 0; H = 0.
// Opcode keys: 0x118 (B) .. 0x11F (A).
// ---------------------------------------------------------------------------

func TestCB_RR(t *testing.T) {
	type row struct {
		in      uint8
		carryIn bool
		want    uint8
		z, c    bool
	}
	rows := []row{
		{in: 0x00, carryIn: false, want: 0x00, z: true, c: false},
		{in: 0x00, carryIn: true, want: 0x80, z: false, c: false}, // carry rotated into bit 7
		{in: 0x01, carryIn: false, want: 0x00, z: true, c: true},  // LSB ejected, result zero
		{in: 0x01, carryIn: true, want: 0x80, z: false, c: true},  // bit 7 from carry, C from old LSB
		{in: 0x80, carryIn: false, want: 0x40, z: false, c: false},
		{in: 0xFF, carryIn: false, want: 0x7F, z: false, c: true},
		{in: 0xFF, carryIn: true, want: 0xFF, z: false, c: true},
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x/cin=%t", name, r.in, r.carryIn),
				func(t *testing.T) {
					_, mb := newTestCPU(t)
					op, oldPC := cbResetForOp(t, mb, name)
					op.set(r.in)
					if r.carryIn {
						mb.Cpu.SetFlagC()
					}

					key := OpCode(0x118 + regIdx)
					cycles := OPCODES[key](mb, 0)
					cbMark(key)

					assert.Equalf(t, r.want, op.get(),
						"RR %s in=%#02x cin=%t: result", name, r.in, r.carryIn)
					assert.Equalf(t, cbExpectedFlags(r.z, false, false, r.c),
						mb.Cpu.Registers.F,
						"RR %s in=%#02x cin=%t: flags", name, r.in, r.carryIn)
					assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
						"RR %s in=%#02x cin=%t: PC", name, r.in, r.carryIn)
					assert.Equalf(t, cbExpectedCycles(name, false), cycles,
						"RR %s in=%#02x cin=%t: cycles", name, r.in, r.carryIn)
				})
		}
	}
}

// ---------------------------------------------------------------------------
// SLA r — arithmetic shift left.
// result = (in << 1) & 0xFF; bit 0 forced to 0; C = old MSB; Z = (result==0).
// Opcode keys: 0x120 (B) .. 0x127 (A).
// ---------------------------------------------------------------------------

func TestCB_SLA(t *testing.T) {
	type row struct {
		in   uint8
		want uint8
		z, c bool
	}
	rows := []row{
		{in: 0x00, want: 0x00, z: true, c: false},
		{in: 0x01, want: 0x02, z: false, c: false},
		{in: 0x80, want: 0x00, z: true, c: true},  // MSB ejected -> 0
		{in: 0xFF, want: 0xFE, z: false, c: true}, // bit 0 forced to 0
		{in: 0xAA, want: 0x54, z: false, c: true},
		{in: 0x55, want: 0xAA, z: false, c: false},
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x", name, r.in), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(r.in)

				key := OpCode(0x120 + regIdx)
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				assert.Equalf(t, r.want, op.get(),
					"SLA %s in=%#02x: result", name, r.in)
				assert.Equalf(t, cbExpectedFlags(r.z, false, false, r.c),
					mb.Cpu.Registers.F, "SLA %s in=%#02x: flags", name, r.in)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"SLA %s in=%#02x: PC", name, r.in)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"SLA %s in=%#02x: cycles", name, r.in)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// SRA r — arithmetic shift right.
// MSB preserved (sign extend); C = old LSB; Z = (result == 0); N = 0; H = 0.
// Opcode keys: 0x128 (B) .. 0x12F (A).
// ---------------------------------------------------------------------------

func TestCB_SRA(t *testing.T) {
	type row struct {
		in   uint8
		want uint8
		z, c bool
	}
	rows := []row{
		{in: 0x00, want: 0x00, z: true, c: false},
		{in: 0x01, want: 0x00, z: true, c: true},   // LSB ejected; result zero
		{in: 0x80, want: 0xC0, z: false, c: false}, // MSB preserved -> 0xC0
		{in: 0x81, want: 0xC0, z: false, c: true},  // MSB preserved + LSB ejects to C
		{in: 0xFF, want: 0xFF, z: false, c: true},  // all ones stay (sign extend)
		{in: 0x40, want: 0x20, z: false, c: false},
		{in: 0xAA, want: 0xD5, z: false, c: false},
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x", name, r.in), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(r.in)

				key := OpCode(0x128 + regIdx)
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				assert.Equalf(t, r.want, op.get(),
					"SRA %s in=%#02x: result", name, r.in)
				assert.Equalf(t, cbExpectedFlags(r.z, false, false, r.c),
					mb.Cpu.Registers.F, "SRA %s in=%#02x: flags", name, r.in)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"SRA %s in=%#02x: PC", name, r.in)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"SRA %s in=%#02x: cycles", name, r.in)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// SWAP r — swap upper and lower nibbles.
// result = ((in & 0x0F) << 4) | ((in & 0xF0) >> 4); Z = (result == 0);
// N = 0; H = 0; C = 0 (always cleared).
// Opcode keys: 0x130 (B) .. 0x137 (A).
// ---------------------------------------------------------------------------

func TestCB_SWAP(t *testing.T) {
	type row struct {
		in   uint8
		want uint8
		z    bool
	}
	rows := []row{
		{in: 0x00, want: 0x00, z: true},
		{in: 0x12, want: 0x21, z: false},
		{in: 0xAB, want: 0xBA, z: false},
		{in: 0xF0, want: 0x0F, z: false},
		{in: 0x0F, want: 0xF0, z: false},
		{in: 0xFF, want: 0xFF, z: false},
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x", name, r.in), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(r.in)
				// Pre-set C to 1 so we prove SWAP clears it.
				mb.Cpu.SetFlagC()

				key := OpCode(0x130 + regIdx)
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				assert.Equalf(t, r.want, op.get(),
					"SWAP %s in=%#02x: result", name, r.in)
				assert.Equalf(t, cbExpectedFlags(r.z, false, false, false),
					mb.Cpu.Registers.F, "SWAP %s in=%#02x: flags (Z only)", name, r.in)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"SWAP %s in=%#02x: PC", name, r.in)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"SWAP %s in=%#02x: cycles", name, r.in)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// SRL r — logical shift right. MSB forced to 0; C = old LSB.
// Z = (result == 0); N = 0; H = 0.
// Opcode keys: 0x138 (B) .. 0x13F (A).
// ---------------------------------------------------------------------------

func TestCB_SRL(t *testing.T) {
	type row struct {
		in   uint8
		want uint8
		z, c bool
	}
	rows := []row{
		{in: 0x00, want: 0x00, z: true, c: false},
		{in: 0x01, want: 0x00, z: true, c: true},   // LSB ejected; result zero
		{in: 0x80, want: 0x40, z: false, c: false}, // MSB shifted right; bit 7 cleared
		{in: 0xFF, want: 0x7F, z: false, c: true},  // bit 7 cleared
		{in: 0xAA, want: 0x55, z: false, c: false},
		{in: 0x55, want: 0x2A, z: false, c: true},
	}
	for regIdx, name := range cbRegisterOrder {
		for _, r := range rows {
			t.Run(fmt.Sprintf("%s/in=%#02x", name, r.in), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(r.in)

				key := OpCode(0x138 + regIdx)
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				assert.Equalf(t, r.want, op.get(),
					"SRL %s in=%#02x: result", name, r.in)
				assert.Equalf(t, cbExpectedFlags(r.z, false, false, r.c),
					mb.Cpu.Registers.F, "SRL %s in=%#02x: flags", name, r.in)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"SRL %s in=%#02x: PC", name, r.in)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"SRL %s in=%#02x: cycles", name, r.in)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// BIT b, r — test bit b of operand.
// Z = !(operand & (1<<b));  N = 0;  H = 1;  C = unchanged.
// The operand value is NOT modified by BIT.
// Opcode keys: 0x140 + b*8 + reg, for b in 0..7 and reg in 0..7 -> 64 ops.
// ---------------------------------------------------------------------------

func TestCB_BIT(t *testing.T) {
	for bit := uint8(0); bit < 8; bit++ {
		for regIdx, name := range cbRegisterOrder {
			// Cover both branches: bit set, bit clear.  With operand=0xFF,
			// every bit is set, so Z must be 0; with operand=0x00, every
			// bit is clear, so Z must be 1.  We also exercise an operand
			// equal to (1<<bit) to confirm a single-bit positive isolates
			// just the bit under test.
			cases := []struct {
				operand uint8
				wantZ   bool
				label   string
			}{
				{operand: 0x00, wantZ: true, label: "all-zero"},
				{operand: 0xFF, wantZ: false, label: "all-ones"},
				{operand: 1 << bit, wantZ: false, label: "isolated-set"},
				{operand: ^(1 << bit) & 0xFF, wantZ: true, label: "isolated-clear"},
			}
			for _, cIn := range []bool{false, true} {
				for _, c := range cases {
					t.Run(fmt.Sprintf("b%d/%s/op=%#02x/cin=%t",
						bit, name, c.operand, cIn),
						func(t *testing.T) {
							_, mb := newTestCPU(t)
							op, oldPC := cbResetForOp(t, mb, name)
							op.set(c.operand)
							// Seed C so we can prove BIT does not touch it.
							if cIn {
								mb.Cpu.SetFlagC()
							}

							before := op.get()
							key := OpCode(0x140 + OpCode(bit)*8 + OpCode(regIdx))
							cycles := OPCODES[key](mb, 0)
							cbMark(key)

							// Operand must NOT be modified by BIT.
							assert.Equalf(t, before, op.get(),
								"BIT %d,%s op=%#02x: operand mutated",
								bit, name, c.operand)

							// Flags: Z per result, N=0, H=1, C unchanged.
							assert.Equalf(t, cbExpectedFlags(c.wantZ, false, true, cIn),
								mb.Cpu.Registers.F,
								"BIT %d,%s op=%#02x cin=%t: flags",
								bit, name, c.operand, cIn)
							assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
								"BIT %d,%s: PC", bit, name)
							assert.Equalf(t, cbExpectedCycles(name, true), cycles,
								"BIT %d,%s: cycles", bit, name)
						})
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// RES b, r — clear bit b of operand. NO flag change.
// Opcode keys: 0x180 + b*8 + reg.
// ---------------------------------------------------------------------------

func TestCB_RES(t *testing.T) {
	for bit := uint8(0); bit < 8; bit++ {
		for regIdx, name := range cbRegisterOrder {
			t.Run(fmt.Sprintf("b%d/%s", bit, name), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(0xFF) // every bit set; expect ~(1<<bit) afterwards
				// Seed every flag so we can prove RES leaves them alone.
				mb.Cpu.SetFlagZ()
				mb.Cpu.SetFlagN()
				mb.Cpu.SetFlagH()
				mb.Cpu.SetFlagC()
				flagsBefore := mb.Cpu.Registers.F

				key := OpCode(0x180 + OpCode(bit)*8 + OpCode(regIdx))
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				wantValue := uint8(0xFF) &^ (1 << bit)
				assert.Equalf(t, wantValue, op.get(),
					"RES %d,%s in=0xFF: result", bit, name)
				assert.Equalf(t, flagsBefore, mb.Cpu.Registers.F,
					"RES %d,%s: flags must be unchanged", bit, name)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"RES %d,%s: PC", bit, name)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"RES %d,%s: cycles", bit, name)

				// And the inverse: starting from 0x00 the byte stays 0x00.
				op.set(0x00)
				mb.Cpu.Registers.F = 0x00
				mb.Cpu.Registers.PC = 0xC050
				cycles2 := OPCODES[key](mb, 0)
				assert.Equalf(t, uint8(0x00), op.get(),
					"RES %d,%s in=0x00: must remain 0", bit, name)
				assert.Equalf(t, uint8(0x00), mb.Cpu.Registers.F,
					"RES %d,%s in=0x00: flags untouched", bit, name)
				assert.Equalf(t, uint16(0xC052), mb.Cpu.Registers.PC,
					"RES %d,%s in=0x00: PC", bit, name)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles2,
					"RES %d,%s in=0x00: cycles", bit, name)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// SET b, r — set bit b of operand. NO flag change.
// Opcode keys: 0x1C0 + b*8 + reg.
// ---------------------------------------------------------------------------

func TestCB_SET(t *testing.T) {
	for bit := uint8(0); bit < 8; bit++ {
		for regIdx, name := range cbRegisterOrder {
			t.Run(fmt.Sprintf("b%d/%s", bit, name), func(t *testing.T) {
				_, mb := newTestCPU(t)
				op, oldPC := cbResetForOp(t, mb, name)
				op.set(0x00) // every bit clear; expect (1<<bit) afterwards
				// Seed every flag so we can prove SET leaves them alone.
				mb.Cpu.SetFlagZ()
				mb.Cpu.SetFlagN()
				mb.Cpu.SetFlagH()
				mb.Cpu.SetFlagC()
				flagsBefore := mb.Cpu.Registers.F

				key := OpCode(0x1C0 + OpCode(bit)*8 + OpCode(regIdx))
				cycles := OPCODES[key](mb, 0)
				cbMark(key)

				wantValue := uint8(1 << bit)
				assert.Equalf(t, wantValue, op.get(),
					"SET %d,%s in=0x00: result", bit, name)
				assert.Equalf(t, flagsBefore, mb.Cpu.Registers.F,
					"SET %d,%s: flags must be unchanged", bit, name)
				assert.Equalf(t, oldPC+2, mb.Cpu.Registers.PC,
					"SET %d,%s: PC", bit, name)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles,
					"SET %d,%s: cycles", bit, name)

				// And the inverse: 0xFF stays 0xFF after setting any bit.
				op.set(0xFF)
				mb.Cpu.Registers.F = 0x00
				mb.Cpu.Registers.PC = 0xC050
				cycles2 := OPCODES[key](mb, 0)
				assert.Equalf(t, uint8(0xFF), op.get(),
					"SET %d,%s in=0xFF: must remain 0xFF", bit, name)
				assert.Equalf(t, uint8(0x00), mb.Cpu.Registers.F,
					"SET %d,%s in=0xFF: flags untouched", bit, name)
				assert.Equalf(t, uint16(0xC052), mb.Cpu.Registers.PC,
					"SET %d,%s in=0xFF: PC", bit, name)
				assert.Equalf(t, cbExpectedCycles(name, false), cycles2,
					"SET %d,%s in=0xFF: cycles", bit, name)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Coverage failsafes.
//
// TestCB_AllOpcodesPresentInMap checks the OPCODES map statically — every
// CB opcode key in 0x100..0x1FF must have a non-nil handler installed.
//
// TestCB_zCoverage_AllOpcodes verifies the per-mnemonic test functions
// above collectively drive all 256 distinct CB opcodes (no gaps, no
// duplicates that would mask a missing entry).  The trailing "z" in the
// name pushes it last in the sorted-test-name order so cbExercised has
// been fully populated by the other TestCB_* functions when this runs.
// ---------------------------------------------------------------------------

func TestCB_AllOpcodesPresentInMap(t *testing.T) {
	missing := []OpCode{}
	for op := OpCode(0x100); op <= 0x1FF; op++ {
		fn, ok := OPCODES[op]
		if !ok || fn == nil {
			missing = append(missing, op)
		}
	}
	require.Empty(t, missing, "CB opcodes missing/nil in OPCODES map: %#v", missing)
}

func TestCB_zCoverage_AllOpcodes(t *testing.T) {
	// All 256 expected CB opcode keys.
	want := map[OpCode]struct{}{}
	for op := OpCode(0x100); op <= 0x1FF; op++ {
		want[op] = struct{}{}
	}

	cbExercisedMu.Lock()
	defer cbExercisedMu.Unlock()

	missing := []OpCode{}
	for op := range want {
		if _, ok := cbExercised[op]; !ok {
			missing = append(missing, op)
		}
	}
	extra := []OpCode{}
	for op := range cbExercised {
		if _, ok := want[op]; !ok {
			extra = append(extra, op)
		}
	}

	assert.Empty(t, missing, "CB opcodes never exercised: %#v", missing)
	assert.Empty(t, extra, "non-CB opcodes accidentally exercised: %#v", extra)
	assert.Equal(t, 256, len(cbExercised),
		"expected exactly 256 distinct CB opcodes exercised, got %d", len(cbExercised))
}
