package opcodes

import (
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

		addr := (uint16)(0xbeef)
		mb.SetItem(&addr, &value)
		return 4
	},

	// LD BC, d16 - Load 16-bit immediate into BC
	0x01: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.SetBC(value)
		c.Registers.PC += 3
		return 12
	},

	// // LD (BC), A - Save A to address pointed by BC
	0x02: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		bc := ((uint16)(c.Registers.B) << 8) | (uint16)(c.Registers.C)
		a := (uint16)(c.Registers.A)
		mb.SetItem(&bc, &a)
		c.Registers.PC += 1
		return 8
	},
}
