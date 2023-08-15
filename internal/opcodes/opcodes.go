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
		c.Registers.B = c.Registers.B
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
		c.Registers.D = c.Registers.D
		c.Registers.PC += 1
		return 4
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
		c.Registers.C = c.Registers.C
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
		c.Registers.E = c.Registers.E
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
}
