package motherboard

import (
	"fmt"
)

var interruptAddresses = map[byte]uint16{
	0: INTR_VBLANK_ADDR,    // V-Blank
	1: INTR_LCDSTAT_ADDR,   // LCDC Status
	2: INTR_TIMER_ADDR,     // Timer Overflow
	3: INTR_SERIAL_ADDR,    // Serial Transfer
	4: INTR_HIGHTOLOW_ADDR, // Hi-Lo P10-P13
}

type Interrupts struct {

	InterruptsEnabling bool  // Interrupts are being enabled
	InterruptsOn       bool  // Interrupts are on
	IE                 uint8 // Interrupt enable register
	IF                 uint8 // Interrupt flag register

}

func (i *Interrupts) ReportOn(interrupt uint8) []string {

	validInterrupts := i.CheckValidInterrupts()
	var interruptOn string = "ON"
	switch interrupt {
	case 0:
		if validInterrupts&0x01 == 0 {
			interruptOn = "OFF"
		}

		return []string{"VBLNK", interruptOn, fmt.Sprintf("IF: %v", (i.IF&0x01)>>0), fmt.Sprintf("IE: %v", (i.IE&0x01)>>0)}

	case 1:
		if validInterrupts&0x02 == 0 {
			interruptOn = "OFF"
		}
		return []string{"LCDST", interruptOn, fmt.Sprintf("IF: %v", (i.IF&0x02)>>1), fmt.Sprintf("IE: %v", (i.IE&0x02)>>1)}

	case 2:
		if validInterrupts&0x04 == 0 {
			interruptOn = "OFF"
		}
		return []string{"TIMER", interruptOn, fmt.Sprintf("IF: %v", (i.IF&0x04)>>2), fmt.Sprintf("IE: %v", (i.IE&0x04)>>2)}

	case 3:
		if validInterrupts&0x08 == 0 {
			interruptOn = "OFF"
		}
		return []string{"SERIA", interruptOn, fmt.Sprintf("IF: %v", (i.IF&0x08)>>3), fmt.Sprintf("IE: %v", (i.IE&0x08)>>3)}

	case 4:
		if validInterrupts&0x10 == 0 {
			interruptOn = "OFF"
		}
		return []string{"HILO", interruptOn, fmt.Sprintf("IF: %v", (i.IF&0x10)>>4), fmt.Sprintf("IE: %v", (i.IE&0x10)>>4)}

	}
	return []string{"", "", "", ""}
}

func (i *Interrupts) CheckValidInterrupts() uint8 {
	valid_interrupts := i.IE & i.IF

	// Find flags that are set in IF but not enabled in IE
	// disabled_interrupts := i.IF &^ i.IE

	// if disabled_interrupts != 0 {
	// 	internal.Logger.Warningf("Warning: Interrupt flags are set but not enabled. IE: %08b, IF: %08b", i.IE, i.IF)
	// }

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

	c.Mb.SetItem(sp-1, (pc&0xff00)>>8)
	c.Mb.SetItem(sp-2, pc&0xFF)
	c.Registers.SP -= 2
	c.Registers.PC = interruptAddresses[interrupt]
}
