package cpu

import (
	"fmt"
	"math/rand"

	"github.com/duysqubix/gobc/internal"
)

const (
	FLAGC uint8 = 0x04 // Math operation raised carry
	FLAGH uint8 = 0x05 // Math operation raised half carry
	FLAGN uint8 = 0x06 // Math operation was a subtraction
	FLAGZ uint8 = 0x07 // Math operation result was zero

	INTR_VBLANK    uint8 = 0x1  // VBlank interrupt
	INTR_LCDSTAT   uint8 = 0x2  // LCD status interrupt
	INTR_TIMER     uint8 = 0x4  // Timer interrupt
	INTR_SERIAL    uint8 = 0x8  // Serial interrupt
	INTR_HIGHTOLOW uint8 = 0x10 // Joypad interrupt

	CLOCK_FREQ_GB  uint32 = 4194304 // 4.194304 MHz
	CLOCK_FREQ_CGB uint32 = 8388608 // 8.388608 MHz
)

// Registers is a struct that represents the CPU registers
type Registers struct {
	A  uint8  // Accumulator
	B  uint8  // General purpose
	C  uint8  // General purpose
	D  uint8  // General purpose
	E  uint8  // General purpose
	F  uint8  // Flags
	H  uint8  // General purpose
	L  uint8  // General purpose
	SP uint16 // Stack pointer
	PC uint16 // Program counter
}

type Cpu struct {
	Registers        *Registers
	Halted           bool
	IntrMasterEnable bool
}

func NewCpu() *Cpu {
	return &Cpu{
		Registers: &Registers{
			A:  0,
			B:  0,
			C:  0,
			D:  0,
			E:  0,
			F:  0,
			H:  0,
			L:  0,
			SP: 0,
			PC: 0,
		},
		Halted:           false,
		IntrMasterEnable: false,
	}

}

func (c *Cpu) RandomizeRegisters(seed int64) {
	r := rand.New(rand.NewSource(seed))

	c.Registers.A = uint8(r.Intn(0xffff))
	c.Registers.B = uint8(r.Intn(0xffff))
	c.Registers.C = uint8(r.Intn(0xffff))
	c.Registers.D = uint8(r.Intn(0xffff))
	c.Registers.E = uint8(r.Intn(0xffff))
	c.Registers.F = uint8((r.Intn(0x0f) << 4))
	c.Registers.H = uint8(r.Intn(0xffff))
	c.Registers.L = uint8(r.Intn(0xffff))
	c.Registers.SP = uint16(r.Intn(0xffffff))
	c.Registers.PC = uint16(r.Intn(0xffffff))

}

func (c *Cpu) ClearAllFlags() {c.Registers.F = 0}

func (c *Cpu) IsFlagZSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGZ)) }
func (c *Cpu) IsFlagNSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGN)) }
func (c *Cpu) IsFlagHSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGH)) }
func (c *Cpu) IsFlagCSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGC)) }

func (c *Cpu) ToggleFlagC() { internal.ToggleBit(&c.Registers.F, uint8(FLAGC)) }
func (c *Cpu) ToggleFlagH() { internal.ToggleBit(&c.Registers.F, uint8(FLAGH)) }
func (c *Cpu) ToggleFlagN() { internal.ToggleBit(&c.Registers.F, uint8(FLAGN)) }
func (c *Cpu) ToggleFlagZ() { internal.ToggleBit(&c.Registers.F, uint8(FLAGZ)) }

func (c *Cpu) SetFlagZ() { internal.SetBit(&c.Registers.F, uint8(FLAGZ)) }
func (c *Cpu) SetFlagN() { internal.SetBit(&c.Registers.F, uint8(FLAGN)) }
func (c *Cpu) SetFlagH() { internal.SetBit(&c.Registers.F, uint8(FLAGH)) }
func (c *Cpu) SetFlagC() { internal.SetBit(&c.Registers.F, uint8(FLAGC)) }

