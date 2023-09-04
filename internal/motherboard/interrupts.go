package motherboard

import (
	"github.com/duysqubix/gobc/internal"
)

var interruptAddresses = map[byte]uint16{
	0: INTR_VBLANK_ADDR,    // V-Blank
	1: INTR_LCDSTAT_ADDR,   // LCDC Status
	2: INTR_TIMER_ADDR,     // Timer Overflow
	3: INTR_SERIAL_ADDR,    // Serial Transfer
	4: INTR_HIGHTOLOW_ADDR, // Hi-Lo P10-P13
}

type Interrupts struct {
	// Master_Enable bool  // Master interrupt enable

	InterruptsEnabling bool  // Interrupts are being enabled
	InterruptsOn       bool  // Interrupts are on
	IE                 uint8 // Interrupt enable register
	IF                 uint8 // Interrupt flag register

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

func (c *CPU) SetInterruptFlag(f uint8) {
	req := c.Interrupts.IF | 0xE0
	c.Interrupts.IF = req | (1 << f)
}

func (c *CPU) ServiceInterrupt(interrupt uint8) {
	if !c.Interrupts.InterruptsOn && c.Halted {
		c.Halted = false
		c.Mb.Cpu.Registers.PC++
		return
	}

	c.Interrupts.InterruptsOn = false
	c.Halted = false
	c.Interrupts.IF &^= (1 << interrupt)
	sp := c.Registers.SP
	pc := c.Registers.PC

	sp1 := sp - 1
	sp2 := sp - 2
	pc1 := (pc & 0xff00) >> 8
	pc2 := pc & 0xFF

	c.Mb.SetItem(&sp1, &pc1)
	c.Mb.SetItem(&sp2, &pc2)
	c.Registers.SP -= 2
	c.Registers.PC = interruptAddresses[interrupt]
}

// func (c *CPU) handleInterrupt(f uint8, addr uint16) bool {
// 	flag := uint8(1 << f)

// 	if (c.Interrupts.IE&flag) != 0 && (c.Interrupts.IF&flag) != 0 {
// 		// clear flag
// 		if c.Halted {
// 			c.Registers.PC += 1 // Escape HALT on retrun from interrupt
// 		}

// 		// handle interrupt
// 		// logger.Warnf("Interrupts Active: %s\n", InterruptFlagDump(c.Interrupts.IF))

// 		if c.Interrupts.Master_Enable {
// 			// logger.Warnf("Setting Address to %#x\n", addr)
// 			// logger.Warnf("PRE: IE: %08b, IF: %08b, flag: %08b, addr: %#x\n", c.Interrupts.IE, c.Interrupts.IF, flag, addr)
// 			logger.Warnf("Interrupts Active: %s\n", InterruptFlagDump(c.Interrupts.IF))
// 			c.Interrupts.IF &^= flag

// 			sp1 := c.Registers.SP - 1
// 			pc1 := c.Registers.PC >> 8

// 			sp2 := c.Registers.SP - 2
// 			pc2 := c.Registers.PC & 0xFF
// 			c.Mb.SetItem(&sp1, &pc1)
// 			c.Mb.SetItem(&sp2, &pc2)
// 			// logger.Warnf("sp1: %#x, pc1: %#x, sp2: %#x, pc2: %#x\n", sp1, pc1, sp2, pc2)
// 			c.Registers.SP -= 2
// 			c.Registers.PC = addr
// 			c.Interrupts.Master_Enable = false
// 			// logger.Warnf("POST: IE: %08b, IF: %08b, flag: %08b, addr: %#x\n", c.Interrupts.IE, c.Interrupts.IF, flag, addr)

// 		}
// 		return true
// 	}
// 	return false
// }

// func (c *CPU) CheckForInterrupts() bool {
// 	intr := c.Interrupts

// 	if intr.Queued {
// 		return false
// 	}

// 	if (intr.IF&0b11111)&(intr.IE&0b11111) != 0 {
// 		switch {
// 		case c.handleInterrupt(INTR_VBLANK, INTR_VBLANK_ADDR):
// 			intr.Queued = true
// 		case c.handleInterrupt(INTR_LCDSTAT, INTR_LCDSTAT_ADDR):
// 			intr.Queued = true
// 		case c.handleInterrupt(INTR_TIMER, INTR_TIMER_ADDR):
// 			intr.Queued = true
// 		case c.handleInterrupt(INTR_SERIAL, INTR_SERIAL_ADDR):
// 			intr.Queued = true
// 		case c.handleInterrupt(INTR_HIGHTOLOW, INTR_HIGHTOLOW_ADDR):
// 			intr.Queued = true
// 		default:
// 			internal.Logger.Error("No interrupt triggered, but it should!")
// 			intr.Queued = false
// 		}
// 		return true
// 	} else {
// 		intr.Queued = false
// 		return false
// 	}

// }
