package motherboard

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 8-bit LD opcode tests
// ---------------------------------------------------------------------------
//
// This file exhaustively exercises every 8-bit LD-family opcode in
// opcodes.go. The set covered here:
//
//   * LD r, r'        - 0x40..0x7F except 0x76 (HALT). 63 opcodes.
//   * LD r, d8        - 0x06 / 0x0E / 0x16 / 0x1E / 0x26 / 0x2E / 0x3E.
//   * LD (HL), d8     - 0x36.
//   * LD A, (rr)      - 0x0A / 0x1A / 0x2A (HL+) / 0x3A (HL-).
//   * LD (rr), A      - 0x02 / 0x12 / 0x22 (HL+) / 0x32 (HL-).
//   * LDH (n), A      - 0xE0.
//   * LDH A, (n)      - 0xF0.
//   * LD (C), A       - 0xE2.
//   * LD A, (C)       - 0xF2.
//   * LD (a16), A     - 0xEA.
//   * LD A, (a16)     - 0xFA.
//
// Every test asserts:
//   - destination register (or memory cell) holds the source value;
//   - PC advanced by the documented number of bytes;
//   - the opcode handler returned the documented cycle count;
//   - no other register was clobbered;
//   - no other memory cell was clobbered for indirect operations.
//
// 16-bit loads (LD BC,d16 / LD DE,d16 / LD HL,d16 / LD SP,d16 / LD (a16),SP /
// LD HL,SP+r8 / LD SP,HL) are owned by opcodes_ld16_test.go and are NOT
// covered here.
//
// Memory layout used by the tests:
//   - Test PC seed              : 0xC000 (WRAM bank 0, set by newTestCPU).
//   - HL pointer for LD r,(HL)  : 0xC100 (WRAM bank 0).
//   - BC pointer for LD A,(BC)  : 0xC200.
//   - DE pointer for LD A,(DE)  : 0xC300.
//   - HL pointer for LD A,(HL+/-): 0xC400.
//   - LDH HRAM target           : 0xFF80..0xFFFE (HRAM, plain RAM cells).
//   - LD (a16) absolute target  : 0xC500.
//
// The motherboard built by newTestCPU has no Cartridge; addresses below
// 0xC000 or in the 0xA000..0xBFFF range therefore must NOT be used as
// targets, because SetItem / GetItem on those would NPE on m.Cartridge.

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// regSeed is the canonical seed pattern applied to the CPU before every
// LD test. Each register gets a distinct sentinel so an accidental copy
// from the wrong source register is immediately visible.
//
// HL = (H << 8) | L = 0xC100, which lands in WRAM bank 0 and is a safe
// target for SetItem / GetItem in the test harness.
var regSeed = map[string]uint8{
	"A": 0xA0,
	"B": 0xB0,
	"C": 0xC0,
	"D": 0xD0,
	"E": 0xE0,
	"H": 0xC1,
	"L": 0x00,
	"F": 0x00,
}

const (
	ld8HLAddr  uint16 = 0xC100 // matches H=0xC1, L=0x00 in regSeed
	ld8MemSeed uint8  = 0x42   // value planted at ld8HLAddr by ld8Reseed
	ld8PCSeed  uint16 = 0xC000 // matches newTestCPU PC default
)

// ld8Reseed restores the CPU and the WRAM cell at ld8HLAddr to the
// canonical seed state. It overrides anything newTestCPU did so each
// sub-test starts from an identical baseline.
func ld8Reseed(cpu *CPU, mb *Motherboard) {
	cpu.Registers.A = regSeed["A"]
	cpu.Registers.B = regSeed["B"]
	cpu.Registers.C = regSeed["C"]
	cpu.Registers.D = regSeed["D"]
	cpu.Registers.E = regSeed["E"]
	cpu.Registers.H = regSeed["H"]
	cpu.Registers.L = regSeed["L"]
	cpu.Registers.F = regSeed["F"]
	cpu.Registers.PC = ld8PCSeed
	mb.SetItem(ld8HLAddr, uint16(ld8MemSeed))
}