func (c *Cpu) ResetFlagZ() { internal.ResetBit(&c.Registers.F, uint8(FLAGZ)) }
func (c *Cpu) ResetFlagN() { internal.ResetBit(&c.Registers.F, uint8(FLAGN)) }
func (c *Cpu) ResetFlagH() { internal.ResetBit(&c.Registers.F, uint8(FLAGH)) }
func (c *Cpu) ResetFlagC() { internal.ResetBit(&c.Registers.F, uint8(FLAGC)) }

func (c *Cpu) SetBC(value uint16) {
	c.Registers.B = uint8(value >> 8)
	c.Registers.C = uint8(value & 0xFF)
}

func (c *Cpu) SetDE(value uint16) {
	c.Registers.D = uint8(value >> 8)
	c.Registers.E = uint8(value & 0xFF)
}

func (c *Cpu) SetHL(value uint16) {
	c.Registers.H = uint8(value >> 8)
	c.Registers.L = uint8(value & 0xFF)
}

func (c *Cpu) SetAF(value uint16) {
	c.Registers.A = uint8(value >> 8)
	c.Registers.F = uint8(value & 0xFF)
}

func (c *Cpu) BC() uint16 {
	return (uint16)(c.Registers.B)<<8 | (uint16)(c.Registers.C)
}

func (c *Cpu) DE() uint16 {
	return (uint16)(c.Registers.D)<<8 | (uint16)(c.Registers.E)
}

func (c *Cpu) HL() uint16 {
	return (uint16)(c.Registers.H)<<8 | (uint16)(c.Registers.L)
}

func (cpu *Cpu) Dump(header string) {
	reg := cpu.Registers
	fmt.Printf("GOBC -- %s\n", header)
	fmt.Printf("A: %X(%d) F: %X(%d) <%04b|ZNHC>\n", reg.A, reg.A, reg.F, reg.F, (reg.F >> 4))
	fmt.Printf("B: %X(%d) C: %X(%d)\n", reg.B, reg.B, reg.C, reg.C)
	fmt.Printf("D: %X(%d)  E: %X(%d)\n", reg.D, reg.D, reg.E, reg.E)
	fmt.Printf("HL: %X(%d) SP: %X(%d) PC: %X(%d)\n", uint16(reg.H)<<8|uint16(reg.L), uint16(reg.H)<<8|uint16(reg.L), reg.SP, reg.SP, reg.PC, reg.PC)
	fmt.Println("*=============================================*")
}

func (c *Cpu) CpSetFlags(a uint8, b uint8) {

	// Check for carry using 16bit arithmetic
	al := uint16(a)
	bl := uint16(b)

	r := al - bl

	c.ResetFlagZ()
	if (r & 0xff) == 0 {
		c.SetFlagZ()
	}

	c.SetFlagN()

	c.ResetFlagH()
	if (al^bl^r)&0x10 != 0 {
		c.SetFlagH()
	}

	c.ResetFlagC()
	if r&0x100 != 0 {
		c.SetFlagC()
	}
}

func (c *Cpu) AndSetFlags(a uint8, b uint8) uint8 {
	r := a & b
	c.ResetFlagZ()
	if r == 0 {
		c.SetFlagZ()
	}
	c.ResetFlagN()
	c.SetFlagH()
	c.ResetFlagC()
	return r
}

func (c *Cpu) OrSetFlags(a uint8, b uint8) uint8 {
	r := a | b
	c.ResetFlagZ()
	if r == 0 {
		c.SetFlagZ()
	}
	c.ResetFlagN()
	c.ResetFlagH()
	c.ResetFlagC()
	return r
}

func (c *Cpu) XorSetFlags(a uint8, b uint8) uint8 {
	r := a ^ b
	c.ResetFlagZ()
	if r == 0 {
		c.SetFlagZ()
	}
	c.ResetFlagN()
	c.ResetFlagH()
	c.ResetFlagC()
	return r
}

func (c *Cpu) SubSetFlags8(a uint8, b uint8) uint8 {
	// Check for carry using 16bit arithmetic
	al := uint16(a)
	bl := uint16(b)

	r := al - bl

	c.ResetFlagZ()
	if (r & 0xff) == 0 {
		c.SetFlagZ()
	}

	c.SetFlagN()

	c.ResetFlagH()
	if (al^bl^r)&0x10 != 0 {
		c.SetFlagH()
	}

	c.ResetFlagC()
	if r&0x100 != 0 {
		c.SetFlagC()
	}

	return uint8(r)
}

