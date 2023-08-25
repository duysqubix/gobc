package motherboard

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"

	"github.com/duysqubix/gobc/internal"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
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

type CPU struct {
	Registers  *Registers   // CPU registers
	Halted     bool         // CPU halted
	Interrupts *Interrupts  // Interrupts
	Mb         *Motherboard // Motherboard
	IsStuck    bool         // CPU is stuck
	Stopped    bool         // CPU is stopped
}

func NewCpu(mb *Motherboard) *CPU {
	return &CPU{
		Registers: &Registers{
			A:  0x1,
			B:  0,
			C:  0,
			D:  0,
			E:  0x8F,
			F:  0xD0,
			H:  0,
			L:  0x87,
			SP: 0xFFFE,
			PC: 0,
		},
		Halted: false,
		Interrupts: &Interrupts{
			Master_Enable: false,
			IE:            0,
			IF:            0,
			Queued:        false,
		},
		Mb: mb,
	}

}

func (c *CPU) Tick() OpCycles {

	switch {
	case c.CheckForInterrupts():

		c.Halted = false
		return 0

	case c.Halted && c.Interrupts.Queued:
		// GBCPUman.pdf page 20
		// WARNING: The instruction immediately following the HALT instruction is "skipped" when interrupts are
		// disabled (DI) on the GB,GBP, and SGB.
		c.Halted = false
		c.Registers.PC += 1

	case c.Halted:
		return 4
	default:
	}

	old_pc := c.Registers.PC
	old_sp := c.Registers.SP
	cycles := c.ExecuteInstruction()

	if !c.Halted && (old_pc == c.Registers.PC) && (old_sp == c.Registers.SP) && !c.IsStuck {
		logger.Errorf("CPU is stuck at PC: %#x SP: %#x", c.Registers.PC, c.Registers.SP)
		c.DumpState()
		c.IsStuck = true
	}

	c.Interrupts.Queued = false
	return cycles
}

func (c *CPU) ExecuteInstruction() OpCycles {
	var value uint16

	opcode := OpCode(c.Mb.GetItem(&c.Registers.PC))
	// fmt.Printf("Pre-Execution :Opcode: %s [%#x] | PC: %#x | SP: %#x\n", internal.OPCODE_NAMES[opcode], opcode, c.Registers.PC, c.Registers.SP)
	if opcode.CBPrefix() {
		pcn := c.Registers.PC + 1
		opcode = OpCode(c.Mb.GetItem(&pcn))
		opcode = opcode.Shift()

	}
	pc := c.Registers.PC
	opcode_len := internal.OPCODE_LENGTHS[opcode]
	switch opcode_len {

	// 8 bit immediate
	case 2:
		pc += 1
		value = uint16(c.Mb.GetItem(&pc))

	// 16 bit immediate
	case 3:
		pc += 1
		b := uint16(c.Mb.GetItem(&pc))
		pc += 1
		a := uint16(c.Mb.GetItem(&pc))
		value = (a << 8) | b

	default:
		value = 0
	}
	// fmt.Printf("Post-Execution :Opcode: %s [%#x] | PC: %#x | SP: %#x | Value: %#x\n", internal.OPCODE_NAMES[opcode], opcode, c.Registers.PC, c.Registers.SP, value)

	if c.Mb.Breakpoints.Enabled {
		if internal.IsInUint16Array(pc, c.Mb.Breakpoints.Addrs) {
			reader := bufio.NewReader(os.Stdin)

			old_level := logger.Level
			logger.SetLevel(log.DebugLevel)
			c.DumpState()
			logger.Debugf("Executing %s [%#x] with value $%X | PC: $%X", internal.OPCODE_NAMES[opcode], opcode, value, pc)
			logger.SetLevel(old_level)
			reader.ReadString('\n')
		}
	}
	return OPCODES[opcode](c.Mb, value)
}

func (c *CPU) RandomizeRegisters(seed int64) {
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

func (c *CPU) ClearAllFlags() { c.Registers.F = 0 }

func (c *CPU) IsFlagZSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGZ)) }
func (c *CPU) IsFlagNSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGN)) }
func (c *CPU) IsFlagHSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGH)) }
func (c *CPU) IsFlagCSet() bool { return internal.IsBitSet(c.Registers.F, uint8(FLAGC)) }

func (c *CPU) ToggleFlagC() { internal.ToggleBit(&c.Registers.F, uint8(FLAGC)) }
func (c *CPU) ToggleFlagH() { internal.ToggleBit(&c.Registers.F, uint8(FLAGH)) }
func (c *CPU) ToggleFlagN() { internal.ToggleBit(&c.Registers.F, uint8(FLAGN)) }
func (c *CPU) ToggleFlagZ() { internal.ToggleBit(&c.Registers.F, uint8(FLAGZ)) }

func (c *CPU) SetFlagZ() { internal.SetBit(&c.Registers.F, uint8(FLAGZ)) }
func (c *CPU) SetFlagN() { internal.SetBit(&c.Registers.F, uint8(FLAGN)) }
func (c *CPU) SetFlagH() { internal.SetBit(&c.Registers.F, uint8(FLAGH)) }
func (c *CPU) SetFlagC() { internal.SetBit(&c.Registers.F, uint8(FLAGC)) }