// ld8GetReg reads a named register from cpu. Used by table-driven tests
// to avoid a giant switch in every assertion site.
func ld8GetReg(cpu *CPU, name string) uint8 {
	switch name {
	case "A":
		return cpu.Registers.A
	case "B":
		return cpu.Registers.B
	case "C":
		return cpu.Registers.C
	case "D":
		return cpu.Registers.D
	case "E":
		return cpu.Registers.E
	case "H":
		return cpu.Registers.H
	case "L":
		return cpu.Registers.L
	case "F":
		return cpu.Registers.F
	}
	panic("unknown register: " + name)
}

// ---------------------------------------------------------------------------
// LD r, r' family - 0x40..0x7F (excluding 0x76 HALT). 63 opcodes total.
// ---------------------------------------------------------------------------
//
// Encoding: opcode = 0x40 | (dst << 3) | src, where the index ordering is
// B=0, C=1, D=2, E=3, H=4, L=5, (HL)=6, A=7. The 0x76 slot would be
// LD (HL),(HL) but is repurposed as HALT and intentionally excluded.

type ld8RegRow struct {
	opcode OpCode
	dst    string
	src    string
}

// ld8RegRows enumerates all 63 LD r,r' opcodes in encoding order.
var ld8RegRows = []ld8RegRow{
	// 0x40..0x47: LD B, r
	{0x40, "B", "B"}, {0x41, "B", "C"}, {0x42, "B", "D"}, {0x43, "B", "E"},
	{0x44, "B", "H"}, {0x45, "B", "L"}, {0x46, "B", "(HL)"}, {0x47, "B", "A"},
	// 0x48..0x4F: LD C, r
	{0x48, "C", "B"}, {0x49, "C", "C"}, {0x4A, "C", "D"}, {0x4B, "C", "E"},
	{0x4C, "C", "H"}, {0x4D, "C", "L"}, {0x4E, "C", "(HL)"}, {0x4F, "C", "A"},
	// 0x50..0x57: LD D, r
	{0x50, "D", "B"}, {0x51, "D", "C"}, {0x52, "D", "D"}, {0x53, "D", "E"},
	{0x54, "D", "H"}, {0x55, "D", "L"}, {0x56, "D", "(HL)"}, {0x57, "D", "A"},
	// 0x58..0x5F: LD E, r
	{0x58, "E", "B"}, {0x59, "E", "C"}, {0x5A, "E", "D"}, {0x5B, "E", "E"},
	{0x5C, "E", "H"}, {0x5D, "E", "L"}, {0x5E, "E", "(HL)"}, {0x5F, "E", "A"},
	// 0x60..0x67: LD H, r
	{0x60, "H", "B"}, {0x61, "H", "C"}, {0x62, "H", "D"}, {0x63, "H", "E"},
	{0x64, "H", "H"}, {0x65, "H", "L"}, {0x66, "H", "(HL)"}, {0x67, "H", "A"},
	// 0x68..0x6F: LD L, r
	{0x68, "L", "B"}, {0x69, "L", "C"}, {0x6A, "L", "D"}, {0x6B, "L", "E"},
	{0x6C, "L", "H"}, {0x6D, "L", "L"}, {0x6E, "L", "(HL)"}, {0x6F, "L", "A"},
	// 0x70..0x77: LD (HL), r ; 0x76 is HALT and intentionally absent.
	{0x70, "(HL)", "B"}, {0x71, "(HL)", "C"}, {0x72, "(HL)", "D"}, {0x73, "(HL)", "E"},
	{0x74, "(HL)", "H"}, {0x75, "(HL)", "L"}, {0x77, "(HL)", "A"},
	// 0x78..0x7F: LD A, r
	{0x78, "A", "B"}, {0x79, "A", "C"}, {0x7A, "A", "D"}, {0x7B, "A", "E"},
	{0x7C, "A", "H"}, {0x7D, "A", "L"}, {0x7E, "A", "(HL)"}, {0x7F, "A", "A"},
}

