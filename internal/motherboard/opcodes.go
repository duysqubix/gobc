package motherboard

import (
	"github.com/duysqubix/gobc/internal"
)

func (o *OpCode) CBPrefix() bool {
	return *o == 0xcb
}

func (o *OpCode) Shift() OpCode {
	return *o + CB_SHIFT
}

var ILLEGAL_OPCODES = []OpCode{0xd3, 0xdb, 0xdd, 0xe3, 0xe4, 0xeb, 0xec, 0xed, 0xf4, 0xfc, 0xfd}

// OPCODES is a map of opcodes to their logic
var OPCODES = OpCodeMap{

	/****************************** 0xn0 **********************/
	// NOP - No operation (0)
	0x00: func(mb *Motherboard, value uint16) OpCycles {
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// STOP 0 - Stop CPU & LCD display until button pressed (16)
	0x10: func(mb *Motherboard, value uint16) OpCycles {
		mb.Cpu.Halted = true
		if mb.Cgb {
			var addr uint16 = 0xff04
			var value uint16 = 0x00
			mb.SetItem(&addr, &value)
		}

		// reset timer
		mb.Cpu.Mb.Timer.DIV = 0x00
		mb.Cpu.Registers.PC += 2
		return 4
	},

	// JR NZ, r8 - Relative jump if last result was not zero (32)
	0x20: func(mb *Motherboard, value uint16) OpCycles {

		if !mb.Cpu.IsFlagZSet() {
			mb.Cpu.Registers.PC += (2 + (uint16(value^0x80) - 0x80)) & 0xffff
			return 12
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// JR NC, r8 - Relative jump if last result caused no carry (48)
	0x30: func(mb *Motherboard, value uint16) OpCycles {

		if !mb.Cpu.IsFlagCSet() {
			mb.Cpu.Registers.PC += (2 + (uint16(value^0x80) - 0x80)) & 0xffff
			return 12
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD B, B - Copy B to B (64)
	0x40: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Registers.B = mb.Cpu.Registers.B
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD D, B - Copy B to D (80)
	0x50: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Registers.B
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD H, B - Copy B to H (96)
	0x60: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Registers.B
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD (HL), B - Save B at address pointed to by HL (112)
	0x70: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := uint16(mb.Cpu.Registers.B)
		mb.SetItem(&hl, &b)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD A, B - Add B to A (128)
	0x80: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SUB B - Subtract B from A (144)
	0x90: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// AND B - Logical AND B against A (160)
	0xa0: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// OR B - Logical OR B against A (176)
	0xb0: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// RET NZ - Return if last result was not zero (192)
	0xc0: func(mb *Motherboard, value uint16) OpCycles {

		var pch, pcl uint8
		if !mb.Cpu.IsFlagZSet() {
			spadd1 := mb.Cpu.Registers.SP + 1
			pch = mb.GetItem(&spadd1)
			pcl = mb.GetItem(&mb.Cpu.Registers.SP)
			mb.Cpu.Registers.PC = (uint16(pch) << 8) | uint16(pcl)

			mb.Cpu.Registers.SP += 2
			return 20
		} else {
			mb.Cpu.Registers.PC += 1
			return 8
		}
	},

	// RET NC - Return if last result did not cause carry (208)
	0xd0: func(mb *Motherboard, value uint16) OpCycles {

		var pch, pcl uint8
		if !mb.Cpu.IsFlagCSet() {
			spadd1 := mb.Cpu.Registers.SP + 1
			pch = mb.GetItem(&spadd1)
			pcl = mb.GetItem(&mb.Cpu.Registers.SP)

			mb.Cpu.Registers.PC = (uint16(pch) << 8) | uint16(pcl)

			mb.Cpu.Registers.SP += 2
			return 20
		} else {
			mb.Cpu.Registers.PC += 1
			return 8
		}
	},

	// LDH (a8), A - Save A at address $FF00 + 8-bit immediate (224)
	0xe0: func(mb *Motherboard, value uint16) OpCycles {

		var addr uint16 = 0xff00 + value
		a := uint16(mb.Cpu.Registers.A)
		mb.SetItem(&addr, &a)
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// LDH A, (a8) - Load A with value at address $FF00 + 8-bit immediate (240)
	0xf0: func(mb *Motherboard, value uint16) OpCycles {

		var addr uint16 = 0xff00 + value
		a := mb.GetItem(&addr)
		mb.Cpu.Registers.A = a
		mb.Cpu.Registers.PC += 2
		return 12
	},

	/****************************** 0xn1 **********************/
	// LD BC, d16 - Load 16-bit immediate into BC (1)
	0x01: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.SetBC(value)
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// LD DE, d16 - Load 16-bit immediate into DE (17)
	0x11: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.SetDE(value)
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// LD HL, d16 - Load 16-bit immediate into HL (33)
	0x21: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.SetHL(value)
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// LD SP, d16 - Load 16-bit immediate into SP (49)
	0x31: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.SP = value
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// LD B, C - Copy C to B (65)
	0x41: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Registers.C
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD D, C - Copy C to D (81)
	0x51: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Registers.C
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD H, C - Copy C to H (97)
	0x61: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Registers.C
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD (HL), C - Save C at address pointed to by HL (113)
	0x71: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		cr := uint16(mb.Cpu.Registers.C)
		mb.SetItem(&hl, &cr)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD A, C - Add C to A (129)
	0x81: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SUB C - Subtract C from A (145)
	0x91: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// AND C - Logical AND C against A (161)
	0xa1: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// OR C - Logical OR C against A (177)
	0xb1: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// POP BC - Pop two bytes from stack into BC (193)
	0xc1: func(mb *Motherboard, value uint16) OpCycles {

		var pch, pcl uint8
		spadd1 := mb.Cpu.Registers.SP + 1
		pch = mb.GetItem(&spadd1)
		pcl = mb.GetItem(&mb.Cpu.Registers.SP)

		mb.Cpu.SetBC((uint16(pch) << 8) | uint16(pcl))

		mb.Cpu.Registers.SP += 2
		mb.Cpu.Registers.PC += 1
		return 12
	},

	// POP DE - Pop two bytes from stack into DE (209)
	0xd1: func(mb *Motherboard, value uint16) OpCycles {

		var pch, pcl uint8
		spadd1 := mb.Cpu.Registers.SP + 1
		pch = mb.GetItem(&spadd1)
		pcl = mb.GetItem(&mb.Cpu.Registers.SP)

		mb.Cpu.SetDE((uint16(pch) << 8) | uint16(pcl))

		mb.Cpu.Registers.SP += 2
		mb.Cpu.Registers.PC += 1
		return 12
	},

	// POP HL - Pop two bytes from stack into HL (225)
	0xe1: func(mb *Motherboard, value uint16) OpCycles {

		var pch, pcl uint8
		spadd1 := mb.Cpu.Registers.SP + 1
		pch = mb.GetItem(&spadd1)
		pcl = mb.GetItem(&mb.Cpu.Registers.SP)

		mb.Cpu.SetHL((uint16(pch) << 8) | uint16(pcl))

		mb.Cpu.Registers.SP += 2
		mb.Cpu.Registers.PC += 1
		return 12
	},

	// POP AF - Pop two bytes from stack into AF (241)
	0xf1: func(mb *Motherboard, value uint16) OpCycles {

		spadd1 := mb.Cpu.Registers.SP + 1
		mb.Cpu.Registers.A = mb.GetItem(&spadd1)
		mb.Cpu.Registers.F = mb.GetItem(&mb.Cpu.Registers.SP) & 0xF0 & 0xF0

		mb.Cpu.Registers.SP += 2
		mb.Cpu.Registers.PC += 1
		return 12
	},

	/****************************** 0xn2 **********************/
	// LD (BC), A - Save A to address pointed by BC (2)
	0x02: func(mb *Motherboard, value uint16) OpCycles {

		bc := mb.Cpu.BC()
		a := (uint16)(mb.Cpu.Registers.A)
		mb.SetItem(&bc, &a)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD (DE), A - Save A at address pointed to by DE (18)
	0x12: func(mb *Motherboard, value uint16) OpCycles {

		de := mb.Cpu.DE()
		a := uint16(mb.Cpu.Registers.A)
		mb.SetItem(&de, &a)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD (HL+), A - Save A at address pointed by HL, increment HL (34)
	0x22: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		a := uint16(mb.Cpu.Registers.A)
		mb.SetItem(&hl, &a)
		hl += 1
		mb.Cpu.SetHL(hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD (HL-), A - Save A at address pointed by HL, decrement HL (50)
	0x32: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		a := uint16(mb.Cpu.Registers.A)
		mb.SetItem(&hl, &a)
		hl -= 1
		mb.Cpu.SetHL(hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD B, D - Copy D to B (66)
	0x42: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Registers.D
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD D, D - Copy D to D (82)
	0x52: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Registers.D = mb.Cpu.Registers.D
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD H, D - Copy D to H (98)
	0x62: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Registers.D
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD (HL), D - Save D at address pointed to by HL (114)
	0x72: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		d := uint16(mb.Cpu.Registers.D)
		mb.SetItem(&hl, &d)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD A, D - Add D to A (130)
	0x82: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SUB D - Subtract D from A (146)
	0x92: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// AND D - Logical AND D against A (162)
	0xa2: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// OR D - Logical OR D against A (178)
	0xb2: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// JP NZ, a16 - Absolute jump to 16-bit location if last result was not zero (194)
	0xc2: func(mb *Motherboard, value uint16) OpCycles {

		if !mb.Cpu.IsFlagZSet() {
			mb.Cpu.Registers.PC = value
			return 16
		}
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// JP NC, a16 - Absolute jump to 16-bit location if last result caused no carry (210)
	0xd2: func(mb *Motherboard, value uint16) OpCycles {

		if !mb.Cpu.IsFlagCSet() {
			mb.Cpu.Registers.PC = value
			return 16
		}
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// LD (C), A - Save A at address $FF00 + register C (226)
	0xe2: func(mb *Motherboard, value uint16) OpCycles {

		var addr uint16 = 0xff00 + uint16(mb.Cpu.Registers.C)
		a := uint16(mb.Cpu.Registers.A)
		mb.SetItem(&addr, &a)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD A, (C) - Load A with value at address $FF00 + register C (242)
	0xf2: func(mb *Motherboard, value uint16) OpCycles {

		var addr uint16 = 0xff00 + uint16(mb.Cpu.Registers.C)
		a := mb.GetItem(&addr)
		mb.Cpu.Registers.A = a
		mb.Cpu.Registers.PC += 1
		return 8
	},

	/****************************** 0xn3 **********************/
	// // INC BC - Increment BC (3)
	0x03: func(mb *Motherboard, value uint16) OpCycles {

		bc := mb.Cpu.BC()
		bc += 1
		mb.Cpu.SetBC(bc)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// INC DE - Increment DE (19)
	0x13: func(mb *Motherboard, value uint16) OpCycles {

		de := mb.Cpu.DE()
		de += 1
		mb.Cpu.SetDE(de)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// INC HL - Increment HL (35)
	0x23: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		hl += 1
		mb.Cpu.SetHL(hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// INC SP - Increment SP (51)
	0x33: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.SP += 1
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD B, E - Copy E to B (67)
	0x43: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Registers.E
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD D, E - Copy E to D (83)
	0x53: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Registers.E
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD H, E - Copy E to H (99)
	0x63: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Registers.E
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD (HL), E - Save E at address pointed to by HL (115)
	0x73: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		e := uint16(mb.Cpu.Registers.E)
		mb.SetItem(&hl, &e)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD A, E - Add E to A (131)
	0x83: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SUB E - Subtract E from A (147)
	0x93: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// AND E - Logical AND E against A (163)
	0xa3: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// OR E - Logical OR E against A (179)
	0xb3: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// JP a16 - Absolute jump to 16-bit location (195)
	0xc3: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC = value
		return 16
	},

	// 0xd3 - Illegal opcode
	// 0xe3 - Illegal opcode

	// DI - Disable interrupts (243)
	0xf3: func(mb *Motherboard, value uint16) OpCycles {
		// logger.Error("DI - Disable interrupts (243)")
		// mb.Cpu.Interrupts.Master_Enable = false
		mb.Cpu.Interrupts.InterruptsOn = false
		mb.Cpu.Registers.PC += 1
		return 4
	},

	/****************************** 0xn4 **********************/
	// // INC B - Increment B (4)
	0x04: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Inc(mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// INC D - Increment D (20)
	0x14: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Inc(mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// INC H - Increment H (36)
	0x24: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Inc(mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// INC (HL) - Increment value pointed by HL (52)
	0x34: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		v := mb.GetItem(&hl)
		v = mb.Cpu.Inc(v)

		v16 := uint16(v)
		mb.SetItem(&hl, &v16)
		mb.Cpu.Registers.PC += 1
		return 12
	},

	// LD B, H - Copy H to B (68)
	0x44: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Registers.H
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD D, H - Copy H to D (84)
	0x54: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Registers.H
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD H, H - Copy H to H (100)
	0x64: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Registers.H = mb.Cpu.Registers.H
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD (HL), H - Save H at address pointed to by HL (116)
	0x74: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		h := uint16(mb.Cpu.Registers.H)
		mb.SetItem(&hl, &h)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD A, H - Add H to A (132)
	0x84: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SUB H - Subtract H from A (148)
	0x94: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// AND H - Logical AND H against A (164)
	0xa4: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// OR H - Logical OR H against A (180)
	0xb4: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CALL NZ, a16 - Call routine at 16-bit location if last result was not zero (196)
	0xc4: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 3
		if !mb.Cpu.IsFlagZSet() {
			sp1 := mb.Cpu.Registers.SP - 1
			sp2 := mb.Cpu.Registers.SP - 2

			pch := (mb.Cpu.Registers.PC >> 8) & 0xff
			pcl := mb.Cpu.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			mb.Cpu.Registers.SP -= 2
			mb.Cpu.Registers.PC = value
			return 24
		} else {
			return 12
		}
	},

	// CALL NC, a16 - Call routine at 16-bit location if last result caused no carry (212)
	0xd4: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 3
		if !mb.Cpu.IsFlagCSet() {
			sp1 := mb.Cpu.Registers.SP - 1
			sp2 := mb.Cpu.Registers.SP - 2

			pch := (mb.Cpu.Registers.PC >> 8) & 0xff
			pcl := mb.Cpu.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			mb.Cpu.Registers.SP -= 2
			mb.Cpu.Registers.PC = value
			return 24
		} else {
			return 12
		}
	},

	// 0xe4 - Illegal opcode
	// 0xf4 - Illegal opcode

	/****************************** 0xn5 **********************/
	// DEC B - Decrement B (5)
	0x05: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Dec(mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// DEC D - Decrement D (21)
	0x15: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Dec(mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// DEC H - Decrement H (37)
	0x25: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Dec(mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// DEC (HL) - Decrement value pointed by HL (53)
	0x35: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		v := mb.GetItem(&hl)
		v = mb.Cpu.Dec(v)

		v16 := uint16(v)
		mb.SetItem(&hl, &v16)
		mb.Cpu.Registers.PC += 1
		return 12
	},

	// LD B, L - Copy L to B (69)
	0x45: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Registers.L
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD D, L - Copy L to D (85)
	0x55: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Registers.L
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD H, L - Copy L to H (101)
	0x65: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Registers.L
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD (HL), L - Save L at address pointed to by HL (117)
	0x75: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		l := uint16(mb.Cpu.Registers.L)
		mb.SetItem(&hl, &l)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD A, L - Add L to A (133)
	0x85: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SUB L - Subtract L from A (149)
	0x95: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// AND L - Logical AND L against A (165)
	0xa5: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// OR L - Logical OR L against A (181)
	0xb5: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// PUSH BC - Push BC onto stack (197)
	0xc5: func(mb *Motherboard, value uint16) OpCycles {

		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		br := uint16(mb.Cpu.Registers.B)
		cr := uint16(mb.Cpu.Registers.C)
		mb.SetItem(&sp1, &br)
		mb.SetItem(&sp2, &cr)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC += 1
		return 16
	},

	// PUSH DE - Push DE onto stack (213)
	0xd5: func(mb *Motherboard, value uint16) OpCycles {

		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		dr := uint16(mb.Cpu.Registers.D)
		er := uint16(mb.Cpu.Registers.E)
		mb.SetItem(&sp1, &dr)
		mb.SetItem(&sp2, &er)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC += 1
		return 16
	},

	// PUSH HL - Push HL onto stack (229)
	0xe5: func(mb *Motherboard, value uint16) OpCycles {

		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		hr := uint16(mb.Cpu.Registers.H)
		lr := uint16(mb.Cpu.Registers.L)
		mb.SetItem(&sp1, &hr)
		mb.SetItem(&sp2, &lr)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC += 1
		return 16
	},

	// PUSH AF - Push AF onto stack (229)
	0xf5: func(mb *Motherboard, value uint16) OpCycles {

		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		ar := uint16(mb.Cpu.Registers.A)
		fr := uint16(mb.Cpu.Registers.F)
		mb.SetItem(&sp1, &ar)
		mb.SetItem(&sp2, &fr)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC += 1
		return 16
	},

	/****************************** 0xn6 **********************/
	// LD B, d8 - Load 8-bit immediate into B (6)
	0x06: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = uint8(value)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD D, d8 - Load 8-bit immediate into D (22)
	0x16: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = uint8(value)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD H, d8 - Load 8-bit immediate into H (38)
	0x26: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = uint8(value)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD (HL), d8 - Save 8-bit immediate to address pointed by HL (54)
	0x36: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		value &= 0xff
		mb.SetItem(&hl, &value)
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// LD B, (HL) - Copy value pointed by HL to B (70)
	0x46: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.B = mb.GetItem(&hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD D, (HL) - Copy value pointed by HL to D (86)
	0x56: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.D = mb.GetItem(&hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD H, (HL) - Copy value pointed by HL to H (102)
	0x66: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.H = mb.GetItem(&hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// HALT - Power down CPU until an interrupt occurs (118)
	0x76: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Halted = true
		return 4
	},

	// ADD A, (HL) - Add value pointed by HL to A (134)
	0x86: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.GetItem(&hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// SUB (HL) - Subtract value pointed by HL from A (150)
	0x96: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.GetItem(&hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// AND (HL) - Logical AND value pointed by HL against A (166)
	0xa6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.GetItem(&hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// OR (HL) - Logical OR value pointed by HL against A (182)
	0xb6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.GetItem(&hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD, d8 - Add 8-bit immediate to A (198)
	0xc6: func(mb *Motherboard, value uint16) OpCycles {

		v := uint8(value)
		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, v)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SUB d8 - Subtract 8-bit immediate from A (214)
	0xd6: func(mb *Motherboard, value uint16) OpCycles {

		v := uint8(value)
		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, v)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// AND d8 - Logical AND 8-bit immediate against A (230)
	0xe6: func(mb *Motherboard, value uint16) OpCycles {

		v := uint8(value)
		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, v)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// OR d8 - Logical OR 8-bit immediate against A (246)
	0xf6: func(mb *Motherboard, value uint16) OpCycles {

		v := uint8(value)
		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, v)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn7 **********************/
	// RLCA - Rotate A left. Old bit 7 to Carry flag (7)
	0x07: func(mb *Motherboard, value uint16) OpCycles {

		a := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(a, 7) {
			mb.Cpu.SetFlagC()
			a = (a << 1) + 1
		} else {
			mb.Cpu.ResetFlagC()
			a = (a << 1)
		}

		mb.Cpu.Registers.A = a
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// RLA - Rotate A left through Carry flag (23)
	0x17: func(mb *Motherboard, value uint16) OpCycles {

		a := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(a, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register A to the left by one bit
		mb.Cpu.Registers.A = (a << 1) & 0xff
		if oldCarry {
			mb.Cpu.Registers.A |= 0x01
		}

		mb.Cpu.Registers.PC += 1
		return 4
	},

	// DAA - Decimal adjust A (39)
	0x27: func(mb *Motherboard, value uint16) OpCycles {

		a := int16(mb.Cpu.Registers.A)

		var corr int16 = 0

		if mb.Cpu.IsFlagHSet() {
			corr |= 0x06
		}

		if mb.Cpu.IsFlagCSet() {
			corr |= 0x60
		}

		if mb.Cpu.IsFlagNSet() {
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
			internal.SetBit(&flag, FLAGZ)
		}

		if corr&0x60 != 0 {
			internal.SetBit(&flag, FLAGC)
		}

		mb.Cpu.Registers.F &= 0b01000000
		mb.Cpu.Registers.F |= flag

		mb.Cpu.Registers.A = uint8(a)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SCF - Set carry flag (55)
	0x37: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.SetFlagC()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD B, A - Copy A to B (71)
	0x47: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.B = mb.Cpu.Registers.A
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD D, A - Copy A to D (87)
	0x57: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.D = mb.Cpu.Registers.A
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD H, A - Copy A to H (103)
	0x67: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.H = mb.Cpu.Registers.A
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD (HL), A - Save A at address pointed to by HL (119)
	0x77: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		a := uint16(mb.Cpu.Registers.A)
		mb.SetItem(&hl, &a)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD A, A - Add A to A (135)
	0x87: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AddSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SUB A - Subtract A from A (151)
	0x97: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SubSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// AND A - Logical AND A against A (167)
	0xa7: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AndSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// OR A - Logical OR A against A (183)
	0xb7: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.OrSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// RST 00H - Push present address onto stack. Jump to address $0000 (199)
	0xc7: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC = 0x00
		return 16
	},

	// RST 10H - Push present address onto stack. Jump to address $0010 (215)
	0xd7: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC = 0x10
		return 16
	},

	// RST 20 H - Push present address onto stack. Jump to address $0020 (231)
	0xe7: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC = 0x20
		return 16
	},

	// RST 30H - Push present address onto stack. Jump to address $0030 (247)
	0xf7: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2
		mb.Cpu.Registers.PC = 0x30
		return 16
	},

	/****************************** 0xn8 **********************/
	// LD (a16), SP - Save SP at given address (8)
	// value is the address
	0x08: func(mb *Motherboard, value uint16) OpCycles {

		addr1 := value
		value1 := mb.Cpu.Registers.SP & 0xFF

		addr2 := value + 1
		value2 := (mb.Cpu.Registers.SP >> 8) & 0xFF

		mb.SetItem(&addr1, &value1)
		mb.SetItem(&addr2, &value2)

		mb.Cpu.Registers.PC += 3
		return 20
	},

	// JR r8 - Relative jump by signed immediate (24)
	0x18: func(mb *Motherboard, value uint16) OpCycles {

		v := int16(value^0x80) - 0x80                   // convert to signed int
		mb.Cpu.Registers.PC += (2 + uint16(v)) & 0xffff // add to PC
		return 12
	},

	// JR Z, r8 - Relative jump by signed immediate if Z flag is set (40)
	0x28: func(mb *Motherboard, value uint16) OpCycles {

		v := int16(value^0x80) - 0x80 // convert to signed int

		if mb.Cpu.IsFlagZSet() {
			mb.Cpu.Registers.PC += (2 + uint16(v)) & 0xffff // add to PC
			return 12
		}

		mb.Cpu.Registers.PC += 2
		return 8
	},

	// JR C, r8 - Relative jump by signed immediate if C flag is set (56)
	0x38: func(mb *Motherboard, value uint16) OpCycles {

		v := int16(value^0x80) - 0x80 // convert to signed int

		if mb.Cpu.IsFlagCSet() {
			mb.Cpu.Registers.PC += (2 + uint16(v)) & 0xffff // add to PC
			return 12
		}

		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD C, B - Copy B to C (72)
	0x48: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Registers.B
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD E, B - Copy B to E (88)
	0x58: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Registers.B
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD L, B - Copy B to L (104)
	0x68: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Registers.B
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD A, B - Copy B to A (120)
	0x78: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Registers.B
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// ADC A, B - Add B and carry flag to A (136)
	0x88: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SBC A, B - Subtract B and carry flag from A (152)
	0x98: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// XOR B - Logical XOR B against A (168)
	0xa8: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CP B - Compare B against A (184)
	0xb8: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.B)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// RET Z - Return if last result was zero (200)
	0xc8: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		if mb.Cpu.IsFlagZSet() {
			nsp := mb.Cpu.Registers.SP + 1
			pcl := mb.GetItem(&mb.Cpu.Registers.SP)
			pch := mb.GetItem(&nsp)
			mb.Cpu.Registers.SP += 2
			mb.Cpu.Registers.PC = uint16(pch)<<8 | uint16(pcl)
			return 20
		} else {
			return 8
		}
	},

	// RET C - Return if last result caused carry (216)
	0xd8: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		if mb.Cpu.IsFlagCSet() {
			nsp := mb.Cpu.Registers.SP + 1
			pcl := mb.GetItem(&mb.Cpu.Registers.SP)
			pch := mb.GetItem(&nsp)
			mb.Cpu.Registers.SP += 2
			mb.Cpu.Registers.PC = uint16(pch)<<8 | uint16(pcl)
			return 20
		} else {
			return 8
		}
	},

	// ADD SP, r8 - Add signed 8-bit immediate to SP (232)
	0xe8: func(mb *Motherboard, value uint16) OpCycles {

		value &= 0xff
		var i8 int8 = int8((value ^ 0x80) - 0x80)
		sp := int32(mb.Cpu.Registers.SP)
		r := sp + int32(i8)
		i8_32 := int32(i8)

		mb.Cpu.ClearAllFlags()

		if ((sp&0xf)+(i8_32&0xf))&0x10 > 0xf {
			mb.Cpu.SetFlagH()
		}

		if (sp^i8_32^r)&0x100 == 0x100 {
			mb.Cpu.SetFlagC()
		}

		// var i8 int8 = int8((value ^ 0x80) - 0x80)
		// r := int32(mb.Cpu.Registers.SP) + int32(i8)
		// sp := int32(mb.Cpu.Registers.SP)

		// mb.Cpu.ClearAllFlags()
		// if (sp^int32(i8)^r)&0x100 == 0x100 {
		// 	mb.Cpu.SetFlagC()
		// }

		// // if (int32(mb.Cpu.Registers.SP)^int32(i8)^r)&0x10 != 0x0 {
		// // 	mb.Cpu.SetFlagH()
		// // }

		// if (sp&0xf)+(int32(i8)&0xf)&0x10 == 0x10 {
		// 	mb.Cpu.SetFlagH()
		// }
		mb.Cpu.Registers.SP = uint16(r)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// LD HL, SP+r8 - Add signed 8-bit immediate to SP (232)
	0xf8: func(mb *Motherboard, value uint16) OpCycles {

		value &= 0xff
		var i8 int8 = int8((value ^ 0x80) - 0x80)
		// var i8 int8 = int8(value)
		sp := int32(mb.Cpu.Registers.SP)
		i8_32 := int32(i8)
		r := sp + i8_32

		mb.Cpu.SetHL(uint16(r))

		mb.Cpu.ClearAllFlags()
		if ((sp&0xf)+(i8_32&0xf))&0x10 > 0xf {
			mb.Cpu.SetFlagH()
		}

		if (sp^i8_32^r)&0x100 == 0x100 {
			mb.Cpu.SetFlagC()
		}

		// if (int32(mb.Cpu.Registers.SP)^int32(i8)^r)&0x10 == 0x10 {
		// 	mb.Cpu.SetFlagC()
		// }

		// if (int32(mb.Cpu.Registers.SP)^int32(i8)^r)&0x100 == 0x100 {
		// 	mb.Cpu.SetFlagH()
		// }

		mb.Cpu.Registers.PC += 2
		return 12
	},

	/****************************** 0xn9 **********************/
	// ADD HL, BC - Add BC to HL (9)
	0x09: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.AddSetFlags16(mb.Cpu.HL(), mb.Cpu.BC())

		mb.Cpu.SetHL(uint16(hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD HL, DE - Add DE to HL (25)
	0x19: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.AddSetFlags16(mb.Cpu.HL(), mb.Cpu.DE())
		mb.Cpu.SetHL(uint16(hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD HL, HL - Add HL to HL (41)
	0x29: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.AddSetFlags16(mb.Cpu.HL(), mb.Cpu.HL())
		mb.Cpu.SetHL(uint16(hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADD HL, SP - Add SP to HL (57)
	0x39: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.AddSetFlags16(mb.Cpu.HL(), mb.Cpu.Registers.SP)
		mb.Cpu.SetHL(uint16(hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD C, C - Copy C to C (73)
	0x49: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Registers.C = mb.Cpu.Registers.C
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD E, C - Copy C to E (89)
	0x59: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Registers.C
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD L, C - Copy C to L (105)
	0x69: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Registers.C
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD A, C - Copy C to A (121)
	0x79: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Registers.C
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// ADC A, C - Add C and carry flag to A (137)
	0x89: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SBC A, C - Subtract C and carry flag from A (153)
	0x99: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// XOR C - Logical XOR C against A (169)
	0xa9: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CP C - Compare C against A (185)
	0xb9: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// RET - Pop two bytes from stack & jump to that address (201)
	0xc9: func(mb *Motherboard, value uint16) OpCycles {

		sp2 := mb.Cpu.Registers.SP + 1
		pcl := mb.GetItem(&mb.Cpu.Registers.SP)
		pch := mb.GetItem(&sp2)
		mb.Cpu.Registers.SP += 2
		mb.Cpu.Registers.PC = uint16(pch)<<8 | uint16(pcl)
		return 16
	},

	// RETI - Pop two bytes from stack & jump to that address then enable interrupts (217)
	0xd9: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Interrupts.Master_Enable = true
		mb.Cpu.Interrupts.InterruptsEnabling = true
		sp2 := mb.Cpu.Registers.SP + 1
		pcl := mb.GetItem(&mb.Cpu.Registers.SP)
		pch := mb.GetItem(&sp2)
		mb.Cpu.Registers.SP += 2
		mb.Cpu.Registers.PC = uint16(pch)<<8 | uint16(pcl)
		return 16
	},

	// JP (HL) - Jump to address contained in HL (233)
	0xe9: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC = mb.Cpu.HL()
		return 4
	},

	// LD SP, HL - Copy HL to SP (233)
	0xf9: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.SP = mb.Cpu.HL()
		mb.Cpu.Registers.PC += 1
		return 8
	},

	/****************************** 0xna **********************/
	// LD A, (BC) - Load A from address pointed to by BC (10)
	0x0A: func(mb *Motherboard, value uint16) OpCycles {

		bc := mb.Cpu.BC()
		a := mb.GetItem(&bc)
		mb.Cpu.Registers.A = uint8(a)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD A, (DE) - Load A with data from address pointed to by DE (26)
	0x1A: func(mb *Motherboard, value uint16) OpCycles {

		de := mb.Cpu.DE()
		a := mb.GetItem(&de)
		mb.Cpu.Registers.A = uint8(a)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD A, (HL+) - Load A with data from address pointed to by HL, increment HL (42)
	0x2A: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		a := mb.GetItem(&hl)
		mb.Cpu.Registers.A = uint8(a)
		hl += 1
		mb.Cpu.SetHL(hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD A, (HL-) - Load A with data from address pointed to by HL, decrement HL (58)
	0x3A: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		a := mb.GetItem(&hl)
		mb.Cpu.Registers.A = uint8(a)
		hl -= 1
		mb.Cpu.SetHL(hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD C, D - Copy D to C (74)
	0x4A: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Registers.D
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD E, D - Copy D to E (90)
	0x5A: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Registers.D
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD L, D - Copy D to L (106)
	0x6A: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Registers.D
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD A, D - Copy D to A (122)
	0x7A: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Registers.D
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// ADC A, D - Add D and carry flag to A (138)
	0x8A: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SBC A, D - Subtract D and carry flag from A (154)
	0x9A: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// XOR D - Logical XOR D against A (170)
	0xaa: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CP D - Compare D against A (186)
	0xba: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.D)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// JP Z, a16 - Absolute jump to 16-bit location if Z flag is set (202)
	0xca: func(mb *Motherboard, value uint16) OpCycles {

		if mb.Cpu.IsFlagZSet() {
			mb.Cpu.Registers.PC = value
			return 16
		}
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// JP C, a16 - Absolute jump to 16-bit location if C flag is set (218)
	0xda: func(mb *Motherboard, value uint16) OpCycles {

		if mb.Cpu.IsFlagCSet() {
			mb.Cpu.Registers.PC = value
			return 16
		}
		mb.Cpu.Registers.PC += 3
		return 12
	},

	// LD (a16), A - Save A at given address (234)
	0xea: func(mb *Motherboard, value uint16) OpCycles {

		a := uint16(mb.Cpu.Registers.A)
		mb.SetItem(&value, &a)
		mb.Cpu.Registers.PC += 3
		return 16
	},

	// LD A, (a16) - Load A from given address (250)
	0xfa: func(mb *Motherboard, value uint16) OpCycles {

		a := mb.GetItem(&value)
		mb.Cpu.Registers.A = uint8(a)
		mb.Cpu.Registers.PC += 3
		return 16
	},

	/****************************** 0xnb **********************/
	// DEC BC - Decrement BC (11)
	0x0B: func(mb *Motherboard, value uint16) OpCycles {

		bc := mb.Cpu.BC()
		bc -= 1
		mb.Cpu.SetBC(bc)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// DEC DE - Decrement DE (27)
	0x1B: func(mb *Motherboard, value uint16) OpCycles {

		de := mb.Cpu.DE()
		de -= 1
		mb.Cpu.SetDE(de)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// DEC HL - Decrement HL (43)
	0x2B: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		hl -= 1
		mb.Cpu.SetHL(hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// DEC SP - Decrement SP (59)
	0x3B: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.SP -= 1
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD C, E - Copy E to C (75)
	0x4B: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Registers.E
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD E, E - Copy E to E (91)
	0x5B: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Registers.E = mb.Cpu.Registers.E
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD L, E - Copy E to L (107)
	0x6B: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Registers.E
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD A, E - Copy E to A (123)
	0x7B: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Registers.E
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// ADC A, E - Add E and carry flag to A (139)
	0x8B: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SBC A, E - Subtract E and carry flag from A (155)
	0x9B: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// XOR E - Logical XOR E against A (171)
	0xab: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CP E - Compare E against A (187)
	0xbb: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// PREFIX CB - CB prefix (203) --- isn't callable
	0xcb: func(mb *Motherboard, value uint16) OpCycles {
		logger.Fatal("0xcb is not callable, this is just here for documentation")
		return 101
	},

	// 0xdb - Illegal instruction
	// 0xeb - Illegal instruction
	// EI - Enable interrupts (235)
	0xfb: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Interrupts.Master_Enable = true
		mb.Cpu.Interrupts.InterruptsEnabling = true
		mb.Cpu.Registers.PC += 1
		return 4
	},

	/****************************** 0xnc **********************/
	// INC C - Increment C (12)
	0x0C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Inc(mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// INC E - Increment E (28)
	0x1C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Inc(mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// INC L - Increment L (44)
	0x2C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Inc(mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// INC A - Increment A (60)
	0x3C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Inc(mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD C, H - Copy H to C (76)
	0x4C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Registers.H
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD E, H - Copy H to E (92)
	0x5C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Registers.H
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD L, H - Copy H to L (108)
	0x6C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Registers.H
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD A, H - Copy H to A (124)
	0x7C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Registers.H
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// ADC A, H - Add H and carry flag to A (140)
	0x8C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SBC A, H - Subtract H and carry flag from A (156)
	0x9C: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// XOR H - Logical XOR H against A (172)
	0xac: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CP H - Compare H against A (188)
	0xbc: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.H)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CALL Z, a16 - Call routine at 16-bit address if Z flag is set (204)
	0xcc: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 3

		if mb.Cpu.IsFlagZSet() {
			// mb.Cpu.Dump("PRE CALL Z, Flag Z is set")

			sp1 := mb.Cpu.Registers.SP - 1
			sp2 := mb.Cpu.Registers.SP - 2

			pch := (mb.Cpu.Registers.PC >> 8) & 0xff
			pcl := mb.Cpu.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			mb.Cpu.Registers.SP -= 2

			mb.Cpu.Registers.PC = value
			// mb.Cpu.Dump("POST CALL Z, Flag Z is set")
			// os.Exit(0)
			return 24
		}

		return 12
	},

	// CALL C, a16 - Call routine at 16-bit address if C flag is set (220)
	0xdc: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 3

		if mb.Cpu.IsFlagCSet() {
			sp1 := mb.Cpu.Registers.SP - 1
			sp2 := mb.Cpu.Registers.SP - 2

			pch := (mb.Cpu.Registers.PC >> 8) & 0xff
			pcl := mb.Cpu.Registers.PC & 0xff
			mb.SetItem(&sp1, &pch)
			mb.SetItem(&sp2, &pcl)
			mb.Cpu.Registers.SP -= 2

			mb.Cpu.Registers.PC = value
			return 24
		}
		return 12
	},

	// 0xec - Illegal instruction
	// 0xfc - Illegal instruction

	/****************************** 0xnd **********************/
	// DEC C - Decrement C (13)
	0x0D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Dec(mb.Cpu.Registers.C)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// DEC E - Decrement E (29)
	0x1D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Dec(mb.Cpu.Registers.E)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// DEC L - Decrement L (45)
	0x2D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Dec(mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// DEC A - Decrement A (61)
	0x3D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Dec(mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD C, L - Copy L to C (77)
	0x4D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Registers.L
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD E, L - Copy L to E (93)
	0x5D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Registers.L
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD L, L - Copy L to L (109)
	0x6D: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Registers.L = mb.Cpu.Registers.L
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD A, L - Copy L to A (125)
	0x7D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.Registers.L
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// ADC A, L - Add L and carry flag to A (141)
	0x8D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SBC A, L - Subtract L and carry flag from A (157)
	0x9D: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// XOR L - Logical XOR L against A (173)
	0xad: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CP L - Compare L against A (189)
	0xbd: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.L)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CALL a16 - Call routine at 16-bit address (205)
	0xcd: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 3
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2

		mb.Cpu.Registers.PC = value
		return 24
	},

	// 0xdd - Illegal instruction

	/****************************** 0xne **********************/
	// LD C, d8 - Load 8-bit immediate into C (14)
	0x0E: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = uint8(value)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD E, d8 - Load 8-bit immediate into E (30)
	0x1E: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = uint8(value)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD L, d8 - Load 8-bit immediate into L (46)
	0x2E: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = uint8(value)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD A, d8 - Load 8-bit immediate into A (62)
	0x3E: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = uint8(value)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// LD C, (HL) - Copy value pointed by HL to C (78)
	0x4E: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.C = mb.GetItem(&hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD E, (HL) - Copy value pointed by HL to E (94)
	0x5E: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.E = mb.GetItem(&hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD L, (HL) - Copy value pointed by HL to L (110)
	0x6E: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.L = mb.GetItem(&hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// LD A, (HL) - Copy value pointed by HL to A (126)
	0x7E: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.GetItem(&hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADC A, (HL) - Add value pointed by HL and carry flag to A (142)
	0x8E: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.GetItem(&hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// SBC A, (HL) - Subtract value pointed by HL and carry flag from A (158)
	0x9E: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.GetItem(&hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// XOR (HL) - Logical XOR value pointed by HL against A (174)
	0xae: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.GetItem(&hl))
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// CP (HL) - Compare value pointed by HL against A (190)
	0xbe: func(mb *Motherboard, value uint16) OpCycles {

		addr := mb.Cpu.HL()
		hl := mb.GetItem(&addr)
		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, hl)
		mb.Cpu.Registers.PC += 1
		return 8
	},

	// ADC A, d8 - Add 8-bit immediate and carry flag to A (206)
	0xce: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, uint8(value))
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SBC A, d8 - Subtract 8-bit immediate and carry flag from A (222)
	0xde: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, uint8(value))
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// XOR d8 - Logical XOR n against A (236)
	0xee: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, uint8(value))
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// CP d8 - Compare n against A (252)
	0xfe: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, uint8(value))
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xnf **********************/
	// RRCA - Rotate A right. Old bit 0 to Carry flag (15)
	0x0F: func(mb *Motherboard, value uint16) OpCycles {

		a := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(a, 0) {
			mb.Cpu.SetFlagC()
			a = (a >> 1) + 0x80
		} else {
			mb.Cpu.ResetFlagC()
			a = (a >> 1)
		}

		mb.Cpu.Registers.A = a
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// RRA - Rotate A right through Carry flag (31)
	0x1F: func(mb *Motherboard, value uint16) OpCycles {

		a := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(a, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register A to the right by one bit
		mb.Cpu.Registers.A = (a >> 1) & 0xff
		if oldCarry {
			mb.Cpu.Registers.A |= 0x80
		}

		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CPL - Complement A register (47)
	0x2F: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = ^mb.Cpu.Registers.A
		mb.Cpu.SetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CCF - Complement carry flag (63)
	0x3F: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		if mb.Cpu.IsFlagCSet() {
			mb.Cpu.ResetFlagC()
		} else {
			mb.Cpu.SetFlagC()
		}
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD C, A - Copy A to C (79)
	0x4F: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.C = mb.Cpu.Registers.A
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD E, A - Copy A to E (95)
	0x5F: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.E = mb.Cpu.Registers.A
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD L, A - Copy A to L (111)
	0x6F: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.L = mb.Cpu.Registers.A
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// LD A, A - Copy A to A (127)
	0x7F: func(mb *Motherboard, value uint16) OpCycles {

		// mb.Cpu.Registers.A = mb.Cpu.Registers.A
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// ADC A, A - Add A and carry flag to A (143)
	0x8F: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.AdcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// SBC A, A - Subtract A and carry flag from A (159)
	0x9F: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.SbcSetFlags8(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// XOR A - Logical XOR A against A (175)
	0xaf: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.A = mb.Cpu.XorSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// CP A - Compare A against A (191)
	0xbf: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.CpSetFlags(mb.Cpu.Registers.A, mb.Cpu.Registers.A)
		mb.Cpu.Registers.PC += 1
		return 4
	},

	// RST 08H - Push present address onto stack. Jump to address $0008 (207)
	0xcf: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2

		mb.Cpu.Registers.PC = 0x08
		return 16
	},

	// RST 18H - Push present address onto stack. Jump to address $0018 (223)
	0xdf: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2

		mb.Cpu.Registers.PC = 0x18
		return 16
	},

	// RST 28H - Push present address onto stack. Jump to address $0028 (239)
	0xef: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2

		mb.Cpu.Registers.PC = 0x28
		return 16
	},

	// RST 38H - Push present address onto stack. Jump to address $0038 (255)
	0xff: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.Registers.PC += 1
		sp1 := mb.Cpu.Registers.SP - 1
		sp2 := mb.Cpu.Registers.SP - 2

		pch := (mb.Cpu.Registers.PC >> 8) & 0xff
		pcl := mb.Cpu.Registers.PC & 0xff
		mb.SetItem(&sp1, &pch)
		mb.SetItem(&sp2, &pcl)
		mb.Cpu.Registers.SP -= 2

		mb.Cpu.Registers.PC = 0x38
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
	0x100: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RL B - Rotate B left through Carry flag (272) [minus 0xFF for CB prefix]
	0x110: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register B to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SLA B - Shift B left into Carry. LSB of B set to 0 (288) [minus 0xFF for CB prefix]
	0x120: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		b = (b << 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SWAP B - Swap upper & lower nibles of B (304) [minus 0xFF for CB prefix]
	0x130: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 0, B - Test bit 0 of B (320) [minus 0xFF for CB prefix]
	0x140: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 2, B - Test bit 2 of B (336) [minus 0xFF for CB prefix]
	0x150: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 4, B - Test bit 4 of B (352) [minus 0xFF for CB prefix]
	0x160: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 6, B - Test bit 6 of B (368) [minus 0xFF for CB prefix]
	0x170: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 0, B - Reset bit 0 of B (384) [minus 0xFF for CB prefix]
	0x180: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 2, B - Reset bit 2 of B (400) [minus 0xFF for CB prefix]
	0x190: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 4, B - Reset bit 4 of B (416) [minus 0xFF for CB prefix]
	0x1A0: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 6, B - Reset bit 6 of B (432) [minus 0xFF for CB prefix]
	0x1B0: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 0, B - Set bit 0 of B (448) [minus 0xFF for CB prefix]
	0x1C0: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 2, B - Set bit 2 of B (464) [minus 0xFF for CB prefix]
	0x1D0: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 4, B - Set bit 4 of B (480) [minus 0xFF for CB prefix]
	0x1E0: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 6, B - Set bit 6 of B (496) [minus 0xFF for CB prefix]
	0x1F0: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn1 **********************/
	// RLC C - Rotate C left. Old bit 7 to Carry flag (257) [minus 0xFF for CB prefix]
	0x101: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.C
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.C = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RL C - Rotate C left through Carry flag (273) [minus 0xFF for CB prefix]
	0x111: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.C
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register C to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.C = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SLA C - Shift C left into Carry. LSB of C set to 0 (289) [minus 0xFF for CB prefix]
	0x121: func(mb *Motherboard, value uint16) OpCycles {

		c := mb.Cpu.Registers.C
		mb.Cpu.ResetAllFlags()

		if (c & 0x80) != 0 {
			mb.Cpu.SetFlagC()
		}

		result := (c << 1) & 0xff
		mb.Cpu.Registers.C = result

		if result == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SWAP C - Swap upper & lower nibles of C (305) [minus 0xFF for CB prefix]
	0x131: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.C
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.C = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 0, C - Test bit 0 of C (321) [minus 0xFF for CB prefix]
	0x141: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 2, C - Test bit 2 of C (337) [minus 0xFF for CB prefix]
	0x151: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 4, C - Test bit 4 of C (353) [minus 0xFF for CB prefix]
	0x161: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 6, C - Test bit 6 of C (369) [minus 0xFF for CB prefix]
	0x171: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 0, C - Reset bit 0 of C (385) [minus 0xFF for CB prefix]
	0x181: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 2, C - Reset bit 2 of C (401) [minus 0xFF for CB prefix]
	0x191: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 4, C - Reset bit 4 of C (417) [minus 0xFF for CB prefix]
	0x1A1: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 6, C - Reset bit 6 of C (433) [minus 0xFF for CB prefix]
	0x1B1: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 0, C - Set bit 0 of C (449) [minus 0xFF for CB prefix]
	0x1C1: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 2, C - Set bit 2 of C (465) [minus 0xFF for CB prefix]
	0x1D1: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 4, C - Set bit 4 of C (481) [minus 0xFF for CB prefix]
	0x1E1: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 6, C - Set bit 6 of C (497) [minus 0xFF for CB prefix]
	0x1F1: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn2 **********************/
	// RLC D - Rotate D left. Old bit 7 to Carry flag (258) [minus 0xFF for CB prefix]
	0x102: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.D
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.D = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RL D - Rotate D left through Carry flag (274) [minus 0xFF for CB prefix]
	0x112: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.D
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register D to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.D = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SLA D - Shift D left into Carry. LSB of D set to 0 (290) [minus 0xFF for CB prefix]
	0x122: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.D
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		b = (b << 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.D = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SWAP D - Swap upper & lower nibles of D (306) [minus 0xFF for CB prefix]
	0x132: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.D
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.D = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 0, D - Test bit 0 of D (322) [minus 0xFF for CB prefix]
	0x142: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 2, D - Test bit 2 of D (338) [minus 0xFF for CB prefix]
	0x152: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 4, D - Test bit 4 of D (354) [minus 0xFF for CB prefix]
	0x162: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 6, D - Test bit 6 of D (370) [minus 0xFF for CB prefix]
	0x172: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 0, D - Reset bit 0 of D (386) [minus 0xFF for CB prefix]
	0x182: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 2, D - Reset bit 2 of D (402) [minus 0xFF for CB prefix]
	0x192: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 4, D - Reset bit 4 of D (418) [minus 0xFF for CB prefix]
	0x1A2: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 6, D - Reset bit 6 of D (434) [minus 0xFF for CB prefix]
	0x1B2: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 0, D - Set bit 0 of D (450) [minus 0xFF for CB prefix]
	0x1C2: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 2, D - Set bit 2 of D (466) [minus 0xFF for CB prefix]
	0x1D2: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 4, D - Set bit 4 of D (482) [minus 0xFF for CB prefix]
	0x1E2: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 6, D - Set bit 6 of D (498) [minus 0xFF for CB prefix]
	0x1F2: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn3 **********************/
	// RLC E - Rotate E left. Old bit 7 to Carry flag (259) [minus 0xFF for CB prefix]
	0x103: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.E
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.E = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RL E - Rotate E left through Carry flag (275) [minus 0xFF for CB prefix]
	0x113: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.E
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register E to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.E = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SLA E - Shift E left into Carry. LSB of E set to 0 (291) [minus 0xFF for CB prefix]
	0x123: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.E
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		b = (b << 1) & 0xff
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.E = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SWAP E - Swap upper & lower nibles of E (307) [minus 0xFF for CB prefix]
	0x133: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.E
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.E = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 0, E - Test bit 0 of E (323) [minus 0xFF for CB prefix]
	0x143: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.E, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 2, E - Test bit 2 of E (339) [minus 0xFF for CB prefix]
	0x153: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.E, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 4, E - Test bit 4 of E (355) [minus 0xFF for CB prefix]
	0x163: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.E, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 6, E - Test bit 6 of E (371) [minus 0xFF for CB prefix]
	0x173: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.E, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 0, E - Reset bit 0 of E (387) [minus 0xFF for CB prefix]
	0x183: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 2, E - Reset bit 2 of E (403) [minus 0xFF for CB prefix]
	0x193: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 4, E - Reset bit 4 of E (419) [minus 0xFF for CB prefix]
	0x1A3: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 6, E - Reset bit 6 of E (435) [minus 0xFF for CB prefix]
	0x1B3: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 0, E - Set bit 0 of E (451) [minus 0xFF for CB prefix]
	0x1C3: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 2, E - Set bit 2 of E (467) [minus 0xFF for CB prefix]
	0x1D3: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 4, E - Set bit 4 of E (483) [minus 0xFF for CB prefix]
	0x1E3: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 6, E - Set bit 6 of E (499) [minus 0xFF for CB prefix]
	0x1F3: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn4 **********************/
	// RLC H - Rotate H left. Old bit 7 to Carry flag (260) [minus 0xFF for CB prefix]
	0x104: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.H
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.H = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RL H - Rotate H left through Carry flag (276) [minus 0xFF for CB prefix]
	0x114: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.H
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register H to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.H = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SLA H - Shift H left into Carry. LSB of H set to 0 (292) [minus 0xFF for CB prefix]
	0x124: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.H
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		b = (b << 1) & 0xff
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.H = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SWAP H - Swap upper & lower nibles of H (308) [minus 0xFF for CB prefix]
	0x134: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.H
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.H = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 0, H - Test bit 0 of H (324) [minus 0xFF for CB prefix]
	0x144: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.H, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 2, H - Test bit 2 of H (340) [minus 0xFF for CB prefix]
	0x154: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.H, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 4, H - Test bit 4 of H (356) [minus 0xFF for CB prefix]
	0x164: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.H, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 6, H - Test bit 6 of H (372) [minus 0xFF for CB prefix]
	0x174: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.H, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 0, H - Reset bit 0 of H (388) [minus 0xFF for CB prefix]
	0x184: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 2, H - Reset bit 2 of H (404) [minus 0xFF for CB prefix]
	0x194: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 4, H - Reset bit 4 of H (420) [minus 0xFF for CB prefix]
	0x1A4: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 6, H - Reset bit 6 of H (436) [minus 0xFF for CB prefix]
	0x1B4: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 0, H - Set bit 0 of H (452) [minus 0xFF for CB prefix]
	0x1C4: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 2, H - Set bit 2 of H (468) [minus 0xFF for CB prefix]
	0x1D4: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 4, H - Set bit 4 of H (484) [minus 0xFF for CB prefix]
	0x1E4: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 6, H - Set bit 6 of H (500) [minus 0xFF for CB prefix]
	0x1F4: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn5 **********************/
	// RLC L - Rotate L left. Old bit 7 to Carry flag (261) [minus 0xFF for CB prefix]
	0x105: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.L
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.L = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RL L - Rotate L left through Carry flag (277) [minus 0xFF for CB prefix]
	0x115: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.L
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register L to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.L = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SLA L - Shift L left into Carry. LSB of L set to 0 (293) [minus 0xFF for CB prefix]
	0x125: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.L
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		b = (b << 1) & 0xff
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.L = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SWAP L - Swap upper & lower nibles of L (309) [minus 0xFF for CB prefix]
	0x135: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.L
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.L = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 0, L - Test bit 0 of L (325) [minus 0xFF for CB prefix]
	0x145: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.L, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 2, L - Test bit 2 of L (341) [minus 0xFF for CB prefix]
	0x155: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.L, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 4, L - Test bit 4 of L (357) [minus 0xFF for CB prefix]
	0x165: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.L, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 6, L - Test bit 6 of L (373) [minus 0xFF for CB prefix]
	0x175: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.L, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 0, L - Reset bit 0 of L (389) [minus 0xFF for CB prefix]
	0x185: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 2, L - Reset bit 2 of L (405) [minus 0xFF for CB prefix]
	0x195: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 4, L - Reset bit 4 of L (421) [minus 0xFF for CB prefix]
	0x1A5: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 6, L - Reset bit 6 of L (437) [minus 0xFF for CB prefix]
	0x1B5: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 0, L - Set bit 0 of L (453) [minus 0xFF for CB prefix]
	0x1C5: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 2, L - Set bit 2 of L (469) [minus 0xFF for CB prefix]
	0x1D5: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 4, L - Set bit 4 of L (485) [minus 0xFF for CB prefix]
	0x1E5: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 6, L - Set bit 6 of L (501) [minus 0xFF for CB prefix]
	0x1F5: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn6 **********************/
	// RLC (HL) - Rotate value pointed by HL left. Old bit 7 to Carry flag (262) [minus 0xFF for CB prefix]
	0x106: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RL (HL) - Rotate value pointed by HL left through Carry flag (278) [minus 0xFF for CB prefix]
	0x116: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift value pointed by HL to the left by one bit
		b = (b << 1) & 0xff
		if oldCarry {
			b |= 0x01
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SLA (HL) - Shift value pointed by HL left into Carry. LSB of value pointed by HL set to 0 (294) [minus 0xFF for CB prefix]
	0x126: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		b = (b << 1) & 0xff
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SWAP (HL) - Swap upper & lower nibles of value pointed by HL (310) [minus 0xFF for CB prefix]
	0x136: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// BIT 0, (HL) - Test bit 0 of value pointed by HL (326) [minus 0xFF for CB prefix]
	0x146: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(b, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// BIT 2, (HL) - Test bit 2 of value pointed by HL (342) [minus 0xFF for CB prefix]
	0x156: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(b, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// BIT 4, (HL) - Test bit 4 of value pointed by HL (358) [minus 0xFF for CB prefix]
	0x166: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(b, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// BIT 6, (HL) - Test bit 6 of value pointed by HL (374) [minus 0xFF for CB prefix]
	0x176: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(b, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// RES 0, (HL) - Reset bit 0 of value pointed by HL (390) [minus 0xFF for CB prefix]
	0x186: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 0)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RES 2, (HL) - Reset bit 2 of value pointed by HL (406) [minus 0xFF for CB prefix]
	0x196: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 2)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RES 4, (HL) - Reset bit 4 of value pointed by HL (422) [minus 0xFF for CB prefix]
	0x1A6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 4)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RES 6, (HL) - Reset bit 6 of value pointed by HL (438) [minus 0xFF for CB prefix]
	0x1B6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 6)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 0, (HL) - Set bit 0 of value pointed by HL (454) [minus 0xFF for CB prefix]
	0x1C6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 0)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 2, (HL) - Set bit 2 of value pointed by HL (470) [minus 0xFF for CB prefix]
	0x1D6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 2)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 4, (HL) - Set bit 4 of value pointed by HL (486) [minus 0xFF for CB prefix]
	0x1E6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 4)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 6, (HL) - Set bit 6 of value pointed by HL (502) [minus 0xFF for CB prefix]
	0x1F6: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 6)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	/****************************** 0xn7 **********************/
	// RLC A - Rotate A left. Old bit 7 to Carry flag (263) [minus 0xFF for CB prefix]
	0x107: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
			b = (b << 1) + 0x01
		} else {
			mb.Cpu.ResetFlagC()
			b = (b << 1)
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.A = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RL A - Rotate A left through Carry flag (279) [minus 0xFF for CB prefix]
	0x117: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register A to the left by one bit
		b = (b << 1) & 0xff

		if oldCarry {
			b |= 0x01
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.A = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SLA A - Shift A left into Carry. LSB of A set to 0 (295) [minus 0xFF for CB prefix]
	0x127: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		b = (b << 1) & 0xff
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.A = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SWAP A - Swap upper & lower nibles of A (311) [minus 0xFF for CB prefix]
	0x137: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		b = ((b & 0x0f) << 4) | ((b & 0xf0) >> 4)

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.A = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 0, A - Test bit 0 of A (327) [minus 0xFF for CB prefix]
	0x147: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.A, 0) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 2, A - Test bit 2 of A (343) [minus 0xFF for CB prefix]
	0x157: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.A, 2) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 4, A - Test bit 4 of A (359) [minus 0xFF for CB prefix]
	0x167: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.A, 4) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 6, A - Test bit 6 of A (375) [minus 0xFF for CB prefix]
	0x177: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.A, 6) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 0, A - Reset bit 0 of A (391) [minus 0xFF for CB prefix]
	0x187: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 2, A - Reset bit 2 of A (407) [minus 0xFF for CB prefix]
	0x197: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 4, A - Reset bit 4 of A (423) [minus 0xFF for CB prefix]
	0x1A7: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 6, A - Reset bit 6 of A (439) [minus 0xFF for CB prefix]
	0x1B7: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 0, A - Set bit 0 of A (455) [minus 0xFF for CB prefix]
	0x1C7: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 0)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 2, A - Set bit 2 of A (471) [minus 0xFF for CB prefix]
	0x1D7: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 2)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 4, A - Set bit 4 of A (487) [minus 0xFF for CB prefix]
	0x1E7: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 4)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 6, A - Set bit 6 of A (503) [minus 0xFF for CB prefix]
	0x1F7: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 6)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn8 **********************/
	// RRC B - Rotate B right. Old bit 0 to Carry flag (264) [minus 0xFF for CB prefix]
	0x108: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			mb.Cpu.ResetFlagC()
			b = (b >> 1)
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RR B - Rotate B right through Carry flag (280) [minus 0xFF for CB prefix]
	0x118: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register B to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRA B - Shift B right into Carry. MSB doesn't change (296) [minus 0xFF for CB prefix]
	0x128: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register B to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(mb.Cpu.Registers.B, 7) {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRL B - Shift B right into Carry. MSB set to 0 (312) [minus 0xFF for CB prefix]
	0x138: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.B
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register B to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.B = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT B 1 - Test bit 1 of B (328) [minus 0xFF for CB prefix]
	0x148: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT B 3 - Test bit 3 of B (344) [minus 0xFF for CB prefix]
	0x158: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT B 5 - Test bit 5 of B (360) [minus 0xFF for CB prefix]
	0x168: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT B 7 - Test bit 7 of B (376) [minus 0xFF for CB prefix]
	0x178: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.B, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES B 1 - Reset bit 0 of B (392) [minus 0xFF for CB prefix]
	0x188: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES B 3 - Reset bit 3 of B (408) [minus 0xFF for CB prefix]
	0x198: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES B 5 - Reset bit 5 of B (424) [minus 0xFF for CB prefix]
	0x1A8: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES B 7 - Reset bit 7 of B (440) [minus 0xFF for CB prefix]
	0x1B8: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.B, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET B 1 - Set bit 1 of B (456) [minus 0xFF for CB prefix]
	0x1C8: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET B 3 - Set bit 3 of B (472) [minus 0xFF for CB prefix]
	0x1D8: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET B 5 - Set bit 5 of B (488) [minus 0xFF for CB prefix]
	0x1E8: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET B 7 - Set bit 7 of B (504) [minus 0xFF for CB prefix]
	0x1F8: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.B, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xn9 **********************/
	// RRC C - Rotate C right. Old bit 0 to Carry flag (265) [minus 0xFF for CB prefix]
	0x109: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.C
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			mb.Cpu.ResetFlagC()
			b = (b >> 1)
		}
		if b == 0 {
			mb.Cpu.SetFlagZ()
		}
		mb.Cpu.Registers.C = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RR C - Rotate C right through Carry flag (281) [minus 0xFF for CB prefix]
	0x119: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.C
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register C to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.C = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRA C - Shift C right into Carry. MSB doesn't change (297) [minus 0xFF for CB prefix]
	0x129: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.C
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register C to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(mb.Cpu.Registers.C, 7) {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.C = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRL C - Shift C right into Carry. MSB set to 0 (313) [minus 0xFF for CB prefix]
	0x139: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.C
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register C to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.C = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 1, C - Test bit 1 of C (329) [minus 0xFF for CB prefix]
	0x149: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 3, C - Test bit 3 of C (345) [minus 0xFF for CB prefix]
	0x159: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 5, C - Test bit 5 of C (361) [minus 0xFF for CB prefix]
	0x169: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 7, C - Test bit 7 of C (377) [minus 0xFF for CB prefix]
	0x179: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.C, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 1, C - Reset bit 1 of C (393) [minus 0xFF for CB prefix]
	0x189: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 3, C - Reset bit 3 of C (409) [minus 0xFF for CB prefix]
	0x199: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 5, C - Reset bit 5 of C (425) [minus 0xFF for CB prefix]
	0x1A9: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 7, C - Reset bit 7 of C (441) [minus 0xFF for CB prefix]
	0x1B9: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.C, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 1, C - Set bit 1 of C (457) [minus 0xFF for CB prefix]
	0x1C9: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 3, C - Set bit 3 of C (473) [minus 0xFF for CB prefix]
	0x1D9: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 5, C - Set bit 5 of C (489) [minus 0xFF for CB prefix]
	0x1E9: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 7, C - Set bit 7 of C (505) [minus 0xFF for CB prefix]
	0x1F9: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.C, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xna **********************/
	// RRC D - Rotate D right. Old bit 0 to Carry flag (266) [minus 0xFF for CB prefix]
	0x10a: func(mb *Motherboard, value uint16) OpCycles {

		d := uint16(mb.Cpu.Registers.D)
		v := uint16((d >> 1) | ((d & 1) << 7) | ((d & 1) << 8))

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.D = uint8(v & 0xFF)

		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RR D - Rotate D right through Carry flag (282) [minus 0xFF for CB prefix]
	0x11a: func(mb *Motherboard, value uint16) OpCycles {

		b := uint16(mb.Cpu.Registers.D)

		var oldCarry uint16 = 0
		if mb.Cpu.IsFlagCSet() {
			oldCarry = 1
		}
		v := uint16(b>>1) + uint16((oldCarry << 7)) + uint16((b&1)<<8)

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.D = uint8(v & 0xFF)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRA D - Shift D right into Carry. MSB doesn't change (298) [minus 0xFF for CB prefix]
	0x12a: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.D
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register D to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(mb.Cpu.Registers.D, 7) {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.D = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRL D - Shift D right into Carry. MSB set to 0 (314) [minus 0xFF for CB prefix]
	0x13a: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.D
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register D to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.D = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 1, D - Test bit 1 of D (330) [minus 0xFF for CB prefix]
	0x14a: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 3, D - Test bit 3 of D (346) [minus 0xFF for CB prefix]
	0x15a: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 5, D - Test bit 5 of D (362) [minus 0xFF for CB prefix]
	0x16a: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 7, D - Test bit 7 of D (378) [minus 0xFF for CB prefix]
	0x17a: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.D, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 1, D - Reset bit 2 of D (394) [minus 0xFF for CB prefix]
	0x18a: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 3, D - Reset bit 3 of D (410) [minus 0xFF for CB prefix]
	0x19a: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 5, D - Reset bit 5 of D (426) [minus 0xFF for CB prefix]
	0x1AA: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 7, D - Reset bit 7 of D (442) [minus 0xFF for CB prefix]
	0x1BA: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.D, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 1, D - Set bit 1 of D (458) [minus 0xFF for CB prefix]
	0x1CA: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 3, D - Set bit 3 of D (474) [minus 0xFF for CB prefix]
	0x1DA: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 5, D - Set bit 5 of D (490) [minus 0xFF for CB prefix]
	0x1EA: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 7, D - Set bit 7 of D (506) [minus 0xFF for CB prefix]
	0x1FA: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.D, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xnb **********************/
	// RRC E - Rotate E right. Old bit 0 to Carry flag (267) [minus 0xFF for CB prefix]
	0x10b: func(mb *Motherboard, value uint16) OpCycles {

		d := uint16(mb.Cpu.Registers.E)
		v := uint16((d >> 1) | ((d & 1) << 7) | ((d & 1) << 8))

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.E = uint8(v & 0xFF)

		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RR E - Rotate E right through Carry flag (283) [minus 0xFF for CB prefix]
	0x11b: func(mb *Motherboard, value uint16) OpCycles {

		b := uint16(mb.Cpu.Registers.E)

		var oldCarry uint16 = 0
		if mb.Cpu.IsFlagCSet() {
			oldCarry = 1
		}
		v := uint16(b>>1) + uint16((oldCarry << 7)) + uint16((b&1)<<8)

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.E = uint8(v & 0xFF)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRA E - Shift E right into Carry. MSB doesn't change (299) [minus 0xFF for CB prefix]
	0x12b: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.E
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register E to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(mb.Cpu.Registers.E, 7) {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.E = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRL E - Shift E right into Carry. MSB set to 0 (315) [minus 0xFF for CB prefix]
	0x13b: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.E
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}
		// shift register E to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.E = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 1, E - Test bit 1 of E (331) [minus 0xFF for CB prefix]
	0x14b: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.E, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 3, E - Test bit 3 of E (347) [minus 0xFF for CB prefix]
	0x15b: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.E, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 5, E - Test bit 5 of E (363) [minus 0xFF for CB prefix]
	0x16b: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.E, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 7, E - Test bit 7 of E (379) [minus 0xFF for CB prefix]
	0x17b: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.E, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 1, E - Reset bit 3 of E (395) [minus 0xFF for CB prefix]
	0x18b: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 3, E - Reset bit 3 of E (411) [minus 0xFF for CB prefix]
	0x19b: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 5, E - Reset bit 5 of E (427) [minus 0xFF for CB prefix]
	0x1AB: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 7, E - Reset bit 7 of E (443) [minus 0xFF for CB prefix]
	0x1BB: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.E, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 1, E - Set bit 1 of E (459) [minus 0xFF for CB prefix]
	0x1CB: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 3, E - Set bit 3 of E (475) [minus 0xFF for CB prefix]
	0x1DB: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 5, E - Set bit 5 of E (491) [minus 0xFF for CB prefix]
	0x1EB: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 7, E - Set bit 7 of E (507) [minus 0xFF for CB prefix]
	0x1FB: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.E, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xnc **********************/
	// RRC H - Rotate H right. Old bit 0 to Carry flag (268) [minus 0xFF for CB prefix]
	0x10c: func(mb *Motherboard, value uint16) OpCycles {
		d := uint16(mb.Cpu.Registers.H)
		v := uint16((d >> 1) | ((d & 1) << 7) | ((d & 1) << 8))

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.H = uint8(v & 0xFF)

		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RR H - Rotate H right through Carry flag (284) [minus 0xFF for CB prefix]
	0x11c: func(mb *Motherboard, value uint16) OpCycles {

		b := uint16(mb.Cpu.Registers.H)

		var oldCarry uint16 = 0
		if mb.Cpu.IsFlagCSet() {
			oldCarry = 1
		}
		v := uint16(b>>1) + uint16((oldCarry << 7)) + uint16((b&1)<<8)

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.H = uint8(v & 0xFF)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRA H - Shift H right into Carry. MSB doesn't change (300) [minus 0xFF for CB prefix]
	0x12c: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.H
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register H to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(mb.Cpu.Registers.H, 7) {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.H = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRL H - Shift H right into Carry. MSB set to 0 (316) [minus 0xFF for CB prefix]
	0x13c: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.H
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}
		// shift register H to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.H = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 1, H - Test bit 1 of H (332) [minus 0xFF for CB prefix]
	0x14c: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.H, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 3, H - Test bit 3 of H (348) [minus 0xFF for CB prefix]
	0x15c: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.H, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 5, H - Test bit 5 of H (364) [minus 0xFF for CB prefix]
	0x16c: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.H, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 7, H - Test bit 7 of H (380) [minus 0xFF for CB prefix]
	0x17c: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.H, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 1, H - Reset bit 1 of H (396) [minus 0xFF for CB prefix]
	0x18c: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 3, H - Reset bit 3 of H (412) [minus 0xFF for CB prefix]
	0x19c: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 5, H - Reset bit 5 of H (428) [minus 0xFF for CB prefix]
	0x1AC: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 7, H - Reset bit 7 of H (444) [minus 0xFF for CB prefix]
	0x1BC: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.H, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 1, H - Set bit 1 of H (460) [minus 0xFF for CB prefix]
	0x1CC: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 3, H - Set bit 3 of H (476) [minus 0xFF for CB prefix]
	0x1DC: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 5, H - Set bit 5 of H (492) [minus 0xFF for CB prefix]
	0x1EC: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 7, H - Set bit 7 of H (508) [minus 0xFF for CB prefix]
	0x1FC: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.H, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xnd **********************/
	// RRC L - Rotate L right. Old bit 0 to Carry flag (269) [minus 0xFF for CB prefix]
	0x10d: func(mb *Motherboard, value uint16) OpCycles {

		d := uint16(mb.Cpu.Registers.L)
		v := uint16((d >> 1) | ((d & 1) << 7) | ((d & 1) << 8))

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.L = uint8(v & 0xFF)

		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RR L - Rotate L right through Carry flag (285) [minus 0xFF for CB prefix]
	0x11d: func(mb *Motherboard, value uint16) OpCycles {
		b := uint16(mb.Cpu.Registers.L)

		var oldCarry uint16 = 0
		if mb.Cpu.IsFlagCSet() {
			oldCarry = 1
		}
		v := uint16(b>>1) + uint16((oldCarry << 7)) + uint16((b&1)<<8)

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.L = uint8(v & 0xFF)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRA L - Shift L right into Carry. MSB doesn't change (301) [minus 0xFF for CB prefix]
	0x12d: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.L
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register L to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(mb.Cpu.Registers.L, 7) {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.L = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRL L - Shift L right into Carry. MSB set to 0 (317) [minus 0xFF for CB prefix]
	0x13d: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.L
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}
		// shift register L to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.L = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 1, L - Test bit 1 of L (333) [minus 0xFF for CB prefix]
	0x14d: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.L, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 3, L - Test bit 3 of L (349) [minus 0xFF for CB prefix]
	0x15d: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.L, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 5, L - Test bit 5 of L (365) [minus 0xFF for CB prefix]
	0x16d: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.L, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 7, L - Test bit 7 of L (381) [minus 0xFF for CB prefix]
	0x17d: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.L, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 1, L - Reset bit 5 of L (397) [minus 0xFF for CB prefix]
	0x18d: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 3, L - Reset bit 3 of L (413) [minus 0xFF for CB prefix]
	0x19d: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 5, L - Reset bit 5 of L (429) [minus 0xFF for CB prefix]
	0x1AD: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 7, L - Reset bit 7 of L (445) [minus 0xFF for CB prefix]
	0x1BD: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.L, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 1, L - Set bit 1 of L (461) [minus 0xFF for CB prefix]
	0x1CD: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 3, L - Set bit 3 of L (477) [minus 0xFF for CB prefix]
	0x1DD: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 5, L - Set bit 5 of L (493) [minus 0xFF for CB prefix]
	0x1ED: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 7, L - Set bit 7 of L (509) [minus 0xFF for CB prefix]
	0x1FD: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.L, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	/****************************** 0xne **********************/
	// RRC (HL) - Rotate value pointed by HL right. Old bit 0 to Carry flag (270) [minus 0xFF for CB prefix]
	0x10e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
			b = (b >> 1) + 0x80
		} else {
			mb.Cpu.ResetFlagC()
			b = (b >> 1)
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RR (HL) - Rotate value pointed by HL right through Carry flag (286) [minus 0xFF for CB prefix]
	0x11e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		oldCarry := mb.Cpu.IsFlagCSet()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift value pointed by HL to the right by one bit
		b = (b >> 1) & 0xff
		if oldCarry {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SRA (HL) - Shift value pointed by HL right into Carry. MSB doesn't change (302) [minus 0xFF for CB prefix]
	0x12e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetAllFlags()

		if b&0x01 != 0 {
			mb.Cpu.SetFlagC()
		}

		if b&0x80 != 0 {
			b >>= 1
			b |= 0x80
		} else {
			b >>= 1
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SRL (HL) - Shift value pointed by HL right into Carry. MSB set to 0 (318) [minus 0xFF for CB prefix]
	0x13e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if internal.IsBitSet(b, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}
		// shift value pointed by HL to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// BIT 1, (HL) - Test bit 1 of value pointed by HL (334) [minus 0xFF for CB prefix]
	0x14e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(b, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// BIT 3, (HL) - Test bit 3 of value pointed by HL (350) [minus 0xFF for CB prefix]
	0x15e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(b, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// BIT 5, (HL) - Test bit 5 of value pointed by HL (366) [minus 0xFF for CB prefix]
	0x16e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(b, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// BIT 7, (HL) - Test bit 7 of value pointed by HL (382) [minus 0xFF for CB prefix]
	0x17e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(b, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 12
	},

	// RES 1, (HL) - Reset bit 1 of value pointed by HL (398) [minus 0xFF for CB prefix]
	0x18e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 1)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RES 3, (HL) - Reset bit 3 of value pointed by HL (414) [minus 0xFF for CB prefix]
	0x19e: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 3)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RES 5, (HL) - Reset bit 5 of value pointed by HL (430) [minus 0xFF for CB prefix]
	0x1AE: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 5)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// RES 7, (HL) - Reset bit 7 of value pointed by HL (446) [minus 0xFF for CB prefix]
	0x1BE: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.ResetBit(&b, 7)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 1, (HL) - Set bit 1 of value pointed by HL (462) [minus 0xFF for CB prefix]
	0x1CE: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 1)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 3, (HL) - Set bit 3 of value pointed by HL (478) [minus 0xFF for CB prefix]
	0x1DE: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 3)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 5, (HL) - Set bit 5 of value pointed by HL (494) [minus 0xFF for CB prefix]
	0x1EE: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 5)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	// SET 7, (HL) - Set bit 7 of value pointed by HL (510) [minus 0xFF for CB prefix]
	0x1FE: func(mb *Motherboard, value uint16) OpCycles {

		hl := mb.Cpu.HL()
		b := mb.GetItem(&hl)
		internal.SetBit(&b, 7)
		b16 := uint16(b)
		mb.SetItem(&hl, &b16)
		mb.Cpu.Registers.PC += 2
		return 16
	},

	/****************************** 0xnf **********************/
	// RRC A - Rotate A right. Old bit 0 to Carry flag (271) [minus 0xFF for CB prefix]
	0x10f: func(mb *Motherboard, value uint16) OpCycles {

		d := uint16(mb.Cpu.Registers.A)
		v := uint16((d >> 1) | ((d & 1) << 7) | ((d & 1) << 8))

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.A = uint8(v & 0xFF)

		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RR A - Rotate A right through Carry flag (287) [minus 0xFF for CB prefix]
	0x11f: func(mb *Motherboard, value uint16) OpCycles {
		b := uint16(mb.Cpu.Registers.A)

		var oldCarry uint16 = 0
		if mb.Cpu.IsFlagCSet() {
			oldCarry = 1
		}
		v := uint16(b>>1) + uint16((oldCarry << 7)) + uint16((b&1)<<8)

		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if v&0xFF == 0 {
			mb.Cpu.SetFlagZ()
		}

		if v > 0xFF {
			mb.Cpu.SetFlagC()
		}

		mb.Cpu.Registers.A = uint8(v & 0xFF)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRA A - Shift A right into Carry. MSB doesn't change (303) [minus 0xFF for CB prefix]
	0x12f: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()

		if internal.IsBitSet(mb.Cpu.Registers.A, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}

		// shift register A to the right by one bit
		b = (b >> 1) & 0xff
		if internal.IsBitSet(mb.Cpu.Registers.A, 7) {
			b |= 0x80
		}

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.A = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SRL A - Shift A right into Carry. MSB set to 0 (319) [minus 0xFF for CB prefix]
	0x13f: func(mb *Motherboard, value uint16) OpCycles {

		b := mb.Cpu.Registers.A
		mb.Cpu.ResetFlagZ()
		mb.Cpu.ResetFlagN()
		mb.Cpu.ResetFlagH()
		mb.Cpu.ResetFlagC()

		if internal.IsBitSet(mb.Cpu.Registers.A, 0) {
			mb.Cpu.SetFlagC()
		} else {
			mb.Cpu.ResetFlagC()
		}
		// shift register A to the right by one bit
		b = (b >> 1) & 0xff

		if b == 0 {
			mb.Cpu.SetFlagZ()
		}

		mb.Cpu.Registers.A = b
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 1, A - Test bit 1 of A (335) [minus 0xFF for CB prefix]
	0x14f: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()

		mb.Cpu.SetFlagZ()
		if internal.IsBitSet(mb.Cpu.Registers.A, 1) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 3, A - Test bit 3 of A (351) [minus 0xFF for CB prefix]
	0x15f: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.A, 3) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 5, A - Test bit 5 of A (367) [minus 0xFF for CB prefix]
	0x16f: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.A, 5) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// BIT 7, A - Test bit 7 of A (383) [minus 0xFF for CB prefix]
	0x17f: func(mb *Motherboard, value uint16) OpCycles {

		mb.Cpu.ResetFlagN()
		mb.Cpu.SetFlagH()
		mb.Cpu.SetFlagZ()

		if internal.IsBitSet(mb.Cpu.Registers.A, 7) {
			mb.Cpu.ResetFlagZ()
		}
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 1, A - Reset bit 7 of A (399) [minus 0xFF for CB prefix]
	0x18f: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 3, A - Reset bit 3 of A (415) [minus 0xFF for CB prefix]
	0x19f: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 5, A - Reset bit 5 of A (431) [minus 0xFF for CB prefix]
	0x1AF: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// RES 7, A - Reset bit 7 of A (447) [minus 0xFF for CB prefix]
	0x1BF: func(mb *Motherboard, value uint16) OpCycles {

		internal.ResetBit(&mb.Cpu.Registers.A, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 1, A - Set bit 1 of A (463) [minus 0xFF for CB prefix]
	0x1CF: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 1)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 3, A - Set bit 3 of A (479) [minus 0xFF for CB prefix]
	0x1DF: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 3)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 5, A - Set bit 5 of A (495) [minus 0xFF for CB prefix]
	0x1EF: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 5)
		mb.Cpu.Registers.PC += 2
		return 8
	},

	// SET 7, A - Set bit 7 of A (511) [minus 0xFF for CB prefix]
	0x1FF: func(mb *Motherboard, value uint16) OpCycles {

		internal.SetBit(&mb.Cpu.Registers.A, 7)
		mb.Cpu.Registers.PC += 2
		return 8
	},
}