func (c *CPU) ResetFlagZ() { internal.ResetBit(&c.Registers.F, uint8(FLAGZ)) }
func (c *CPU) ResetFlagN() { internal.ResetBit(&c.Registers.F, uint8(FLAGN)) }
func (c *CPU) ResetFlagH() { internal.ResetBit(&c.Registers.F, uint8(FLAGH)) }
func (c *CPU) ResetFlagC() { internal.ResetBit(&c.Registers.F, uint8(FLAGC)) }

func (c *CPU) SetBC(value uint16) {
	c.Registers.B = uint8(value >> 8)
	c.Registers.C = uint8(value & 0xFF)
}

func (c *CPU) SetDE(value uint16) {
	c.Registers.D = uint8(value >> 8)
	c.Registers.E = uint8(value & 0xFF)
}

func (c *CPU) SetHL(value uint16) {
	c.Registers.H = uint8(value >> 8)
	c.Registers.L = uint8(value & 0xFF)
}

func (c *CPU) SetAF(value uint16) {
	c.Registers.A = uint8(value >> 8)
	c.Registers.F = uint8(value & 0xFF)
}

func (c *CPU) BC() uint16 {
	return (uint16)(c.Registers.B)<<8 | (uint16)(c.Registers.C)
}

func (c *CPU) DE() uint16 {
	return (uint16)(c.Registers.D)<<8 | (uint16)(c.Registers.E)
}

func (c *CPU) HL() uint16 {
	return (uint16)(c.Registers.H)<<8 | (uint16)(c.Registers.L)
}

func (cpu *CPU) Dump(header string) {
	reg := cpu.Registers
	fmt.Printf("GOBC -- %s\n", header)
	fmt.Printf("A: %X(%d) F: %X(%d) <%04b|ZNHC>\n", reg.A, reg.A, reg.F, reg.F, (reg.F >> 4))
	fmt.Printf("B: %X(%d) C: %X(%d)\n", reg.B, reg.B, reg.C, reg.C)
	fmt.Printf("D: %X(%d)  E: %X(%d)\n", reg.D, reg.D, reg.E, reg.E)
	fmt.Printf("HL: %X(%d) SP: %X(%d) PC: %X(%d)\n", uint16(reg.H)<<8|uint16(reg.L), uint16(reg.H)<<8|uint16(reg.L), reg.SP, reg.SP, reg.PC, reg.PC)
	fmt.Println("*=============================================*")
}

func (cpu *CPU) DumpState() {
	pc := cpu.Registers.PC
	pc2 := pc - 1
	pc3 := pc - 2
	opdata := []OpCode{
		OpCode(cpu.Mb.GetItem(&pc)),
		OpCode(cpu.Mb.GetItem(&pc2)),
		OpCode(cpu.Mb.GetItem(&pc3)),
	}

	opdata1 := opdata[0]
	var op_code_str string = fmt.Sprintf("OpCode: $%X", opdata1)
	if opdata1.CBPrefix() {
		op_code_str += fmt.Sprintf(" $%X", opdata[1])
	} else {
		op_code_str += fmt.Sprintf(" $%X $%X", opdata[1], opdata[2])
	}

	var report [][]string = [][]string{
		{"OpCode", op_code_str},
		{"A", fmt.Sprintf("$%X", cpu.Registers.A)},
		{"F", fmt.Sprintf("$%X", cpu.Registers.F)},
		{"B", fmt.Sprintf("$%X", cpu.Registers.B)},
		{"C", fmt.Sprintf("$%X", cpu.Registers.C)},
		{"D", fmt.Sprintf("$%X", cpu.Registers.D)},
		{"E", fmt.Sprintf("$%X", cpu.Registers.E)},
		{"H", fmt.Sprintf("$%X", cpu.Registers.H)},
		{"L", fmt.Sprintf("$%X", cpu.Registers.L)},
		{"SP", fmt.Sprintf("$%X", cpu.Registers.SP)},
		{"PC", fmt.Sprintf("$%X", cpu.Registers.PC)},
		{"IME", fmt.Sprintf("%t", cpu.Interrupts.Master_Enable)},
		{"IE", fmt.Sprintf("%0b", cpu.Interrupts.IE)},
		{"IF", fmt.Sprintf("%0b", cpu.Interrupts.IF)},
		{"Halted", fmt.Sprintf("%t", cpu.Halted)},
		{"Interrupts Queued", fmt.Sprintf("%t", cpu.Interrupts.Queued)},
		{"Stopped", fmt.Sprintf("%t", cpu.Stopped)},
		{"IsStuck", fmt.Sprintf("%t", cpu.IsStuck)},
		{"Cbg", fmt.Sprintf("%t", cpu.Mb.Cbg)},
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	for _, v := range report {
		table.Append(v)
	}

	table.Render()
}

func (c *CPU) CpSetFlags(a uint8, b uint8) {

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

func (c *CPU) AndSetFlags(a uint8, b uint8) uint8 {
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

func (c *CPU) OrSetFlags(a uint8, b uint8) uint8 {
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

func (c *CPU) XorSetFlags(a uint8, b uint8) uint8 {
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

func (c *CPU) SubSetFlags8(a uint8, b uint8) uint8 {
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

func (c *CPU) SbcSetFlags8(a uint8, b uint8) uint8 {
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

func (c *CPU) AddSetFlags16(a uint16, b uint16) uint32 {
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

func (c *CPU) AddSetFlags8(a uint8, b uint8) uint8 {
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

func (c *CPU) AdcSetFlags8(a uint8, b uint8) uint8 {
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

func (c *CPU) Inc(v uint8) uint8 {
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

func (c *CPU) Dec(v uint8) uint8 {
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