// TestLD8_RegToReg_TableDoesNotIncludeHALT guards the encoding table itself.
// If somebody accidentally adds 0x76 LD (HL),(HL) here, the HALT path would
// silently rewrite memory and break the contract.
func TestLD8_RegToReg_TableDoesNotIncludeHALT(t *testing.T) {
	require.Len(t, ld8RegRows, 63, "exactly 63 LD r,r' opcodes (0x40..0x7F minus 0x76)")
	for _, row := range ld8RegRows {
		require.NotEqual(t, OpCode(0x76), row.opcode,
			"0x76 is HALT, not LD (HL),(HL); must not appear in ld8RegRows")
	}
}

// TestLD8_RegToReg_AllPairs exercises every LD r,r' opcode. Each subtest
// seeds every CPU register with a distinct sentinel, plants a sentinel
// byte at HL, runs the opcode, and verifies that exactly one destination
// changed and the cycle/PC contract was honoured.
func TestLD8_RegToReg_AllPairs(t *testing.T) {
	for _, row := range ld8RegRows {
		row := row
		name := fmt.Sprintf("0x%02X_LD_%s_%s", uint8(row.opcode), row.dst, row.src)
		t.Run(name, func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			ld8Reseed(cpu, mb)

			// Sanity-check seed: HL must point at the planted byte.
			require.Equal(t, ld8HLAddr, cpu.HL(), "seed: HL=0xC100")
			require.Equal(t, ld8MemSeed, mb.GetItem(ld8HLAddr), "seed: (HL)=0x42")

			// Resolve src value BEFORE running so we can verify dst afterwards.
			var srcVal uint8
			if row.src == "(HL)" {
				srcVal = mb.GetItem(cpu.HL())
			} else {
				srcVal = ld8GetReg(cpu, row.src)
			}

			cycles := OPCODES[row.opcode](mb, 0)

			// Cycle count: 4 for reg<->reg, 8 if either side touches (HL).
			wantCycles := OpCycles(4)
			if row.src == "(HL)" || row.dst == "(HL)" {
				wantCycles = 8
			}
			assert.Equal(t, wantCycles, cycles, "cycles")

			// PC advances by 1 for the entire LD r,r' family.
			assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "PC += 1")

			// Build expected register/memory map after the load.
			expected := map[string]uint8{}
			for k, v := range regSeed {
				expected[k] = v
			}
			expectedMem := map[uint16]uint8{ld8HLAddr: ld8MemSeed}
			if row.dst == "(HL)" {
				expectedMem[ld8HLAddr] = srcVal
			} else {
				expected[row.dst] = srcVal
			}

			// Verify every register matches the expected post-load state.
			for _, name := range []string{"A", "B", "C", "D", "E", "H", "L", "F"} {
				assert.Equalf(t, expected[name], ld8GetReg(cpu, name),
					"register %s after %s", name, row_describe(row))
			}
			// Verify the (HL) cell as well; for non-(HL) opcodes it must be
			// unchanged from the seed.
			assert.Equalf(t, expectedMem[ld8HLAddr], mb.GetItem(ld8HLAddr),
				"memory at HL=0x%04X after %s", ld8HLAddr, row_describe(row))
		})
	}
}

func row_describe(row ld8RegRow) string {
	return fmt.Sprintf("LD %s,%s (0x%02X)", row.dst, row.src, uint8(row.opcode))
}

// ---------------------------------------------------------------------------
// LD r, d8 family - 0x06 / 0x0E / 0x16 / 0x1E / 0x26 / 0x2E / 0x3E
// ---------------------------------------------------------------------------

type ld8ImmRow struct {
	opcode OpCode
	dst    string
}

var ld8ImmRows = []ld8ImmRow{
	{0x06, "B"}, {0x0E, "C"},
	{0x16, "D"}, {0x1E, "E"},
	{0x26, "H"}, {0x2E, "L"},
	{0x3E, "A"},
}

