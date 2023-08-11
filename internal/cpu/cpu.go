package cpu

type flag uint8

const (
	FLAGC flag = 0x04 // Math operation raised carry
	FLAGH flag = 0x05 // Math operation raised half carry
	FLAGN flag = 0x06 // Math operation was a subtraction
	FLAGZ flag = 0x07 // Math operation result was zero

	INTR_VBLANK    flag = 0x1  // VBlank interrupt
	INTR_LCDSTAT   flag = 0x2  // LCD status interrupt
	INTR_TIMER     flag = 0x4  // Timer interrupt
	INTR_SERIAL    flag = 0x8  // Serial interrupt
	INTR_HIGHTOLOW flag = 0x10 // Joypad interrupt

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
	Registers *Registers
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
	}

}

func (c *Cpu) SetBC(value uint16) {
	c.Registers.B = uint8(value >> 8)
	c.Registers.C = uint8(value & 0xFF)
}

func (c *Cpu) SetDE(value uint16) {
	c.Registers.D = uint8(value >> 8)
	c.Registers.E = uint8(value & 0xFF)
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
