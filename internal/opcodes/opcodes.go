package opcodes

import (
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cpu"
)

type Motherboard interface {
	SetItem(addr *uint16, value *uint16)
	GetItem(addr *uint16) uint8
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

	// // INC B - Increment B
	0x04: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Inc(c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// DEC B - Decrement B
	0x05: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Dec(c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// LD B, d8 - Load 8-bit immediate into B
	0x06: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// RLCA - Rotate A left. Old bit 7 to Carry flag
	0x07: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		a := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(a, 7) {
			c.SetFlagC()
			a = (a << 1) + 1
		} else {
			c.ResetFlagC()
			a = (a << 1)
		}

		c.Registers.A = a
		c.Registers.PC += 1
		return 4
	},

	// LD (a16), SP - Save SP at given address
	// value is the address
	0x08: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		addr1 := value
		value1 := c.Registers.SP & 0xFF

		addr2 := value + 1
		value2 := (c.Registers.SP >> 8) & 0xFF

		mb.SetItem(&addr1, &value1)
		mb.SetItem(&addr2, &value2)

		c.Registers.PC += 3
		return 20
	},

	// ADD HL, BC - Add BC to HL
	0x09: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		hl := c.AddSetFlags16(c.HL(), c.BC())

		c.SetHL(uint16(hl))
		c.Registers.PC += 1
		return 8
	},

	// LD A, (BC) - Load A from address pointed to by BC
	0x0A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		bc := c.BC()
		a := mb.GetItem(&bc)
		c.Registers.A = uint8(a)
		c.Registers.PC += 1
		return 8
	},

	// DEC BC - Decrement BC
	0x0B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		bc := c.BC()
		bc -= 1
		c.SetBC(bc)
		c.Registers.PC += 1
		return 8
	},

	// INC C - Increment C
	0x0C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Inc(c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// DEC C - Decrement C
	0x0D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Dec(c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// LD C, d8 - Load 8-bit immediate into C
	0x0E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// RRCA - Rotate A right. Old bit 0 to Carry flag
	0x0F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		a := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(a, 0) {
			c.SetFlagC()
			a = (a >> 1) + 0x80
		} else {
			c.ResetFlagC()
			a = (a >> 1)
		}

		c.Registers.A = a
		c.Registers.PC += 1
		return 4
	},

	// STOP 0 - Stop CPU & LCD display until button pressed
	0x10: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		// TODO: Implement
		// if c.mb.cgb == true {
		// 	var addr uint16 = 0xff04
		// 	c.mb.SetItem(&addr, 0)
		// }

		c.Registers.PC += 2
		return 4
	},
}