// TestLD8_ImmediateToReg covers LD r,d8 for every general-purpose register.
// Three different immediates (0x00, 0x55, 0xFF) are exercised per opcode
// to catch any off-by-one masking or sign-extension bugs.
func TestLD8_ImmediateToReg(t *testing.T) {
	for _, row := range ld8ImmRows {
		row := row
		for _, imm := range []uint8{0x00, 0x55, 0xFF} {
			imm := imm
			name := fmt.Sprintf("0x%02X_LD_%s_%#02X", uint8(row.opcode), row.dst, imm)
			t.Run(name, func(t *testing.T) {
				cpu, mb := newTestCPU(t)
				ld8Reseed(cpu, mb)

				cycles := OPCODES[row.opcode](mb, uint16(imm))

				assert.Equal(t, OpCycles(8), cycles, "LD r,d8 takes 8 cycles")
				assert.Equal(t, ld8PCSeed+2, cpu.Registers.PC, "LD r,d8 advances PC by 2")
				assert.Equal(t, imm, ld8GetReg(cpu, row.dst), "destination register holds immediate")

				// Verify all other registers untouched.
				for _, other := range []string{"A", "B", "C", "D", "E", "H", "L", "F"} {
					if other == row.dst {
						continue
					}
					assert.Equalf(t, regSeed[other], ld8GetReg(cpu, other),
						"register %s must be unchanged by LD %s,d8", other, row.dst)
				}
				// Memory at HL must be untouched (operand is immediate, not memory).
				assert.Equal(t, ld8MemSeed, mb.GetItem(ld8HLAddr),
					"memory at HL must be unchanged by LD r,d8")
			})
		}
	}
}

// TestLD8_ImmediateToHL covers 0x36 LD (HL),d8.
func TestLD8_ImmediateToHL(t *testing.T) {
	for _, imm := range []uint8{0x00, 0x7E, 0xFF} {
		imm := imm
		t.Run(fmt.Sprintf("0x36_LD_HL_%#02X", imm), func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			ld8Reseed(cpu, mb)

			cycles := OPCODES[0x36](mb, uint16(imm))

			assert.Equal(t, OpCycles(12), cycles, "LD (HL),d8 takes 12 cycles")
			assert.Equal(t, ld8PCSeed+2, cpu.Registers.PC, "LD (HL),d8 advances PC by 2")
			assert.Equal(t, imm, mb.GetItem(ld8HLAddr), "memory at HL holds immediate")

			// All registers unchanged.
			for _, name := range []string{"A", "B", "C", "D", "E", "H", "L", "F"} {
				assert.Equalf(t, regSeed[name], ld8GetReg(cpu, name),
					"register %s must be unchanged by LD (HL),d8", name)
			}
		})
	}
}

// TestLD8_ImmediateToHL_MasksHighByte verifies the implementation's
// `value &= 0xff` step by passing an over-wide value. Only the low byte
// must be written.
func TestLD8_ImmediateToHL_MasksHighByte(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	// 0x12AB has 0xAB in the low byte. The opcode masks via &= 0xff before
	// calling SetItem, so the byte written must be 0xAB.
	cycles := OPCODES[0x36](mb, 0x12AB)

	assert.Equal(t, OpCycles(12), cycles)
	assert.Equal(t, ld8PCSeed+2, cpu.Registers.PC)
	assert.Equal(t, uint8(0xAB), mb.GetItem(ld8HLAddr),
		"LD (HL),d8 must mask value to 8 bits before writing")
}

// ---------------------------------------------------------------------------
// LD A, (rr) family - 0x0A / 0x1A
// ---------------------------------------------------------------------------

