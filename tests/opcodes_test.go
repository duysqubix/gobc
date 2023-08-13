package tests

import (
	"fmt"
	"testing"

	"github.com/duysqubix/gobc/internal/cpu"
	"github.com/duysqubix/gobc/internal/opcodes"
)

type testMotherboard struct {
	c      *cpu.Cpu
	memory []uint16
}

func NewTestMotherboard() *testMotherboard {
	return &testMotherboard{
		c:      cpu.NewCpu(),
		memory: make([]uint16, 0xFFFF),
	}
}

func (mb *testMotherboard) SetItem(addr *uint16, value *uint16) {
	mb.memory[*addr] = *value
}

func (mb *testMotherboard) GetItem(addr *uint16) uint16 {
	return mb.memory[*addr]
}

func (mb *testMotherboard) Cpu() *cpu.Cpu {
	return mb.c
}

func Test_LD00(t *testing.T) {
	mb := NewTestMotherboard()

	op, ok := opcodes.OPCODES[0x00]
	if !ok {
		t.Fatalf("Opcode 0x00 not found")
	}

	expectedCycles := uint8(4)
	expectedPC := uint16(1)
	cycles := op(mb, 0)

	if cycles != expectedCycles {
		t.Errorf("Returned %d cycles, want %d", cycles, expectedCycles)
	}

	if mb.c.Registers.PC != expectedPC {
		t.Errorf("PC register is %d, want %d", mb.c.Registers.PC, expectedPC)
	}

}

func Test_LD01(t *testing.T) {
	mb := NewTestMotherboard()

	op, ok := opcodes.OPCODES[0x01]
	if !ok {
		t.Fatalf("Opcode 0x01 not found")
	}

	value := uint16(12345)
	expectedCycles := uint8(12)
	expectedPC := uint16(3)

	cycles := op(mb, value)

	// Check the number of cycles.
	if cycles != expectedCycles {
		t.Errorf("Returned %d cycles, want %d", cycles, expectedCycles)
	}

	// Check the BC register.
	if mb.c.BC() != value {
		t.Errorf("BC register is %d, want %d", mb.c.BC(), value)
	}

	// Check the PC register.
	if mb.c.Registers.PC != expectedPC {
		t.Errorf("PC register is %d, want %d", mb.c.Registers.PC, expectedPC)
	}
}

func Test_LD02(t *testing.T) {
	mb := NewTestMotherboard()

	op, ok := opcodes.OPCODES[0x02]
	if !ok {
		t.Fatalf("Opcode 0x02 not found")
	}

	expectedCycles := uint8(8)
	expectedPC := uint16(1)

	mb.c.Registers.B = 0x12
	mb.c.Registers.C = 0x30

	mb.c.Registers.A = 0xBE

	cycles := op(mb, 0)

	if cycles != expectedCycles {
		t.Errorf("Returned %d cycles, want %d", cycles, expectedCycles)
	}

	if mb.c.Registers.PC != expectedPC {
		t.Errorf("PC register is %d, want %d", mb.c.Registers.PC, expectedPC)
	}

	if mb.memory[mb.c.BC()] != uint16(mb.c.Registers.A) {
		t.Errorf("Memory at %#x is %#x, want %#x", mb.c.BC(), mb.memory[mb.c.BC()], uint16(mb.c.Registers.A))
	}
}

func Test_LD03(t *testing.T) {
	mb := NewTestMotherboard()

	op, ok := opcodes.OPCODES[0x03]
	if !ok {
		t.Fatalf("Opcode 0x03 not found")
	}

	expectedCycles := uint8(8)
	expectedPC := uint16(1)

	mb.c.Registers.B = 0x12
	mb.c.Registers.C = 0x34

	cycles := op(mb, 0)

	if cycles != expectedCycles {
		t.Errorf("Returned %d cycles, want %d", cycles, expectedCycles)
	}

	if mb.c.Registers.PC != expectedPC {
		t.Errorf("PC register is %d, want %d", mb.c.Registers.PC, expectedPC)
	}

	if mb.c.BC() != 0x1235 {
		t.Errorf("BC register is %#x, want %#x", mb.c.BC(), 0x1235)
	}
}

func Test_LD04(t *testing.T) {
	mb := NewTestMotherboard()

	op, ok := opcodes.OPCODES[0x04]
	if !ok {
		t.Fatalf("Opcode 0x04 not found")
	}

	expectedCycles := uint8(4)
	expectedPC := uint16(1)

	mb.c.Registers.B = 0x12

	cycles := op(mb, 0)

	if cycles != expectedCycles {
		t.Errorf("Returned %d cycles, want %d", cycles, expectedCycles)
	}

	if mb.c.Registers.B != 0x13 {
		t.Errorf("B register is %#x, want %#x", mb.c.Registers.B, 0x13)
	}

	if mb.c.Registers.PC != expectedPC {
		t.Errorf("PC register is %d, want %d", mb.c.Registers.PC, expectedPC)
	}

	fmt.Printf("----%#b-----", mb.c.Registers.F)
	//check flags
	if !mb.c.IsFlagZSet() {
		t.Errorf("Z flag is not set")
	}

	if !mb.c.IsFlagHSet() {
		t.Errorf("H flag is not set")
	}

	if mb.c.IsFlagNSet() {
		t.Errorf("N flag is set")
	}
}

// func Test_LD05(t *testing.T) {
// 	mb := NewTestMotherboard()

// 	op, ok := opcodes.OPCODES[0x05]
// 	if !ok {
// 		t.Fatalf("Opcode 0x05 not found")
// 	}

// 	expectedCycles := uint8(4)
// 	expectedPC := uint16(1)

// 	mb.c.Registers.B = 0x00

// 	cycles := op(mb, 0)

// 	if cycles != expectedCycles {
// 		t.Errorf("Returned %d cycles, want %d", cycles, expectedCycles)
// 	}

// 	if mb.c.Registers.B != 0xFF {
// 		t.Errorf("B register is %#x, want %#x", mb.c.Registers.B, 0xFF)
// 	}

// 	if mb.c.Registers.PC != expectedPC {
// 		t.Errorf("PC register is %d, want %d", mb.c.Registers.PC, expectedPC)
// 	}

// 	//check flags
// 	if !mb.c.IsFlagZSet() {
// 		t.Errorf("Z flag is not set")
// 	}

// 	if !mb.c.IsFlagHSet() {
// 		t.Errorf("H flag is not set")
// 	}

// 	if !mb.c.IsFlagNSet() {
// 		t.Errorf("N flag is not set")
// 	}
// }
