package opcodes

import (
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cpu"
)

type Motherboard interface {
	SetItem(addr *uint16, value *uint16)
	Cpu() *cpu.Cpu
}

type OpLogic func(mb Motherboard, value uint16) uint8

// OPCODES is a map of opcodes to their logic
var OPCODES = map[uint16]OpLogic{

	// NOP - No operation
	0x00: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		return 4
	},

	// LD BC, d16 - Load 16-bit immediate into BC
	0x01: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.SetBC(value)
		c.Registers.PC += 3
		return 12
	},

	// LD (BC), A - Save A to address pointed by BC
	0x02: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		bc := c.BC()
		a := (uint16)(c.Registers.A)
		mb.SetItem(&bc, &a)
		c.Registers.PC += 1
		return 8
	},

	// // INC BC - Increment BC
	0x03: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		bc := c.BC()
		bc += 1
		c.SetBC(bc)
		c.Registers.PC += 1
		return 8
	},

	// INC B - Increment B
	0x04: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B + 1

		if b == 0 {
			c.SetFlagZ()
		}

		if internal.HalfCarryTest(&b) {
			c.SetFlagH()
		}
		c.ResetFlagN()
		c.Registers.B = b
		c.Registers.PC += 1
		return 4
	},

	// DEC B - Decrement B
	0x05: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B - 1

		if b == 0 {
			c.SetFlagZ()
		}

		if internal.HalfCarryTest(&b) {
			c.SetFlagH()
		}
		c.SetFlagN()
		c.Registers.B = b
		c.Registers.PC += 1
		return 4
	},
}