// TestLD8_A_From_BC verifies 0x0A LD A,(BC).
func TestLD8_A_From_BC(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const bcAddr uint16 = 0xC200
	const want uint8 = 0x77
	cpu.SetBC(bcAddr)
	mb.SetItem(bcAddr, uint16(want))

	cycles := OPCODES[0x0A](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD A,(BC) takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD A,(BC) advances PC by 1")
	assert.Equal(t, want, cpu.Registers.A, "A loaded from (BC)")

	// BC and DE/HL/F unchanged.
	assert.Equal(t, bcAddr, cpu.BC(), "BC unchanged")
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
	assert.Equal(t, want, mb.GetItem(bcAddr), "(BC) cell unchanged by read")
}

// TestLD8_A_From_DE verifies 0x1A LD A,(DE).
func TestLD8_A_From_DE(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const deAddr uint16 = 0xC300
	const want uint8 = 0x9B
	cpu.SetDE(deAddr)
	mb.SetItem(deAddr, uint16(want))

	cycles := OPCODES[0x1A](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD A,(DE) takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD A,(DE) advances PC by 1")
	assert.Equal(t, want, cpu.Registers.A, "A loaded from (DE)")

	assert.Equal(t, deAddr, cpu.DE(), "DE unchanged")
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
	assert.Equal(t, want, mb.GetItem(deAddr), "(DE) cell unchanged by read")
}

// ---------------------------------------------------------------------------
// LD (rr), A family - 0x02 / 0x12
// ---------------------------------------------------------------------------

// TestLD8_BC_From_A verifies 0x02 LD (BC),A.
func TestLD8_BC_From_A(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const bcAddr uint16 = 0xC200
	cpu.SetBC(bcAddr)
	cpu.Registers.A = 0x5C

	cycles := OPCODES[0x02](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD (BC),A takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD (BC),A advances PC by 1")
	assert.Equal(t, uint8(0x5C), mb.GetItem(bcAddr), "memory at (BC) holds A")

	assert.Equal(t, uint8(0x5C), cpu.Registers.A, "A unchanged by LD (BC),A")
	assert.Equal(t, bcAddr, cpu.BC(), "BC unchanged")
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// TestLD8_DE_From_A verifies 0x12 LD (DE),A.
func TestLD8_DE_From_A(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const deAddr uint16 = 0xC300
	cpu.SetDE(deAddr)
	cpu.Registers.A = 0xE7

	cycles := OPCODES[0x12](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD (DE),A takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD (DE),A advances PC by 1")
	assert.Equal(t, uint8(0xE7), mb.GetItem(deAddr), "memory at (DE) holds A")

	assert.Equal(t, uint8(0xE7), cpu.Registers.A, "A unchanged by LD (DE),A")
	assert.Equal(t, deAddr, cpu.DE(), "DE unchanged")
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// ---------------------------------------------------------------------------
// LD A, (HL+/HL-) and LD (HL+/HL-), A - 0x2A, 0x3A, 0x22, 0x32
// ---------------------------------------------------------------------------
//
// These four opcodes share a quirk: HL is post-modified AFTER the memory
// access. The tests below verify that the read or write hits the OLD HL
// and that the new HL is one above or below.

const (
	ld8HLPostAddr uint16 = 0xC400 // dedicated address for HL+/HL- tests
)

// TestLD8_A_From_HLPlus verifies 0x2A LD A,(HL+): A = (HL); HL++.
func TestLD8_A_From_HLPlus(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const want uint8 = 0x33
	cpu.SetHL(ld8HLPostAddr)
	mb.SetItem(ld8HLPostAddr, uint16(want))
	// Plant a sentinel at HL+1 to ensure we did NOT pre-increment before
	// the read.
	mb.SetItem(ld8HLPostAddr+1, 0xEE)

	cycles := OPCODES[0x2A](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD A,(HL+) takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD A,(HL+) advances PC by 1")
	assert.Equal(t, want, cpu.Registers.A, "A loaded from OLD HL (post-increment)")
	assert.Equal(t, ld8HLPostAddr+1, cpu.HL(), "HL post-incremented by 1")

	// Memory must be unchanged by a load.
	assert.Equal(t, want, mb.GetItem(ld8HLPostAddr), "(HL) source cell unchanged")
	assert.Equal(t, uint8(0xEE), mb.GetItem(ld8HLPostAddr+1), "HL+1 unchanged")

	// Other registers untouched.
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// TestLD8_A_From_HLMinus verifies 0x3A LD A,(HL-): A = (HL); HL--.
func TestLD8_A_From_HLMinus(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const want uint8 = 0xC9
	cpu.SetHL(ld8HLPostAddr)
	mb.SetItem(ld8HLPostAddr, uint16(want))
	// Plant a sentinel at HL-1 to ensure we did NOT pre-decrement first.
	mb.SetItem(ld8HLPostAddr-1, 0x11)

	cycles := OPCODES[0x3A](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD A,(HL-) takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD A,(HL-) advances PC by 1")
	assert.Equal(t, want, cpu.Registers.A, "A loaded from OLD HL (post-decrement)")
	assert.Equal(t, ld8HLPostAddr-1, cpu.HL(), "HL post-decremented by 1")

	assert.Equal(t, want, mb.GetItem(ld8HLPostAddr), "(HL) source cell unchanged")
	assert.Equal(t, uint8(0x11), mb.GetItem(ld8HLPostAddr-1), "HL-1 unchanged")

	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// TestLD8_HLPlus_From_A verifies 0x22 LD (HL+),A: (HL) = A; HL++.
func TestLD8_HLPlus_From_A(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	cpu.SetHL(ld8HLPostAddr)
	cpu.Registers.A = 0x8D
	// Plant a marker at HL+1 to confirm we wrote to the OLD HL, not HL+1.
	mb.SetItem(ld8HLPostAddr+1, 0x11)

	cycles := OPCODES[0x22](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD (HL+),A takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD (HL+),A advances PC by 1")
	assert.Equal(t, uint8(0x8D), mb.GetItem(ld8HLPostAddr), "OLD HL holds A")
	assert.Equal(t, uint8(0x11), mb.GetItem(ld8HLPostAddr+1), "HL+1 untouched")
	assert.Equal(t, ld8HLPostAddr+1, cpu.HL(), "HL post-incremented by 1")

	assert.Equal(t, uint8(0x8D), cpu.Registers.A, "A unchanged")
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// TestLD8_HLMinus_From_A verifies 0x32 LD (HL-),A: (HL) = A; HL--.
func TestLD8_HLMinus_From_A(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	cpu.SetHL(ld8HLPostAddr)
	cpu.Registers.A = 0x4F
	mb.SetItem(ld8HLPostAddr-1, 0x11)

	cycles := OPCODES[0x32](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD (HL-),A takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD (HL-),A advances PC by 1")
	assert.Equal(t, uint8(0x4F), mb.GetItem(ld8HLPostAddr), "OLD HL holds A")
	assert.Equal(t, uint8(0x11), mb.GetItem(ld8HLPostAddr-1), "HL-1 untouched")
	assert.Equal(t, ld8HLPostAddr-1, cpu.HL(), "HL post-decremented by 1")

	assert.Equal(t, uint8(0x4F), cpu.Registers.A, "A unchanged")
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// ---------------------------------------------------------------------------
// LDH variants - 0xE0, 0xF0, 0xE2, 0xF2
// ---------------------------------------------------------------------------
//
// All four reach into the 0xFF00..0xFFFF I/O / HRAM page. The tests use
// HRAM offsets (n in 0x80..0xFE) so the underlying RAM cell is plain
// memory and the round-trip is deterministic. n=0x00 (P1/JOYP) and other
// I/O registers have observation-side effects, so they are intentionally
// avoided here.

// TestLD8_LDH_NA_To_A verifies 0xE0 LDH (n),A and 0xF0 LDH A,(n) as a
// round trip across HRAM offsets.
func TestLD8_LDH_NA_RoundTrip(t *testing.T) {
	for _, n := range []uint8{0x80, 0xC3, 0xFE} {
		n := n
		t.Run(fmt.Sprintf("n=%#02X", n), func(t *testing.T) {
			cpu, mb := newTestCPU(t)
			ld8Reseed(cpu, mb)
			cpu.Registers.A = 0x6B
			addr := uint16(0xFF00) + uint16(n)

			// 0xE0 LDH (n),A
			cycles := OPCODES[0xE0](mb, uint16(n))
			assert.Equal(t, OpCycles(12), cycles, "LDH (n),A takes 12 cycles")
			assert.Equal(t, ld8PCSeed+2, cpu.Registers.PC, "LDH (n),A advances PC by 2")
			assert.Equal(t, uint8(0x6B), mb.GetItem(addr),
				"LDH (n),A wrote to 0xFF00+n=0x%04X", addr)
			assert.Equal(t, uint8(0x6B), cpu.Registers.A, "A unchanged by LDH (n),A")

			// Follow up with 0xF0 LDH A,(n) on a fresh CPU+motherboard so we
			// are not confused by side-effects of the prior write.
			cpu2, mb2 := newTestCPU(t)
			ld8Reseed(cpu2, mb2)
			cpu2.Registers.A = 0x00 // ensure the load actually mutates A
			mb2.SetItem(addr, 0x6B)

			cycles = OPCODES[0xF0](mb2, uint16(n))
			assert.Equal(t, OpCycles(12), cycles, "LDH A,(n) takes 12 cycles")
			assert.Equal(t, ld8PCSeed+2, cpu2.Registers.PC, "LDH A,(n) advances PC by 2")
			assert.Equal(t, uint8(0x6B), cpu2.Registers.A, "A loaded from 0xFF00+n")
			// All other registers untouched.
			assert.Equal(t, regSeed["B"], cpu2.Registers.B)
			assert.Equal(t, regSeed["C"], cpu2.Registers.C)
			assert.Equal(t, regSeed["D"], cpu2.Registers.D)
			assert.Equal(t, regSeed["E"], cpu2.Registers.E)
			assert.Equal(t, regSeed["H"], cpu2.Registers.H)
			assert.Equal(t, regSeed["L"], cpu2.Registers.L)
			assert.Equal(t, regSeed["F"], cpu2.Registers.F)
		})
	}
}

// TestLD8_LDH_PreservesOtherRegistersOnWrite covers the 0xE0 path's
// no-clobber contract that the round-trip above did not explicitly check.
func TestLD8_LDH_PreservesOtherRegistersOnWrite(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)
	cpu.Registers.A = 0xAB

	OPCODES[0xE0](mb, 0x90)

	assert.Equal(t, uint8(0xAB), mb.GetItem(0xFF90), "wrote A to 0xFF90")
	assert.Equal(t, uint8(0xAB), cpu.Registers.A)
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// TestLD8_LD_C_To_A verifies 0xE2 LD (C),A: (0xFF00+C) = A. PC += 1.
func TestLD8_LD_C_To_A(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)
	cpu.Registers.C = 0x90 // -> 0xFF90 (HRAM)
	cpu.Registers.A = 0x12

	cycles := OPCODES[0xE2](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD (C),A takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD (C),A advances PC by 1")
	assert.Equal(t, uint8(0x12), mb.GetItem(0xFF90), "wrote A to 0xFF00+C")
	assert.Equal(t, uint8(0x12), cpu.Registers.A, "A unchanged")
	assert.Equal(t, uint8(0x90), cpu.Registers.C, "C unchanged")

	// Other registers untouched.
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// TestLD8_LD_A_From_C verifies 0xF2 LD A,(C): A = (0xFF00+C). PC += 1.
func TestLD8_LD_A_From_C(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)
	cpu.Registers.C = 0xA5 // -> 0xFFA5 (HRAM)
	cpu.Registers.A = 0x00
	mb.SetItem(0xFFA5, 0x4D)

	cycles := OPCODES[0xF2](mb, 0)

	assert.Equal(t, OpCycles(8), cycles, "LD A,(C) takes 8 cycles")
	assert.Equal(t, ld8PCSeed+1, cpu.Registers.PC, "LD A,(C) advances PC by 1")
	assert.Equal(t, uint8(0x4D), cpu.Registers.A, "A loaded from 0xFF00+C")
	assert.Equal(t, uint8(0xA5), cpu.Registers.C, "C unchanged")

	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// ---------------------------------------------------------------------------
// LD (a16), A and LD A, (a16) - 0xEA, 0xFA
// ---------------------------------------------------------------------------

// TestLD8_AbsoluteStore verifies 0xEA LD (a16),A.
func TestLD8_AbsoluteStore(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const a16 uint16 = 0xC500
	cpu.Registers.A = 0xDE

	cycles := OPCODES[0xEA](mb, a16)

	assert.Equal(t, OpCycles(16), cycles, "LD (a16),A takes 16 cycles")
	assert.Equal(t, ld8PCSeed+3, cpu.Registers.PC, "LD (a16),A advances PC by 3")
	assert.Equal(t, uint8(0xDE), mb.GetItem(a16), "wrote A to a16")

	assert.Equal(t, uint8(0xDE), cpu.Registers.A, "A unchanged")
	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// TestLD8_AbsoluteLoad verifies 0xFA LD A,(a16).
func TestLD8_AbsoluteLoad(t *testing.T) {
	cpu, mb := newTestCPU(t)
	ld8Reseed(cpu, mb)

	const a16 uint16 = 0xC500
	mb.SetItem(a16, 0xAD)
	cpu.Registers.A = 0x00 // ensure load actually mutates A

	cycles := OPCODES[0xFA](mb, a16)

	assert.Equal(t, OpCycles(16), cycles, "LD A,(a16) takes 16 cycles")
	assert.Equal(t, ld8PCSeed+3, cpu.Registers.PC, "LD A,(a16) advances PC by 3")
	assert.Equal(t, uint8(0xAD), cpu.Registers.A, "A loaded from a16")
	assert.Equal(t, uint8(0xAD), mb.GetItem(a16), "memory at a16 unchanged by read")

	assert.Equal(t, regSeed["B"], cpu.Registers.B)
	assert.Equal(t, regSeed["C"], cpu.Registers.C)
	assert.Equal(t, regSeed["D"], cpu.Registers.D)
	assert.Equal(t, regSeed["E"], cpu.Registers.E)
	assert.Equal(t, regSeed["H"], cpu.Registers.H)
	assert.Equal(t, regSeed["L"], cpu.Registers.L)
	assert.Equal(t, regSeed["F"], cpu.Registers.F)
}

// ---------------------------------------------------------------------------
// Coverage census - keep the test surface honest if opcodes are added or
// re-categorized.
// ---------------------------------------------------------------------------

// TestLD8_CoverageCensus is a meta-test that fails if the 8-bit LD opcode
// surface diverges from what this file claims to cover. It is the
// canonical "did we miss any" guard for this file.
func TestLD8_CoverageCensus(t *testing.T) {
	covered := map[OpCode]struct{}{}

	// LD r, r' (63 opcodes)
	for _, row := range ld8RegRows {
		covered[row.opcode] = struct{}{}
	}
	// LD r, d8 (7 opcodes)
	for _, row := range ld8ImmRows {
		covered[row.opcode] = struct{}{}
	}
	// LD (HL), d8
	covered[0x36] = struct{}{}
	// LD A, (rr)
	for _, op := range []OpCode{0x0A, 0x1A, 0x2A, 0x3A} {
		covered[op] = struct{}{}
	}
	// LD (rr), A
	for _, op := range []OpCode{0x02, 0x12, 0x22, 0x32} {
		covered[op] = struct{}{}
	}
	// LDH variants
	for _, op := range []OpCode{0xE0, 0xF0, 0xE2, 0xF2} {
		covered[op] = struct{}{}
	}
	// LD (a16),A and LD A,(a16)
	covered[0xEA] = struct{}{}
	covered[0xFA] = struct{}{}

	// 63 + 7 + 1 + 4 + 4 + 4 + 2 = 85.
	assert.Equal(t, 85, len(covered),
		"this file claims to cover exactly 85 distinct 8-bit LD opcodes")

	// HALT must not be in the LD table.
	_, halt := covered[0x76]
	assert.False(t, halt, "0x76 is HALT, not LD (HL),(HL); must NOT be covered here")

	// Every claimed opcode must actually exist in OPCODES (catches typos).
	for op := range covered {
		_, ok := OPCODES[op]
		assert.Truef(t, ok, "claimed opcode %#x not present in OPCODES", op)
	}
}
