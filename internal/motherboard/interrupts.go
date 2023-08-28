package motherboard

import (
	"github.com/duysqubix/gobc/internal"
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
		internal.Logger.Warningf("Warning: Interrupt flags are set but not enabled. IE: %08b, IF: %08b", i.IE, i.IF)
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

func (c *CPU) handleInterrupt(f uint8, addr uint16) bool {
	flag := uint8(1 << f)

	if (c.Interrupts.IE&flag) != 0 && (c.Interrupts.IF&flag) != 0 {
		// clear flag
		if c.Halted {
			c.Registers.PC += 1 // Escape HALT on retrun from interrupt
		}

		// handle interrupt
		if c.Interrupts.Master_Enable {
			logger.Warnf("Setting Address to %#x\n", addr)
			logger.Warnf("IE: %08b, IF: %08b, flag: %08b, addr: %#x\n", c.Interrupts.IE, c.Interrupts.IF, flag, addr)
			logger.Warnf("Interrupts Active: %s\n", InterruptFlagDump(c.Interrupts.IF))
			c.Interrupts.IF &^= flag
			sp1 := c.Registers.SP - 1
			pc1 := c.Registers.PC >> 8

			sp2 := c.Registers.SP - 2
			pc2 := c.Registers.PC & 0xFF
			c.Mb.SetItem(&sp1, &pc1)
			c.Mb.SetItem(&sp2, &pc2)
			logger.Warnf("sp1: %#x, pc1: %#x, sp2: %#x, pc2: %#x\n", sp1, pc1, sp2, pc2)
			c.Registers.SP -= 2
			c.Registers.PC = addr
			c.Interrupts.Master_Enable = false
		}
		return true
	}
	return false
}

func (c *CPU) CheckForInterrupts() bool {
	intr := c.Interrupts

	if intr.Queued {
		return false
	}

	if (intr.IF&0b11111)&(intr.IE&0b11111) != 0 {
		switch {
		case c.handleInterrupt(INTR_VBLANK, INTR_VBLANK_ADDR):
			intr.Queued = true
		case c.handleInterrupt(INTR_LCDSTAT, INTR_LCDSTAT_ADDR):
			intr.Queued = true
		case c.handleInterrupt(INTR_TIMER, INTR_TIMER_ADDR):
			intr.Queued = true
		case c.handleInterrupt(INTR_SERIAL, INTR_SERIAL_ADDR):
			intr.Queued = true
		case c.handleInterrupt(INTR_HIGHTOLOW, INTR_HIGHTOLOW_ADDR):
			intr.Queued = true
		default:
			internal.Logger.Error("No interrupt triggered, but it should!")
			intr.Queued = false
		}
		return true
	} else {
		intr.Queued = false
		return false
	}

}
