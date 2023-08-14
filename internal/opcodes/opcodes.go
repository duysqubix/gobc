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
		a := uint16(c.Registers.A)
		c.ResetFlagN()

		if c.IsFlagHSet() || (!c.IsFlagNSet() && (a&0x0F) > 0x09) {
			a += 0x06
		}

		if c.IsFlagCSet() || (!c.IsFlagNSet() && a > 0x9F) {
			a += 0x60
		}

		if (a & 0x100) == 0x100 {
			c.SetFlagC()
		}

		a &= 0xFF

		if a == 0 {
			c.SetFlagZ()
		} else {
			c.ResetFlagZ()
		}

		c.Registers.A = uint8(a)
		c.Registers.PC += 1
		return 4
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
}