func (c *Cpu) SbcSetFlags8(a uint8, b uint8) uint8 {
	// Check for carry using 16bit arithmetic
	al := uint16(a)
	bl := uint16(b)

	var fc uint16 = 0
	if c.IsFlagCSet() {
		fc = 1
	}
	r := al - bl - fc

	c.ResetFlagZ()
	if (r & 0xff) == 0 {
		c.SetFlagZ()
	}

	c.SetFlagN()

	c.ResetFlagH()
	if (al^bl^r)&0x10 != 0 {
		c.SetFlagH()
	}

	c.ResetFlagC()
	if r&0x100 != 0 {
		c.SetFlagC()
	}

	return uint8(r)
}

func (c *Cpu) AddSetFlags16(a uint16, b uint16) uint32 {
	// widen to 32 bits to get carry
	a32 := uint32(a)
	b32 := uint32(b)

	var r uint32 = a32 + b32
	c.ResetFlagN()

	c.ResetFlagC()
	if (r & 0x10000) != 0 {
		c.SetFlagC()
	}

	c.ResetFlagH()
	if (a32^b32^r)&0x1000 != 0 {
		c.SetFlagH()
	}
	// fmt.Printf("AddSetFlags: %X + %X = %X\n", a, b, r)
	return r
}

func (c *Cpu) AddSetFlags8(a uint8, b uint8) uint8 {
	// Check for carry using 16bit arithmetic
	al := uint16(a)
	bl := uint16(b)

	r := al + bl

	c.ResetFlagZ()
	if (r & 0xff) == 0 {
		c.SetFlagZ()
	}

	c.ResetFlagN()

	c.ResetFlagH()
	if (al^bl^r)&0x10 != 0 {
		c.SetFlagH()
	}

	c.ResetFlagC()
	if r&0x100 != 0 {
		c.SetFlagC()
	}

	return uint8(r)
}

func (c *Cpu) AdcSetFlags8(a uint8, b uint8) uint8 {
	// Check for carry using 16bit arithmetic
	al := uint16(a)
	bl := uint16(b)

	var fc uint16 = 0
	if c.IsFlagCSet() {
		fc = 1
	}
	r := al + bl + fc

	c.ResetFlagZ()
	if (r & 0xff) == 0 {
		c.SetFlagZ()
	}

	c.ResetFlagN()

	c.ResetFlagH()
	if (al^bl^r)&0x10 != 0 {
		c.SetFlagH()
	}

	c.ResetFlagC()
	if r&0x100 != 0 {
		c.SetFlagC()
	}

	return uint8(r)
}

// func (c *Cpu) SubSetFlags8(a uint8, b uint8) uint8 {
// 	// Check for carry using 16bit arithmetic
// 	al := uint16(a)
// 	bl := uint16(b)

// 	r := al - bl

// 	c.ResetFlagZ()
// 	if (r & 0xff) == 0 {
// 		c.SetFlagZ()
// 	}

// 	c.SetFlagN()

// 	c.ResetFlagH()
// 	if (al^bl^r)&0x10 != 0 {
// 		c.SetFlagH()
// 	}

// 	c.ResetFlagC()
// 	if r&0x100 != 0 {
// 		c.SetFlagC()
// 	}

// 	return uint8(r)
// }

func (c *Cpu) Inc(v uint8) uint8 {
	r := (v + 1) & 0xff

	c.ResetFlagZ()
	if r == 0 {
		c.SetFlagZ()
	}

	c.ResetFlagN()

	c.ResetFlagH()
	if (v & 0xf) == 0xf {
		c.SetFlagH()
	}

	return r
}

func (c *Cpu) Dec(v uint8) uint8 {
	r := (v - 1) & 0xff

	c.ResetFlagZ()
	if r == 0 {
		c.SetFlagZ()
	}

	c.SetFlagN()

	c.ResetFlagH()
	if (v & 0xf) == 0 {
		c.SetFlagH()
	}

	return r
}
