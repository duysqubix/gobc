package opcodes

import (
	"testing"

	"github.com/duysqubix/gobc/internal/cpu"
)

type testMotherboard struct {
	c *cpu.Cpu
}

func (mb *testMotherboard) SetItem(addr *uint16, value *uint16) {
	// This method is not used in opcode 0x01, so leave it empty.
}

func (mb *testMotherboard) Cpu() *cpu.Cpu {
	return mb.c
}

func Test_LD01(t *testing.T) {
	mb := &testMotherboard{
		c: cpu.NewCpu(),
	}

	op, ok := OPCODES[0x01]
	if !ok {
		t.Fatalf("Opcode 0x01 not found")
	}

	value := uint16(12345)
	expectedCycles := uint8(12)

	// Save the initial PC value for later.
	initialPC := mb.c.Registers.PC

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
	if mb.c.Registers.PC != initialPC+3 {
		t.Errorf("PC register is %d, want %d", mb.c.Registers.PC, initialPC+3)
	}
}
