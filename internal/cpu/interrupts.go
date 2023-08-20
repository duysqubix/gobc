package cpu

import (
	"log"

	"github.com/duysqubix/gobc/internal"
)

const (
	INTR_VBLANK    uint8 = 0x0 // VBlank interrupt      00000001 (bit 0)
	INTR_LCDSTAT   uint8 = 0x1 // LCD status interrupt  00000010 (bit 1)
	INTR_TIMER     uint8 = 0x2 // Timer interrupt       00000100 (bit 2)
	INTR_SERIAL    uint8 = 0x3 // Serial interrupt      00001000 (bit 3)
	INTR_HIGHTOLOW uint8 = 0x4 // Joypad interrupt      00010000 (bit 4)

	INTR_VBLANK_ADDR    uint16 = 0x0040 // VBlank interrupt Memory address
	INTR_LCDSTAT_ADDR   uint16 = 0x0048 // LCD status interrupt Memory address
	INTR_TIMER_ADDR     uint16 = 0x0050 // Timer interrupt Memory address
	INTR_SERIAL_ADDR    uint16 = 0x0058 // Serial interrupt Memory address
	INTR_HIGHTOLOW_ADDR uint16 = 0x0060 // Joypad interrupt Memory address
)

type Interrupts struct {
	Master_Enable bool  // Master interrupt enable
	IE            uint8 // Interrupt enable register
	IF            uint8 // Interrupt flag register
	Queued        bool  // Interrupt queued

}

func (i *Interrupts) CheckValidInterrupts() uint8 {
	valid_interrupts := i.IE & i.IF

	// Find flags that are set in IF but not enabled in IE
	disabled_interrupts := i.IF &^ i.IE

	if disabled_interrupts != 0 {
		log.Printf("Warning: Interrupt flags are set but not enabled. IE: %08b, IF: %08b", i.IE, i.IF)
	}

	return valid_interrupts
}

func (i *Interrupts) SetInterruptFlag(flag uint8)   { internal.SetBit(&i.IF, flag) }
func (i *Interrupts) SetInterruptEnable(flag uint8) { internal.SetBit(&i.IE, flag) }

func (i *Interrupts) ResetInterruptFlag(flag uint8)   { internal.ResetBit(&i.IF, flag) }
func (i *Interrupts) ResetInterruptEnable(flag uint8) { internal.ResetBit(&i.IE, flag) }

func (i *Interrupts) VBlankEnabled() bool    { return internal.IsBitSet(i.IE, INTR_VBLANK) }
func (i *Interrupts) LCDStatEnabled() bool   { return internal.IsBitSet(i.IE, INTR_LCDSTAT) }
func (i *Interrupts) TimerEnabled() bool     { return internal.IsBitSet(i.IE, INTR_TIMER) }
func (i *Interrupts) SerialEnabled() bool    { return internal.IsBitSet(i.IE, INTR_SERIAL) }
func (i *Interrupts) HighToLowEnabled() bool { return internal.IsBitSet(i.IE, INTR_HIGHTOLOW) }

func (i *Interrupts) IsVBlankSet() bool    { return internal.IsBitSet(i.IF, INTR_VBLANK) }
func (i *Interrupts) IsLCDStatSet() bool   { return internal.IsBitSet(i.IF, INTR_LCDSTAT) }
func (i *Interrupts) IsTimerSet() bool     { return internal.IsBitSet(i.IF, INTR_TIMER) }
func (i *Interrupts) IsSerialSet() bool    { return internal.IsBitSet(i.IF, INTR_SERIAL) }
func (i *Interrupts) IsHighToLowSet() bool { return internal.IsBitSet(i.IF, INTR_HIGHTOLOW) }

func (i *Interrupts) SetVBlank()    { internal.SetBit(&i.IF, INTR_VBLANK) }
func (i *Interrupts) SetLCDStat()   { internal.SetBit(&i.IF, INTR_LCDSTAT) }
func (i *Interrupts) SetTimer()     { internal.SetBit(&i.IF, INTR_TIMER) }
func (i *Interrupts) SetSerial()    { internal.SetBit(&i.IF, INTR_SERIAL) }
func (i *Interrupts) SetHighToLow() { internal.SetBit(&i.IF, INTR_HIGHTOLOW) }

func (i *Interrupts) ResetVBlank()    { internal.ResetBit(&i.IF, INTR_VBLANK) }
func (i *Interrupts) ResetLCDStat()   { internal.ResetBit(&i.IF, INTR_LCDSTAT) }
func (i *Interrupts) ResetTimer()     { internal.ResetBit(&i.IF, INTR_TIMER) }
func (i *Interrupts) ResetSerial()    { internal.ResetBit(&i.IF, INTR_SERIAL) }
func (i *Interrupts) ResetHighToLow() { internal.ResetBit(&i.IF, INTR_HIGHTOLOW) }

func (c *Cpu) HandleInterrupt(flag uint8, addr uint16) {
	log.Printf("Handling interrupt: %d\n", flag)
	intr := c.Interrupts

	pch := uint16(c.Registers.PC >> 8)
	pcl := uint16(c.Registers.PC & 0xFF)
	sp1 := c.Registers.SP - 1
	sp2 := c.Registers.SP - 2

	if c.Halted {
		c.Registers.PC += 1 // Escape HALT on retrun from interrupt
	}

	// Handle Interrupt Vector
	if intr.Master_Enable {
		intr.ResetInterruptFlag(flag)

		c.Mb.SetItem(&sp1, &pch)
		c.Mb.SetItem(&sp2, &pcl)
		c.Registers.SP -= 2
		c.Registers.PC = addr
		intr.Master_Enable = false

	}
}

func (c *Cpu) CheckForInterrupts() bool {
	intr := c.Interrupts

	if intr.Queued {
		// interrupt already queued, will only happen with a debugger -- NOT IMPLEMENTED
		return false
	}

	valid_interrupts := intr.CheckValidInterrupts() // holds the interrupts that are enabled and requested
	intr.Queued = false
	log.Printf("Valid Interrupts: %08b\n", valid_interrupts)

	// iterate through individual bits of valid_interrupts
	for i := uint8(0); i < 5; i++ {
		if internal.IsBitSet(valid_interrupts, i) {
			switch i {
			case INTR_VBLANK:
				c.HandleInterrupt(INTR_VBLANK, INTR_VBLANK_ADDR)
				intr.Queued = true

			case INTR_LCDSTAT:
				c.HandleInterrupt(INTR_LCDSTAT, INTR_LCDSTAT_ADDR)
				intr.Queued = true

			case INTR_TIMER:
				c.HandleInterrupt(INTR_TIMER, INTR_TIMER_ADDR)
				intr.Queued = true

			case INTR_SERIAL:
				c.HandleInterrupt(INTR_SERIAL, INTR_SERIAL_ADDR)
				intr.Queued = true

			case INTR_HIGHTOLOW:
				c.HandleInterrupt(INTR_HIGHTOLOW, INTR_HIGHTOLOW_ADDR)
				intr.Queued = true

			default:
				internal.Panicf("Invalid interrupt: %d", i)
			}
		}
	}
	return intr.Queued
}
