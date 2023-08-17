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

	/****************************** 0xn0 **********************/
	// NOP - No operation (0)
	0x00: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		return 4
	},

	// STOP 0 - Stop CPU & LCD display until button pressed (16)
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

	// JR NZ, r8 - Relative jump if last result was not zero (32)
	0x20: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		if !c.IsFlagZSet() {
			c.Registers.PC += (2 + (uint16(value^0x80) - 0x80)) & 0xffff
			return 12
		}
		c.Registers.PC += 2
		return 8
	},

	// JR NC, r8 - Relative jump if last result caused no carry (48)
	0x30: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		if !c.IsFlagCSet() {
			c.Registers.PC += (2 + (uint16(value^0x80) - 0x80)) & 0xffff
			return 12
		}
		c.Registers.PC += 2
		return 8
	},

	// LD B, B - Copy B to B (64)
	0x40: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		// c.Registers.B = c.Registers.B
		c.Registers.PC += 1
		return 4
	},

	// LD D, B - Copy B to D (80)
	0x50: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Registers.B
		c.Registers.PC += 1
		return 4
	},

	// LD H, B - Copy B to H (96)
	0x60: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Registers.B
		c.Registers.PC += 1
		return 4
	},

	// LD (HL), B - Save B at address pointed to by HL (112)
	0x70: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := uint16(c.Registers.B)
		mb.SetItem(&hl, &b)
		c.Registers.PC += 1
		return 8
	},

	// ADD A, B - Add B to A (128)
	0x80: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// SUB B - Subtract B from A (144)
	0x90: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// AND B - Logical AND B against A (160)
	0xa0: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AndSetFlags(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// OR B - Logical OR B against A (176)
	0xb0: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.OrSetFlags(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// RET NZ - Return if last result was not zero (192)
	0xc0: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		var pch, pcl uint8
		if !c.IsFlagZSet() {
			spadd1 := c.Registers.SP + 1
			pch = mb.GetItem(&spadd1)
			pcl = mb.GetItem(&c.Registers.SP)

			c.Registers.PC = (uint16(pch) << 8) | uint16(pcl)

			c.Registers.SP += 2
			return 20
		} else {
			c.Registers.PC += 1
			return 8
		}
	},

	// RET NC - Return if last result did not cause carry (208)
	0xd0: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		var pch, pcl uint8
		if !c.IsFlagCSet() {
			spadd1 := c.Registers.SP + 1
			pch = mb.GetItem(&spadd1)
			pcl = mb.GetItem(&c.Registers.SP)

			c.Registers.PC = (uint16(pch) << 8) | uint16(pcl)

			c.Registers.SP += 2
			return 20
		} else {
			c.Registers.PC += 1
			return 8
		}
	},

	// LDH (a8), A - Save A at address $FF00 + 8-bit immediate (224)
	0xe0: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		var addr uint16 = 0xff00 + value
		a := uint16(c.Registers.A)
		mb.SetItem(&addr, &a)
		c.Registers.PC += 2
		return 12
	},

	// LDH A, (a8) - Load A with value at address $FF00 + 8-bit immediate (240)
	0xf0: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		var addr uint16 = 0xff00 + value
		a := mb.GetItem(&addr)
		c.Registers.A = a
		c.Registers.PC += 2
		return 12
	},

	/****************************** 0xn1 **********************/
	// LD BC, d16 - Load 16-bit immediate into BC (1)
	0x01: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.SetBC(value)
		c.Registers.PC += 3
		return 12
	},

	// LD DE, d16 - Load 16-bit immediate into DE (17)
	0x11: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.SetDE(value)
		c.Registers.PC += 3
		return 12
	},

	// LD HL, d16 - Load 16-bit immediate into HL (33)
	0x21: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.SetHL(value)
		c.Registers.PC += 3
		return 12
	},

	// LD SP, d16 - Load 16-bit immediate into SP (49)
	0x31: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.SP = value
		c.Registers.PC += 3
		return 12
	},

	// LD B, C - Copy C to B (65)
	0x41: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Registers.C
		c.Registers.PC += 1
		return 4
	},

	// LD D, C - Copy C to D (81)
	0x51: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Registers.C
		c.Registers.PC += 1
		return 4
	},

	// LD H, C - Copy C to H (97)
	0x61: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Registers.C
		c.Registers.PC += 1
		return 4
	},

	// LD (HL), C - Save C at address pointed to by HL (113)
	0x71: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		cr := uint16(c.Registers.C)
		mb.SetItem(&hl, &cr)
		c.Registers.PC += 1
		return 8
	},

	// ADD A, C - Add C to A (129)
	0x81: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// SUB C - Subtract C from A (145)
	0x91: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// AND C - Logical AND C against A (161)
	0xa1: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AndSetFlags(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// OR C - Logical OR C against A (177)
	0xb1: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.OrSetFlags(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// POP BC - Pop two bytes from stack into BC (193)
	0xc1: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		var pch, pcl uint8
		spadd1 := c.Registers.SP + 1
		pch = mb.GetItem(&spadd1)
		pcl = mb.GetItem(&c.Registers.SP)

		c.SetBC((uint16(pch) << 8) | uint16(pcl))

		c.Registers.SP += 2
		c.Registers.PC += 1
		return 12
	},

	// POP DE - Pop two bytes from stack into DE (209)
	0xd1: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		var pch, pcl uint8
		spadd1 := c.Registers.SP + 1
		pch = mb.GetItem(&spadd1)
		pcl = mb.GetItem(&c.Registers.SP)

		c.SetDE((uint16(pch) << 8) | uint16(pcl))

		c.Registers.SP += 2
		c.Registers.PC += 1
		return 12
	},

	// POP HL - Pop two bytes from stack into HL (225)
	0xe1: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		var pch, pcl uint8
		spadd1 := c.Registers.SP + 1
		pch = mb.GetItem(&spadd1)
		pcl = mb.GetItem(&c.Registers.SP)

		c.SetHL((uint16(pch) << 8) | uint16(pcl))

		c.Registers.SP += 2
		c.Registers.PC += 1
		return 12
	},

	// POP AF - Pop two bytes from stack into AF (241)
	0xf1: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		spadd1 := c.Registers.SP + 1
		c.Registers.A = mb.GetItem(&spadd1)
		c.Registers.F = mb.GetItem(&c.Registers.SP) & 0xF0 & 0xF0

		c.Registers.SP += 2
		c.Registers.PC += 1
		return 12
	},

	/****************************** 0xn2 **********************/
	// LD (BC), A - Save A to address pointed by BC (2)
	0x02: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		bc := c.BC()
		a := (uint16)(c.Registers.A)
		mb.SetItem(&bc, &a)
		c.Registers.PC += 1
		return 8
	},

	// LD (DE), A - Save A at address pointed to by DE (18)
	0x12: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		de := c.DE()
		a := uint16(c.Registers.A)
		mb.SetItem(&de, &a)
		c.Registers.PC += 1
		return 8
	},

	// LD (HL+), A - Save A at address pointed by HL, increment HL (34)
	0x22: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		a := uint16(c.Registers.A)
		mb.SetItem(&hl, &a)
		hl += 1
		c.SetHL(hl)
		c.Registers.PC += 1
		return 8
	},

	// LD (HL-), A - Save A at address pointed by HL, decrement HL (50)
	0x32: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		a := uint16(c.Registers.A)
		mb.SetItem(&hl, &a)
		hl -= 1
		c.SetHL(hl)
		c.Registers.PC += 1
		return 8
	},

	// LD B, D - Copy D to B (66)
	0x42: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Registers.D
		c.Registers.PC += 1
		return 4
	},

	// LD D, D - Copy D to D (82)
	0x52: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		// c.Registers.D = c.Registers.D
		c.Registers.PC += 1
		return 4
	},

	// LD H, D - Copy D to H (98)
	0x62: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Registers.D
		c.Registers.PC += 1
		return 4
	},

	// LD (HL), D - Save D at address pointed to by HL (114)
	0x72: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		d := uint16(c.Registers.D)
		mb.SetItem(&hl, &d)
		c.Registers.PC += 1
		return 8
	},

	// ADD A, D - Add D to A (130)
	0x82: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// SUB D - Subtract D from A (146)
	0x92: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// AND D - Logical AND D against A (162)
	0xa2: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AndSetFlags(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// OR D - Logical OR D against A (178)
	0xb2: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.OrSetFlags(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// JP NZ, a16 - Absolute jump to 16-bit location if last result was not zero (194)
	0xc2: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		if !c.IsFlagZSet() {
			c.Registers.PC = value
			return 16
		}
		c.Registers.PC += 3
		return 12
	},

	// JP NC, a16 - Absolute jump to 16-bit location if last result caused no carry (210)
	0xd2: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		if !c.IsFlagCSet() {
			c.Registers.PC = value
			return 16
		}
		c.Registers.PC += 3
		return 12
	},

	// LD (C), A - Save A at address $FF00 + register C (226)
	0xe2: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		var addr uint16 = 0xff00 + uint16(c.Registers.C)
		a := uint16(c.Registers.A)
		mb.SetItem(&addr, &a)
		c.Registers.PC += 1
		return 8
	},

	// LD A, (C) - Load A with value at address $FF00 + register C (242)
	0xf2: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		var addr uint16 = 0xff00 + uint16(c.Registers.C)
		a := mb.GetItem(&addr)
		c.Registers.A = a
		c.Registers.PC += 1
		return 8
	},

	/****************************** 0xn3 **********************/
	// // INC BC - Increment BC (3)
	0x03: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		bc := c.BC()
		bc += 1
		c.SetBC(bc)
		c.Registers.PC += 1
		return 8
	},

	// INC DE - Increment DE (19)
	0x13: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		de := c.DE()
		de += 1
		c.SetDE(de)
		c.Registers.PC += 1
		return 8
	},

	// INC HL - Increment HL (35)
	0x23: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		hl += 1
		c.SetHL(hl)
		c.Registers.PC += 1
		return 8
	},

	// INC SP - Increment SP (51)
	0x33: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.SP += 1
		c.Registers.PC += 1
		return 8
	},

	// LD B, E - Copy E to B (67)
	0x43: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Registers.E
		c.Registers.PC += 1
		return 4
	},

	// LD D, E - Copy E to D (83)
	0x53: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Registers.E
		c.Registers.PC += 1
		return 4
	},

	// LD H, E - Copy E to H (99)
	0x63: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Registers.E
		c.Registers.PC += 1
		return 4
	},

	// LD (HL), E - Save E at address pointed to by HL (115)
	0x73: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		e := uint16(c.Registers.E)
		mb.SetItem(&hl, &e)
		c.Registers.PC += 1
		return 8
	},

	// ADD A, E - Add E to A (131)
	0x83: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// SUB E - Subtract E from A (147)
	0x93: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// AND E - Logical AND E against A (163)
	0xa3: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AndSetFlags(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// OR E - Logical OR E against A (179)
	0xb3: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.OrSetFlags(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// JP a16 - Absolute jump to 16-bit location (195)
	0xc3: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC = value
		return 16
	},

	// 0xd3 - Illegal opcode
	// 0xe3 - Illegal opcode

	// DI - Disable interrupts (243)
	0xf3: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.IntrMasterEnable = false
		c.Registers.PC += 1
		return 4
	},

	/****************************** 0xn4 **********************/
	// // INC B - Increment B (4)
	0x04: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Inc(c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// INC D - Increment D (20)
	0x14: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Inc(c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// INC H - Increment H (36)
	0x24: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Inc(c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// INC (HL) - Increment value pointed by HL (52)
	0x34: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		v := mb.GetItem(&hl)
		v = c.Inc(v)

		v16 := uint16(v)
		mb.SetItem(&hl, &v16)
		c.Registers.PC += 1
		return 12
	},

	// LD B, H - Copy H to B (68)
	0x44: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Registers.H
		c.Registers.PC += 1
		return 4
	},

	// LD D, H - Copy H to D (84)
	0x54: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Registers.H
		c.Registers.PC += 1
		return 4
	},

	// LD H, H - Copy H to H (100)
	0x64: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		// c.Registers.H = c.Registers.H
		c.Registers.PC += 1
		return 4
	},

	// LD (HL), H - Save H at address pointed to by HL (116)
	0x74: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		h := uint16(c.Registers.H)
		mb.SetItem(&hl, &h)
		c.Registers.PC += 1
		return 8
	},

	// ADD A, H - Add H to A (132)
	0x84: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// SUB H - Subtract H from A (148)
	0x94: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// AND H - Logical AND H against A (164)
	0xa4: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AndSetFlags(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// OR H - Logical OR H against A (180)
	0xb4: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.OrSetFlags(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// CALL NZ, a16 - Call routine at 16-bit location if last result was not zero (196)
	0xc4: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 3
		if !c.IsFlagZSet() {
			sp1 := c.Registers.SP - 1
			sp2 := c.Registers.SP - 2

			pch := (c.Registers.PC >> 8) & 0xff
			pcl := c.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			c.Registers.SP -= 2
			c.Registers.PC = value
			return 24
		} else {
			return 12
		}
	},

	// CALL NC, a16 - Call routine at 16-bit location if last result caused no carry (212)
	0xd4: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 3
		if !c.IsFlagCSet() {
			sp1 := c.Registers.SP - 1
			sp2 := c.Registers.SP - 2

			pch := (c.Registers.PC >> 8) & 0xff
			pcl := c.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			c.Registers.SP -= 2
			c.Registers.PC = value
			return 24
		} else {
			return 12
		}
	},

	// 0xe4 - Illegal opcode
	// 0xf4 - Illegal opcode

	/****************************** 0xn5 **********************/
	// DEC B - Decrement B (5)
	0x05: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Dec(c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// DEC D - Decrement D (21)
	0x15: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Dec(c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// DEC H - Decrement H (37)
	0x25: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Dec(c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// DEC (HL) - Decrement value pointed by HL (53)
	0x35: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		v := mb.GetItem(&hl)
		v = c.Dec(v)

		v16 := uint16(v)
		mb.SetItem(&hl, &v16)
		c.Registers.PC += 1
		return 12
	},

	// LD B, L - Copy L to B (69)
	0x45: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Registers.L
		c.Registers.PC += 1
		return 4
	},

	// LD D, L - Copy L to D (85)
	0x55: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Registers.L
		c.Registers.PC += 1
		return 4
	},

	// LD H, L - Copy L to H (101)
	0x65: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Registers.L
		c.Registers.PC += 1
		return 4
	},

	// LD (HL), L - Save L at address pointed to by HL (117)
	0x75: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		l := uint16(c.Registers.L)
		mb.SetItem(&hl, &l)
		c.Registers.PC += 1
		return 8
	},

	// ADD A, L - Add L to A (133)
	0x85: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// SUB L - Subtract L from A (149)
	0x95: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// AND L - Logical AND L against A (165)
	0xa5: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AndSetFlags(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// OR L - Logical OR L against A (181)
	0xb5: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.OrSetFlags(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// PUSH BC - Push BC onto stack (197)
	0xc5: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		br := uint16(c.Registers.B)
		cr := uint16(c.Registers.C)
		mb.SetItem(&sp1, &br)
		mb.SetItem(&sp2, &cr)
		c.Registers.SP -= 2
		c.Registers.PC += 1
		return 16
	},

	// PUSH DE - Push DE onto stack (213)
	0xd5: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		dr := uint16(c.Registers.D)
		er := uint16(c.Registers.E)
		mb.SetItem(&sp1, &dr)
		mb.SetItem(&sp2, &er)
		c.Registers.SP -= 2
		c.Registers.PC += 1
		return 16
	},

	// PUSH HL - Push HL onto stack (229)
	0xe5: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		hr := uint16(c.Registers.H)
		lr := uint16(c.Registers.L)
		mb.SetItem(&sp1, &hr)
		mb.SetItem(&sp2, &lr)
		c.Registers.SP -= 2
		c.Registers.PC += 1
		return 16
	},

	// PUSH AF - Push AF onto stack (229)
	0xf5: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		ar := uint16(c.Registers.A)
		fr := uint16(c.Registers.F)
		mb.SetItem(&sp1, &ar)
		mb.SetItem(&sp2, &fr)
		c.Registers.SP -= 2
		c.Registers.PC += 1
		return 16
	},

	/****************************** 0xn6 **********************/
	// LD B, d8 - Load 8-bit immediate into B (6)
	0x06: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// LD D, d8 - Load 8-bit immediate into D (22)
	0x16: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// LD H, d8 - Load 8-bit immediate into H (38)
	0x26: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// LD (HL), d8 - Save 8-bit immediate to address pointed by HL (54)
	0x36: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		value &= 0xff
		mb.SetItem(&hl, &value)
		c.Registers.PC += 2
		return 12
	},

	// LD B, (HL) - Copy value pointed by HL to B (70)
	0x46: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.B = mb.GetItem(&hl)
		c.Registers.PC += 1
		return 8
	},

	// LD D, (HL) - Copy value pointed by HL to D (86)
	0x56: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.D = mb.GetItem(&hl)
		c.Registers.PC += 1
		return 8
	},

	// LD H, (HL) - Copy value pointed by HL to H (102)
	0x66: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.H = mb.GetItem(&hl)
		c.Registers.PC += 1
		return 8
	},

	// HALT - Power down CPU until an interrupt occurs (118)
	0x76: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Halted = true
		return 4
	},

	// ADD A, (HL) - Add value pointed by HL to A (134)
	0x86: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// SUB (HL) - Subtract value pointed by HL from A (150)
	0x96: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// AND (HL) - Logical AND value pointed by HL against A (166)
	0xa6: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = c.AndSetFlags(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// OR (HL) - Logical OR value pointed by HL against A (182)
	0xb6: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = c.OrSetFlags(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// ADD, d8 - Add 8-bit immediate to A (198)
	0xc6: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		v := uint8(value)
		c.Registers.A = c.AddSetFlags8(c.Registers.A, v)
		c.Registers.PC += 2
		return 8
	},

	// SUB d8 - Subtract 8-bit immediate from A (214)
	0xd6: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		v := uint8(value)
		c.Registers.A = c.SubSetFlags8(c.Registers.A, v)
		c.Registers.PC += 2
		return 8
	},

	// AND d8 - Logical AND 8-bit immediate against A (230)
	0xe6: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		v := uint8(value)
		c.Registers.A = c.AndSetFlags(c.Registers.A, v)
		c.Registers.PC += 2
		return 8
	},

	// OR d8 - Logical OR 8-bit immediate against A (246)
	0xf6: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		v := uint8(value)
		c.Registers.A = c.OrSetFlags(c.Registers.A, v)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn7 **********************/
	// RLCA - Rotate A left. Old bit 7 to Carry flag (7)
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

	// RLA - Rotate A left through Carry flag (23)
	0x17: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		a := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(a, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register A to the left by one bit
		c.Registers.A = (a << 1) & 0xff
		if oldCarry {
			c.Registers.A |= 0x01
		}

		c.Registers.PC += 1
		return 4
	},

	// DAA - Decimal adjust A (39)
	0x27: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		a := int16(c.Registers.A)

		var corr int16 = 0

		if c.IsFlagHSet() {
			corr |= 0x06
		}

		if c.IsFlagCSet() {
			corr |= 0x60
		}

		if c.IsFlagNSet() {
			a -= corr
		} else {
			if (a & 0x0f) > 0x09 {
				corr |= 0x06
			}

			if a > 0x99 {
				corr |= 0x60
			}
			a += corr
		}

		var flag uint8 = 0
		if (a & 0xff) == 0 {
			internal.SetBit(&flag, cpu.FLAGZ)
		}

		if corr&0x60 != 0 {
			internal.SetBit(&flag, cpu.FLAGC)
		}

		c.Registers.F &= 0b01000000
		c.Registers.F |= flag

		c.Registers.A = uint8(a)
		c.Registers.PC += 1
		return 4
	},

	// SCF - Set carry flag (55)
	0x37: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.SetFlagC()
		c.ResetFlagN()
		c.ResetFlagH()
		c.Registers.PC += 1
		return 4
	},

	// LD B, A - Copy A to B (71)
	0x47: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.B = c.Registers.A
		c.Registers.PC += 1
		return 4
	},

	// LD D, A - Copy A to D (87)
	0x57: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.D = c.Registers.A
		c.Registers.PC += 1
		return 4
	},

	// LD H, A - Copy A to H (103)
	0x67: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.H = c.Registers.A
		c.Registers.PC += 1
		return 4
	},

	// LD (HL), A - Save A at address pointed to by HL (119)
	0x77: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		a := uint16(c.Registers.A)
		mb.SetItem(&hl, &a)
		c.Registers.PC += 1
		return 8
	},

	// ADD A, A - Add A to A (135)
	0x87: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AddSetFlags8(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// SUB A - Subtract A from A (151)
	0x97: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SubSetFlags8(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// AND A - Logical AND A against A (167)
	0xa7: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AndSetFlags(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// OR A - Logical OR A against A (183)
	0xb7: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.OrSetFlags(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// RST 00H - Push present address onto stack. Jump to address $0000 (199)
	0xc7: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2
		c.Registers.PC = 0x00
		return 16
	},

	// RST 10H - Push present address onto stack. Jump to address $0010 (215)
	0xd7: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2
		c.Registers.PC = 0x10
		return 16
	},

	// RST 20 H - Push present address onto stack. Jump to address $0020 (231)
	0xe7: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2
		c.Registers.PC = 0x20
		return 16
	},

	// RST 30H - Push present address onto stack. Jump to address $0030 (247)
	0xf7: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2
		c.Registers.PC = 0x30
		return 16
	},

	/****************************** 0xn8 **********************/
	// LD (a16), SP - Save SP at given address (8)
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

	// JR r8 - Relative jump by signed immediate (24)
	0x18: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		v := int16(value^0x80) - 0x80              // convert to signed int
		c.Registers.PC += (2 + uint16(v)) & 0xffff // add to PC
		return 12
	},

	// JR Z, r8 - Relative jump by signed immediate if Z flag is set (40)
	0x28: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		v := int16(value^0x80) - 0x80 // convert to signed int

		if c.IsFlagZSet() {
			c.Registers.PC += (2 + uint16(v)) & 0xffff // add to PC
			return 12
		}

		c.Registers.PC += 2
		return 8
	},

	// JR C, r8 - Relative jump by signed immediate if C flag is set (56)
	0x38: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		v := int16(value^0x80) - 0x80 // convert to signed int

		if c.IsFlagCSet() {
			c.Registers.PC += (2 + uint16(v)) & 0xffff // add to PC
			return 12
		}

		c.Registers.PC += 2
		return 8
	},

	// LD C, B - Copy B to C (72)
	0x48: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Registers.B
		c.Registers.PC += 1
		return 4
	},

	// LD E, B - Copy B to E (88)
	0x58: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Registers.B
		c.Registers.PC += 1
		return 4
	},

	// LD L, B - Copy B to L (104)
	0x68: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Registers.B
		c.Registers.PC += 1
		return 4
	},

	// LD A, B - Copy B to A (120)
	0x78: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Registers.B
		c.Registers.PC += 1
		return 4
	},

	// ADC A, B - Add B and carry flag to A (136)
	0x88: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// SBC A, B - Subtract B and carry flag from A (152)
	0x98: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// XOR B - Logical XOR B against A (168)
	0xa8: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// CP B - Compare B against A (184)
	0xb8: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, c.Registers.B)
		c.Registers.PC += 1
		return 4
	},

	// RET Z - Return if last result was zero (200)
	0xc8: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		if c.IsFlagZSet() {
			nsp := c.Registers.SP + 1
			pcl := mb.GetItem(&c.Registers.SP)
			pch := mb.GetItem(&nsp)
			c.Registers.SP += 2
			c.Registers.PC = uint16(pch)<<8 | uint16(pcl)
			return 20
		} else {
			return 8
		}
	},

	// RET C - Return if last result caused carry (216)
	0xd8: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		if c.IsFlagCSet() {
			nsp := c.Registers.SP + 1
			pcl := mb.GetItem(&c.Registers.SP)
			pch := mb.GetItem(&nsp)
			c.Registers.SP += 2
			c.Registers.PC = uint16(pch)<<8 | uint16(pcl)
			return 20
		} else {
			return 8
		}
	},

	// ADD SP, r8 - Add signed 8-bit immediate to SP (232)
	0xe8: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		value &= 0xff
		var i8 int8 = int8((value ^ 0x80) - 0x80)
		sp := int32(c.Registers.SP)
		r := sp + int32(i8)
		i8_32 := int32(i8)

		c.ClearAllFlags()

		if ((sp&0xf)+(i8_32&0xf))&0x10 > 0xf {
			c.SetFlagH()
		}

		if (sp^i8_32^r)&0x100 == 0x100 {
			c.SetFlagC()
		}

		// var i8 int8 = int8((value ^ 0x80) - 0x80)
		// r := int32(c.Registers.SP) + int32(i8)
		// sp := int32(c.Registers.SP)

		// c.ClearAllFlags()
		// if (sp^int32(i8)^r)&0x100 == 0x100 {
		// 	c.SetFlagC()
		// }

		// // if (int32(c.Registers.SP)^int32(i8)^r)&0x10 != 0x0 {
		// // 	c.SetFlagH()
		// // }

		// if (sp&0xf)+(int32(i8)&0xf)&0x10 == 0x10 {
		// 	c.SetFlagH()
		// }
		c.Registers.SP = uint16(r)
		c.Registers.PC += 2
		return 16
	},

	// LD HL, SP+r8 - Add signed 8-bit immediate to SP (232)
	0xf8: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		value &= 0xff
		var i8 int8 = int8((value ^ 0x80) - 0x80)
		// var i8 int8 = int8(value)
		sp := int32(c.Registers.SP)
		i8_32 := int32(i8)
		r := sp + i8_32

		c.SetHL(uint16(r))

		c.ClearAllFlags()
		if ((sp&0xf)+(i8_32&0xf))&0x10 > 0xf {
			c.SetFlagH()
		}

		if (sp^i8_32^r)&0x100 == 0x100 {
			c.SetFlagC()
		}

		// if (int32(c.Registers.SP)^int32(i8)^r)&0x10 == 0x10 {
		// 	c.SetFlagC()
		// }

		// if (int32(c.Registers.SP)^int32(i8)^r)&0x100 == 0x100 {
		// 	c.SetFlagH()
		// }

		c.Registers.PC += 2
		return 12
	},

	/****************************** 0xn9 **********************/
	// ADD HL, BC - Add BC to HL (9)
	0x09: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		hl := c.AddSetFlags16(c.HL(), c.BC())

		c.SetHL(uint16(hl))
		c.Registers.PC += 1
		return 8
	},

	// ADD HL, DE - Add DE to HL (25)
	0x19: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()

		hl := c.AddSetFlags16(c.HL(), c.DE())
		c.SetHL(uint16(hl))
		c.Registers.PC += 1
		return 8
	},

	// ADD HL, HL - Add HL to HL (41)
	0x29: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.AddSetFlags16(c.HL(), c.HL())
		c.SetHL(uint16(hl))
		c.Registers.PC += 1
		return 8
	},

	// ADD HL, SP - Add SP to HL (57)
	0x39: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.AddSetFlags16(c.HL(), c.Registers.SP)
		c.SetHL(uint16(hl))
		c.Registers.PC += 1
		return 8
	},

	// LD C, C - Copy C to C (73)
	0x49: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		// c.Registers.C = c.Registers.C
		c.Registers.PC += 1
		return 4
	},

	// LD E, C - Copy C to E (89)
	0x59: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Registers.C
		c.Registers.PC += 1
		return 4
	},

	// LD L, C - Copy C to L (105)
	0x69: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Registers.C
		c.Registers.PC += 1
		return 4
	},

	// LD A, C - Copy C to A (121)
	0x79: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Registers.C
		c.Registers.PC += 1
		return 4
	},

	// ADC A, C - Add C and carry flag to A (137)
	0x89: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// SBC A, C - Subtract C and carry flag from A (153)
	0x99: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// XOR C - Logical XOR C against A (169)
	0xa9: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// CP C - Compare C against A (185)
	0xb9: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// RET - Pop two bytes from stack & jump to that address (201)
	0xc9: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		sp2 := c.Registers.SP + 1
		pcl := mb.GetItem(&c.Registers.SP)
		pch := mb.GetItem(&sp2)
		c.Registers.SP += 2
		c.Registers.PC = uint16(pch)<<8 | uint16(pcl)
		return 16
	},

	// RETI - Pop two bytes from stack & jump to that address then enable interrupts (217)
	0xd9: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.IntrMasterEnable = true
		sp2 := c.Registers.SP + 1
		pcl := mb.GetItem(&c.Registers.SP)
		pch := mb.GetItem(&sp2)
		c.Registers.SP += 2
		c.Registers.PC = uint16(pch)<<8 | uint16(pcl)
		return 16
	},

	// LD SP, HL - Copy HL to SP (233)
	0xf9: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.SP = c.HL()
		c.Registers.PC += 1
		return 8
	},

	/****************************** 0xna **********************/
	// LD A, (BC) - Load A from address pointed to by BC (10)
	0x0A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		bc := c.BC()
		a := mb.GetItem(&bc)
		c.Registers.A = uint8(a)
		c.Registers.PC += 1
		return 8
	},

	// LD A, (DE) - Load A with data from address pointed to by DE (26)
	0x1A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		de := c.DE()
		a := mb.GetItem(&de)
		c.Registers.A = uint8(a)
		c.Registers.PC += 1
		return 8
	},

	// LD A, (HL+) - Load A with data from address pointed to by HL, increment HL (42)
	0x2A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		a := mb.GetItem(&hl)
		c.Registers.A = uint8(a)
		hl += 1
		c.SetHL(hl)
		c.Registers.PC += 1
		return 8
	},

	// LD A, (HL-) - Load A with data from address pointed to by HL, decrement HL (58)
	0x3A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		a := mb.GetItem(&hl)
		c.Registers.A = uint8(a)
		hl -= 1
		c.SetHL(hl)
		c.Registers.PC += 1
		return 8
	},

	// LD C, D - Copy D to C (74)
	0x4A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Registers.D
		c.Registers.PC += 1
		return 4
	},

	// LD E, D - Copy D to E (90)
	0x5A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Registers.D
		c.Registers.PC += 1
		return 4
	},

	// LD L, D - Copy D to L (106)
	0x6A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Registers.D
		c.Registers.PC += 1
		return 4
	},

	// LD A, D - Copy D to A (122)
	0x7A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Registers.D
		c.Registers.PC += 1
		return 4
	},

	// ADC A, D - Add D and carry flag to A (138)
	0x8A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// SBC A, D - Subtract D and carry flag from A (154)
	0x9A: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// XOR D - Logical XOR D against A (170)
	0xaa: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// CP D - Compare D against A (186)
	0xba: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, c.Registers.D)
		c.Registers.PC += 1
		return 4
	},

	// JP Z, a16 - Absolute jump to 16-bit location if Z flag is set (202)
	0xca: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		if c.IsFlagZSet() {
			c.Registers.PC = value
			return 16
		}
		c.Registers.PC += 3
		return 12
	},

	// JP C, a16 - Absolute jump to 16-bit location if C flag is set (218)
	0xda: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		if c.IsFlagCSet() {
			c.Registers.PC = value
			return 16
		}
		c.Registers.PC += 3
		return 12
	},

	// LD (a16), A - Save A at given address (234)
	0xea: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		a := uint16(c.Registers.A)
		mb.SetItem(&value, &a)
		c.Registers.PC += 3
		return 16
	},

	// LD A, (a16) - Load A from given address (250)
	0xfa: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		a := mb.GetItem(&value)
		c.Registers.A = uint8(a)
		c.Registers.PC += 3
		return 16
	},

	/****************************** 0xnb **********************/
	// DEC BC - Decrement BC (11)
	0x0B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		bc := c.BC()
		bc -= 1
		c.SetBC(bc)
		c.Registers.PC += 1
		return 8
	},

	// DEC DE - Decrement DE (27)
	0x1B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		de := c.DE()
		de -= 1
		c.SetDE(de)
		c.Registers.PC += 1
		return 8
	},

	// DEC HL - Decrement HL (43)
	0x2B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		hl -= 1
		c.SetHL(hl)
		c.Registers.PC += 1
		return 8
	},

	// DEC SP - Decrement SP (59)
	0x3B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.SP -= 1
		c.Registers.PC += 1
		return 8
	},

	// LD C, E - Copy E to C (75)
	0x4B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Registers.E
		c.Registers.PC += 1
		return 4
	},

	// LD E, E - Copy E to E (91)
	0x5B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		// c.Registers.E = c.Registers.E
		c.Registers.PC += 1
		return 4
	},

	// LD L, E - Copy E to L (107)
	0x6B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Registers.E
		c.Registers.PC += 1
		return 4
	},

	// LD A, E - Copy E to A (123)
	0x7B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Registers.E
		c.Registers.PC += 1
		return 4
	},

	// ADC A, E - Add E and carry flag to A (139)
	0x8B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// SBC A, E - Subtract E and carry flag from A (155)
	0x9B: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// XOR E - Logical XOR E against A (171)
	0xab: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// CP E - Compare E against A (187)
	0xbb: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// PREFIX CB - CB prefix (203) --- isn't callable
	0xcb: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		return 4
	},

	// 0xdb - Illegal instruction
	// 0xeb - Illegal instruction
	// EI - Enable interrupts (235)
	0xfb: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.IntrMasterEnable = true
		c.Registers.PC += 1
		return 4
	},

	/****************************** 0xnc **********************/
	// INC C - Increment C (12)
	0x0C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Inc(c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// INC E - Increment E (28)
	0x1C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Inc(c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// INC L - Increment L (44)
	0x2C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Inc(c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// INC A - Increment A (60)
	0x3C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Inc(c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// LD C, H - Copy H to C (76)
	0x4C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Registers.H
		c.Registers.PC += 1
		return 4
	},

	// LD E, H - Copy H to E (92)
	0x5C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Registers.H
		c.Registers.PC += 1
		return 4
	},

	// LD L, H - Copy H to L (108)
	0x6C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Registers.H
		c.Registers.PC += 1
		return 4
	},

	// LD A, H - Copy H to A (124)
	0x7C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Registers.H
		c.Registers.PC += 1
		return 4
	},

	// ADC A, H - Add H and carry flag to A (140)
	0x8C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// SBC A, H - Subtract H and carry flag from A (156)
	0x9C: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// XOR H - Logical XOR H against A (172)
	0xac: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// CP H - Compare H against A (188)
	0xbc: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, c.Registers.H)
		c.Registers.PC += 1
		return 4
	},

	// CALL Z, a16 - Call routine at 16-bit address if Z flag is set (204)
	0xcc: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		if c.IsFlagZSet() {
			sp1 := c.Registers.SP - 1
			sp2 := c.Registers.SP - 2

			pch := (c.Registers.PC >> 8) & 0xff
			pcl := c.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			c.Registers.SP -= 2

			c.Registers.PC = value
			return 24
		}
		c.Registers.PC += 3
		return 12
	},

	// CALL C, a16 - Call routine at 16-bit address if C flag is set (220)
	0xdc: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 3

		if c.IsFlagCSet() {
			sp1 := c.Registers.SP - 1
			sp2 := c.Registers.SP - 2

			pch := (c.Registers.PC >> 8) & 0xff
			pcl := c.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			c.Registers.SP -= 2

			c.Registers.PC = value
			return 24
		}
		return 12
	},

	// 0xec - Illegal instruction
	// 0xfc - Illegal instruction

	/****************************** 0xnd **********************/
	// DEC C - Decrement C (13)
	0x0D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Dec(c.Registers.C)
		c.Registers.PC += 1
		return 4
	},

	// DEC E - Decrement E (29)
	0x1D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Dec(c.Registers.E)
		c.Registers.PC += 1
		return 4
	},

	// DEC L - Decrement L (45)
	0x2D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Dec(c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// DEC A - Decrement A (61)
	0x3D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Dec(c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// LD C, L - Copy L to C (77)
	0x4D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Registers.L
		c.Registers.PC += 1
		return 4
	},

	// LD E, L - Copy L to E (93)
	0x5D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Registers.L
		c.Registers.PC += 1
		return 4
	},

	// LD L, L - Copy L to L (109)
	0x6D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		// c.Registers.L = c.Registers.L
		c.Registers.PC += 1
		return 4
	},

	// LD A, L - Copy L to A (125)
	0x7D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.Registers.L
		c.Registers.PC += 1
		return 4
	},

	// ADC A, L - Add L and carry flag to A (141)
	0x8D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// SBC A, L - Subtract L and carry flag from A (157)
	0x9D: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// XOR L - Logical XOR L against A (173)
	0xad: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// CP L - Compare L against A (189)
	0xbd: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, c.Registers.L)
		c.Registers.PC += 1
		return 4
	},

	// CALL a16 - Call routine at 16-bit address (205)
	0xcd: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 3
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2

		c.Registers.PC = value
		return 24
	},

	// 0xdd - Illegal instruction

	/****************************** 0xne **********************/
	// LD C, d8 - Load 8-bit immediate into C (14)
	0x0E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// LD E, d8 - Load 8-bit immediate into E (30)
	0x1E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// LD L, d8 - Load 8-bit immediate into L (46)
	0x2E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// LD A, d8 - Load 8-bit immediate into A (62)
	0x3E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = uint8(value)
		c.Registers.PC += 2
		return 8
	},

	// LD C, (HL) - Copy value pointed by HL to C (78)
	0x4E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.C = mb.GetItem(&hl)
		c.Registers.PC += 1
		return 8
	},

	// LD E, (HL) - Copy value pointed by HL to E (94)
	0x5E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.E = mb.GetItem(&hl)
		c.Registers.PC += 1
		return 8
	},

	// LD L, (HL) - Copy value pointed by HL to L (110)
	0x6E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.L = mb.GetItem(&hl)
		c.Registers.PC += 1
		return 8
	},

	// LD A, (HL) - Copy value pointed by HL to A (126)
	0x7E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = mb.GetItem(&hl)
		c.Registers.PC += 1
		return 8
	},

	// ADC A, (HL) - Add value pointed by HL and carry flag to A (142)
	0x8E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// SBC A, (HL) - Subtract value pointed by HL and carry flag from A (158)
	0x9E: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// XOR (HL) - Logical XOR value pointed by HL against A (174)
	0xae: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.Registers.A = c.XorSetFlags(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// CP (HL) - Compare value pointed by HL against A (190)
	0xbe: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		c.CpSetFlags(c.Registers.A, mb.GetItem(&hl))
		c.Registers.PC += 1
		return 8
	},

	// ADC A, d8 - Add 8-bit immediate and carry flag to A (206)
	0xce: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, uint8(value))
		c.Registers.PC += 2
		return 8
	},

	// SBC A, d8 - Subtract 8-bit immediate and carry flag from A (222)
	0xde: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, uint8(value))
		c.Registers.PC += 2
		return 8
	},

	// XOR d8 - Logical XOR n against A (236)
	0xee: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, uint8(value))
		c.Registers.PC += 2
		return 8
	},

	// CP d8 - Compare n against A (252)
	0xfe: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, uint8(value))
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xnf **********************/
	// RRCA - Rotate A right. Old bit 0 to Carry flag (15)
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

	// RRA - Rotate A right through Carry flag (31)
	0x1F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		a := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(a, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register A to the right by one bit
		c.Registers.A = (a >> 1) & 0xff
		if oldCarry {
			c.Registers.A |= 0x80
		}

		c.Registers.PC += 1
		return 4
	},

	// CPL - Complement A register (47)
	0x2F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = ^c.Registers.A
		c.SetFlagN()
		c.SetFlagH()
		c.Registers.PC += 1
		return 4
	},

	// CCF - Complement carry flag (63)
	0x3F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.ResetFlagH()
		if c.IsFlagCSet() {
			c.ResetFlagC()
		} else {
			c.SetFlagC()
		}
		c.Registers.PC += 1
		return 4
	},

	// LD C, A - Copy A to C (79)
	0x4F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.C = c.Registers.A
		c.Registers.PC += 1
		return 4
	},

	// LD E, A - Copy A to E (95)
	0x5F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.E = c.Registers.A
		c.Registers.PC += 1
		return 4
	},

	// LD L, A - Copy A to L (111)
	0x6F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.L = c.Registers.A
		c.Registers.PC += 1
		return 4
	},

	// LD A, A - Copy A to A (127)
	0x7F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		// c.Registers.A = c.Registers.A
		c.Registers.PC += 1
		return 4
	},

	// ADC A, A - Add A and carry flag to A (143)
	0x8F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.AdcSetFlags8(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// SBC A, A - Subtract A and carry flag from A (159)
	0x9F: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.SbcSetFlags8(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// XOR A - Logical XOR A against A (175)
	0xaf: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.A = c.XorSetFlags(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// CP A - Compare A against A (191)
	0xbf: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.CpSetFlags(c.Registers.A, c.Registers.A)
		c.Registers.PC += 1
		return 4
	},

	// RST 08H - Push present address onto stack. Jump to address $0008 (207)
	0xcf: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2

		c.Registers.PC = 0x08
		return 16
	},

	// RST 18H - Push present address onto stack. Jump to address $0018 (223)
	0xdf: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2

		c.Registers.PC = 0x18
		return 16
	},

	// RST 28H - Push present address onto stack. Jump to address $0028 (239)
	0xef: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2

		c.Registers.PC = 0x28
		return 16
	},

	// RST 38H - Push present address onto stack. Jump to address $0038 (255)
	0xff: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.Registers.PC += 1
		sp1 := c.Registers.SP - 1
		sp2 := c.Registers.SP - 2

		pch := (c.Registers.PC >> 8) & 0xff
		pcl := c.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2

		c.Registers.PC = 0x38
		return 16
	},

	/************************************************************* CB PREFIX *************************************************************/
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	//
	/************************************************************* CB PREFIX *************************************************************/

	/****************************** 0xn0 **********************/
	// RLC B - Rotate B left. Old bit 7 to Carry flag (256) [minus 0xFF for CB prefix]
	0x100: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// RL B - Rotate B left through Carry flag (272) [minus 0xFF for CB prefix]
	0x110: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register B to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// SLA B - Shift B left into Carry. LSB of B set to 0 (288) [minus 0xFF for CB prefix]
	0x120: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// SWAP B - Swap upper & lower nibles of B (304) [minus 0xFF for CB prefix]
	0x130: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 0, B - Test bit 0 of B (320) [minus 0xFF for CB prefix]
	0x140: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 2, B - Test bit 2 of B (336) [minus 0xFF for CB prefix]
	0x150: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 4, B - Test bit 4 of B (352) [minus 0xFF for CB prefix]
	0x160: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 6, B - Test bit 6 of B (368) [minus 0xFF for CB prefix]
	0x170: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 0, B - Reset bit 0 of B (384) [minus 0xFF for CB prefix]
	0x180: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.B, 0)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn1 **********************/
	// RLC C - Rotate C left. Old bit 7 to Carry flag (257) [minus 0xFF for CB prefix]
	0x101: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// RL C - Rotate C left through Carry flag (273) [minus 0xFF for CB prefix]
	0x111: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register C to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// SLA C - Shift C left into Carry. LSB of C set to 0 (289) [minus 0xFF for CB prefix]
	0x121: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// SWAP C - Swap upper & lower nibles of C (305) [minus 0xFF for CB prefix]
	0x131: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 0, C - Test bit 0 of C (321) [minus 0xFF for CB prefix]
	0x141: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 2, C - Test bit 2 of C (337) [minus 0xFF for CB prefix]
	0x151: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 4, C - Test bit 4 of C (353) [minus 0xFF for CB prefix]
	0x161: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 6, C - Test bit 6 of C (369) [minus 0xFF for CB prefix]
	0x171: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 0, C - Reset bit 0 of C (385) [minus 0xFF for CB prefix]
	0x181: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.C, 0)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn2 **********************/
	// RLC D - Rotate D left. Old bit 7 to Carry flag (258) [minus 0xFF for CB prefix]
	0x102: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// RL D - Rotate D left through Carry flag (274) [minus 0xFF for CB prefix]
	0x112: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register D to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// SLA D - Shift D left into Carry. LSB of D set to 0 (290) [minus 0xFF for CB prefix]
	0x122: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// SWAP D - Swap upper & lower nibles of D (306) [minus 0xFF for CB prefix]
	0x132: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 0, D - Test bit 0 of D (322) [minus 0xFF for CB prefix]
	0x142: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 2, D - Test bit 2 of D (338) [minus 0xFF for CB prefix]
	0x152: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 4, D - Test bit 4 of D (354) [minus 0xFF for CB prefix]
	0x162: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 6, D - Test bit 6 of D (370) [minus 0xFF for CB prefix]
	0x172: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 0, D - Reset bit 0 of D (386) [minus 0xFF for CB prefix]
	0x182: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.D, 0)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn3 **********************/
	// RLC E - Rotate E left. Old bit 7 to Carry flag (259) [minus 0xFF for CB prefix]
	0x103: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// RL E - Rotate E left through Carry flag (275) [minus 0xFF for CB prefix]
	0x113: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register E to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// SLA E - Shift E left into Carry. LSB of E set to 0 (291) [minus 0xFF for CB prefix]
	0x123: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// SWAP E - Swap upper & lower nibles of E (307) [minus 0xFF for CB prefix]
	0x133: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 0, E - Test bit 0 of E (323) [minus 0xFF for CB prefix]
	0x143: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.E, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 2, E - Test bit 2 of E (339) [minus 0xFF for CB prefix]
	0x153: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.E, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 4, E - Test bit 4 of E (355) [minus 0xFF for CB prefix]
	0x163: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.E, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 6, E - Test bit 6 of E (371) [minus 0xFF for CB prefix]
	0x173: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.E, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 0, E - Reset bit 0 of E (387) [minus 0xFF for CB prefix]
	0x183: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.E, 0)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn4 **********************/
	// RLC H - Rotate H left. Old bit 7 to Carry flag (260) [minus 0xFF for CB prefix]
	0x104: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// RL H - Rotate H left through Carry flag (276) [minus 0xFF for CB prefix]
	0x114: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register H to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// SLA H - Shift H left into Carry. LSB of H set to 0 (292) [minus 0xFF for CB prefix]
	0x124: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// SWAP H - Swap upper & lower nibles of H (308) [minus 0xFF for CB prefix]
	0x134: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 0, H - Test bit 0 of H (324) [minus 0xFF for CB prefix]
	0x144: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.H, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 2, H - Test bit 2 of H (340) [minus 0xFF for CB prefix]
	0x154: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.H, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 4, H - Test bit 4 of H (356) [minus 0xFF for CB prefix]
	0x164: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.H, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 6, H - Test bit 6 of H (372) [minus 0xFF for CB prefix]
	0x174: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.H, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 0, H - Reset bit 0 of H (388) [minus 0xFF for CB prefix]
	0x184: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.H, 0)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn5 **********************/
	// RLC L - Rotate L left. Old bit 7 to Carry flag (261) [minus 0xFF for CB prefix]
	0x105: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// RL L - Rotate L left through Carry flag (277) [minus 0xFF for CB prefix]
	0x115: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register L to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// SLA L - Shift L left into Carry. LSB of L set to 0 (293) [minus 0xFF for CB prefix]
	0x125: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// SWAP L - Swap upper & lower nibles of L (309) [minus 0xFF for CB prefix]
	0x135: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 0, L - Test bit 0 of L (325) [minus 0xFF for CB prefix]
	0x145: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.L, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 2, L - Test bit 2 of L (341) [minus 0xFF for CB prefix]
	0x155: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.L, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 4, L - Test bit 4 of L (357) [minus 0xFF for CB prefix]
	0x165: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.L, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 6, L - Test bit 6 of L (373) [minus 0xFF for CB prefix]
	0x175: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.L, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 0, L - Reset bit 0 of L (389) [minus 0xFF for CB prefix]
	0x185: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.L, 0)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn6 **********************/
	// RLC (HL) - Rotate value pointed by HL left. Old bit 7 to Carry flag (262) [minus 0xFF for CB prefix]
	0x106: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// RL (HL) - Rotate value pointed by HL left through Carry flag (278) [minus 0xFF for CB prefix]
	0x116: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift value pointed by HL to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// SLA (HL) - Shift value pointed by HL left into Carry. LSB of value pointed by HL set to 0 (294) [minus 0xFF for CB prefix]
	0x126: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// SWAP (HL) - Swap upper & lower nibles of value pointed by HL (310) [minus 0xFF for CB prefix]
	0x136: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// BIT 0, (HL) - Test bit 0 of value pointed by HL (326) [minus 0xFF for CB prefix]
	0x146: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(b, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// BIT 2, (HL) - Test bit 2 of value pointed by HL (342) [minus 0xFF for CB prefix]
	0x156: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(b, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// BIT 4, (HL) - Test bit 4 of value pointed by HL (358) [minus 0xFF for CB prefix]
	0x166: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(b, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// BIT 6, (HL) - Test bit 6 of value pointed by HL (374) [minus 0xFF for CB prefix]
	0x176: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(b, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// RES 0, (HL) - Reset bit 0 of value pointed by HL (390) [minus 0xFF for CB prefix]
	0x186: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 0)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	/****************************** 0xn7 **********************/
	// RLC A - Rotate A left. Old bit 7 to Carry flag (263) [minus 0xFF for CB prefix]
	0x107: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			c.ResetFlagC()
			b = (b << 1)
		}

		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// RL A - Rotate A left through Carry flag (279) [minus 0xFF for CB prefix]
	0x117: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register A to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// SLA A - Shift A left into Carry. LSB of A set to 0 (295) [minus 0xFF for CB prefix]
	0x127: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		b = (b << 1) & 0xff
		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// SWAP A - Swap upper & lower nibles of A (311) [minus 0xFF for CB prefix]
	0x137: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 0, A - Test bit 0 of A (327) [minus 0xFF for CB prefix]
	0x147: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.A, 0) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 2, A - Test bit 2 of A (343) [minus 0xFF for CB prefix]
	0x157: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.A, 2) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 4, A - Test bit 4 of A (359) [minus 0xFF for CB prefix]
	0x167: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.A, 4) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 6, A - Test bit 6 of A (375) [minus 0xFF for CB prefix]
	0x177: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.A, 6) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 0, A - Reset bit 0 of A (391) [minus 0xFF for CB prefix]
	0x187: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.A, 0)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn8 **********************/
	// RRC B - Rotate B right. Old bit 0 to Carry flag (264) [minus 0xFF for CB prefix]
	0x108: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// RR B - Rotate B right through Carry flag (280) [minus 0xFF for CB prefix]
	0x118: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register B to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// SRA B - Shift B right into Carry. MSB doesn't change (296) [minus 0xFF for CB prefix]
	0x128: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register B to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.B, 7) {
			b |= 0x80
		}

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// SRL B - Shift B right into Carry. MSB set to 0 (312) [minus 0xFF for CB prefix]
	0x138: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.B
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register B to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.B = b
		c.Registers.PC += 2
		return 8
	},

	// BIT B 1 - Test bit 1 of B (328) [minus 0xFF for CB prefix]
	0x148: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT B 3 - Test bit 3 of B (344) [minus 0xFF for CB prefix]
	0x158: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT B 5 - Test bit 5 of B (360) [minus 0xFF for CB prefix]
	0x168: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT B 7 - Test bit 7 of B (376) [minus 0xFF for CB prefix]
	0x178: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.B, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES B 1 - Reset bit 0 of B (392) [minus 0xFF for CB prefix]
	0x188: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.B, 1)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xn9 **********************/
	// RRC C - Rotate C right. Old bit 0 to Carry flag (265) [minus 0xFF for CB prefix]
	0x109: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// RR C - Rotate C right through Carry flag (281) [minus 0xFF for CB prefix]
	0x119: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register C to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// SRA C - Shift C right into Carry. MSB doesn't change (297) [minus 0xFF for CB prefix]
	0x129: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register C to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.C, 7) {
			b |= 0x80
		}

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// SRL C - Shift C right into Carry. MSB set to 0 (313) [minus 0xFF for CB prefix]
	0x139: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.C
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register C to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.C = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 1, C - Test bit 1 of C (329) [minus 0xFF for CB prefix]
	0x149: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 3, C - Test bit 3 of C (345) [minus 0xFF for CB prefix]
	0x159: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 5, C - Test bit 5 of C (361) [minus 0xFF for CB prefix]
	0x169: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 7, C - Test bit 7 of C (377) [minus 0xFF for CB prefix]
	0x179: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.C, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 1, C - Reset bit 1 of C (393) [minus 0xFF for CB prefix]
	0x189: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.C, 1)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xna **********************/
	// RRC D - Rotate D right. Old bit 0 to Carry flag (266) [minus 0xFF for CB prefix]
	0x10a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// RR D - Rotate D right through Carry flag (282) [minus 0xFF for CB prefix]
	0x11a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register D to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// SRA D - Shift D right into Carry. MSB doesn't change (298) [minus 0xFF for CB prefix]
	0x12a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register D to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.D, 7) {
			b |= 0x80
		}

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// SRL D - Shift D right into Carry. MSB set to 0 (314) [minus 0xFF for CB prefix]
	0x13a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.D
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register D to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.D = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 1, D - Test bit 1 of D (330) [minus 0xFF for CB prefix]
	0x14a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 3, D - Test bit 3 of D (346) [minus 0xFF for CB prefix]
	0x15a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 5, D - Test bit 5 of D (362) [minus 0xFF for CB prefix]
	0x16a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 7, D - Test bit 7 of D (378) [minus 0xFF for CB prefix]
	0x17a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.D, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 1, D - Reset bit 2 of D (394) [minus 0xFF for CB prefix]
	0x18a: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.D, 1)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xnb **********************/
	// RRC E - Rotate E right. Old bit 0 to Carry flag (267) [minus 0xFF for CB prefix]
	0x10b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// RR E - Rotate E right through Carry flag (283) [minus 0xFF for CB prefix]
	0x11b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register E to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// SRA E - Shift E right into Carry. MSB doesn't change (299) [minus 0xFF for CB prefix]
	0x12b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register E to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.E, 7) {
			b |= 0x80
		}

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// SRL E - Shift E right into Carry. MSB set to 0 (315) [minus 0xFF for CB prefix]
	0x13b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.E
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}
		// shift register E to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.E = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 1, E - Test bit 1 of E (331) [minus 0xFF for CB prefix]
	0x14b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.E, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 3, E - Test bit 3 of E (347) [minus 0xFF for CB prefix]
	0x15b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.E, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 5, E - Test bit 5 of E (363) [minus 0xFF for CB prefix]
	0x16b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.E, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 7, E - Test bit 7 of E (379) [minus 0xFF for CB prefix]
	0x17b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.E, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 1, E - Reset bit 3 of E (395) [minus 0xFF for CB prefix]
	0x18b: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.E, 1)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xnc **********************/
	// RRC H - Rotate H right. Old bit 0 to Carry flag (268) [minus 0xFF for CB prefix]
	0x10c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// RR H - Rotate H right through Carry flag (284) [minus 0xFF for CB prefix]
	0x11c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register H to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// SRA H - Shift H right into Carry. MSB doesn't change (300) [minus 0xFF for CB prefix]
	0x12c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register H to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.H, 7) {
			b |= 0x80
		}

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// SRL H - Shift H right into Carry. MSB set to 0 (316) [minus 0xFF for CB prefix]
	0x13c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.H
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}
		// shift register H to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.H = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 1, H - Test bit 1 of H (332) [minus 0xFF for CB prefix]
	0x14c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.H, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 3, H - Test bit 3 of H (348) [minus 0xFF for CB prefix]
	0x15c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.H, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 5, H - Test bit 5 of H (364) [minus 0xFF for CB prefix]
	0x16c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.H, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 7, H - Test bit 7 of H (380) [minus 0xFF for CB prefix]
	0x17c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.H, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// 14, H - Reset bit 4 of H (396) [minus 0xFF for CB prefix]
	0x18c: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.H, 1)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xnd **********************/
	// RRC L - Rotate L right. Old bit 0 to Carry flag (269) [minus 0xFF for CB prefix]
	0x10d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// RR L - Rotate L right through Carry flag (285) [minus 0xFF for CB prefix]
	0x11d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register L to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// SRA L - Shift L right into Carry. MSB doesn't change (301) [minus 0xFF for CB prefix]
	0x12d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register L to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.L, 7) {
			b |= 0x80
		}

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// SRL L - Shift L right into Carry. MSB set to 0 (317) [minus 0xFF for CB prefix]
	0x13d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.L
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}
		// shift register L to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.L = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 1, L - Test bit 1 of L (333) [minus 0xFF for CB prefix]
	0x14d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.L, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 3, L - Test bit 3 of L (349) [minus 0xFF for CB prefix]
	0x15d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.L, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 5, L - Test bit 5 of L (365) [minus 0xFF for CB prefix]
	0x16d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.L, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 7, L - Test bit 7 of L (381) [minus 0xFF for CB prefix]
	0x17d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.L, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 1, L - Reset bit 5 of L (397) [minus 0xFF for CB prefix]
	0x18d: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.L, 1)
		c.Registers.PC += 2
		return 8
	},

	/****************************** 0xne **********************/
	// RRC (HL) - Rotate value pointed by HL right. Old bit 0 to Carry flag (270) [minus 0xFF for CB prefix]
	0x10e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// RR (HL) - Rotate value pointed by HL right through Carry flag (286) [minus 0xFF for CB prefix]
	0x11e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift value pointed by HL to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// SRA (HL) - Shift value pointed by HL right into Carry. MSB doesn't change (302) [minus 0xFF for CB prefix]
	0x12e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift value pointed by HL to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.L, 7) {
			b |= 0x80
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// SRL (HL) - Shift value pointed by HL right into Carry. MSB set to 0 (318) [minus 0xFF for CB prefix]
	0x13e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}
		// shift value pointed by HL to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	// BIT 1, (HL) - Test bit 1 of value pointed by HL (334) [minus 0xFF for CB prefix]
	0x14e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(b, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// BIT 3, (HL) - Test bit 3 of value pointed by HL (350) [minus 0xFF for CB prefix]
	0x15e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(b, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// BIT 5, (HL) - Test bit 5 of value pointed by HL (366) [minus 0xFF for CB prefix]
	0x16e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(b, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// BIT 7, (HL) - Test bit 7 of value pointed by HL (382) [minus 0xFF for CB prefix]
	0x17e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(b, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 16
	},

	// RES 1, (HL) - Reset bit 6 of value pointed by HL (398) [minus 0xFF for CB prefix]
	0x18e: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		hl := c.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 1)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		c.Registers.PC += 2
		return 16
	},

	/****************************** 0xnf **********************/
	// RRC A - Rotate A right. Old bit 0 to Carry flag (271) [minus 0xFF for CB prefix]
	0x10f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			c.ResetFlagC()
			b = (b >> 1)
		}

		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// RR A - Rotate A right through Carry flag (287) [minus 0xFF for CB prefix]
	0x11f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		oldCarry := c.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register A to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// SRA A - Shift A right into Carry. MSB doesn't change (303) [minus 0xFF for CB prefix]
	0x12f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()

		if internal.IsBitSet(c.Registers.A, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}

		// shift register A to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(c.Registers.A, 7) {
			b |= 0x80
		}

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// SRL A - Shift A right into Carry. MSB set to 0 (319) [minus 0xFF for CB prefix]
	0x13f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		b := c.Registers.A
		c.ResetFlagZ()
		c.ResetFlagN()
		c.ResetFlagH()
		c.ResetFlagC()

		if internal.IsBitSet(c.Registers.A, 0) {
			c.SetFlagC()
		} else {
			c.ResetFlagC()
		}
		// shift register A to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			c.SetFlagZ()
		}

		c.Registers.A = b
		c.Registers.PC += 2
		return 8
	},

	// BIT 1, A - Test bit 1 of A (335) [minus 0xFF for CB prefix]
	0x14f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()

		c.SetFlagZ()
		if internal.IsBitSet(c.Registers.A, 1) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 3, A - Test bit 3 of A (351) [minus 0xFF for CB prefix]
	0x15f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.A, 3) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 5, A - Test bit 5 of A (367) [minus 0xFF for CB prefix]
	0x16f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.A, 5) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// BIT 7, A - Test bit 7 of A (383) [minus 0xFF for CB prefix]
	0x17f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		c.ResetFlagN()
		c.SetFlagH()
		c.SetFlagZ()

		if internal.IsBitSet(c.Registers.A, 7) {
			c.ResetFlagZ()
		}
		c.Registers.PC += 2
		return 8
	},

	// RES 1, A - Reset bit 7 of A (399) [minus 0xFF for CB prefix]
	0x18f: func(mb Motherboard, value uint16) uint8 {
		c := mb.Cpu()
		internal.ResetBit(&c.Registers.A, 1)
		c.Registers.PC += 2
		return 8
	},
}
