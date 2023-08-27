package cartridge

import (
	"github.com/duysqubix/gobc/internal"
)

type RomOnlyCartridge struct {
	parent  *Cartridge
	sram    bool
	battery bool
	rtc     bool
}

func (c *RomOnlyCartridge) SetItem(addr uint16, value uint8) {
	// implement
	// logger.Debugf("Writing %#x to %#x on ROM Only Cartridge\n", value, addr)
	switch {
	case 0x0000 <= addr && addr < 0x4000:
		if value == 0 {
			value = 1
		}
		c.parent.RomBankSelected = uint8(value & 0b1)
		logger.Warnf("Cartridge: Switching to ROM Bank %d\n", c.parent.RomBankSelected)
	case 0x4000 <= addr && addr < 0xC000:
		logger.Warn("Cartridge: Writing to RAM is not implemented yet")

	}
}

func (c *RomOnlyCartridge) GetItem(addr uint16) uint8 {
	// logger.Debugf("Reading from %#x on ROM Only Cartridge\n", addr)
	switch {
	case 0x0000 <= addr && addr < 0x4000:
		return c.parent.RomBanks[0][addr]

	case 0x4000 <= addr && addr < 0x8000:
		rombank_n := c.parent.RomBankSelected % c.parent.RomBanksCount
		addr -= 0x4000
		return c.parent.RomBanks[rombank_n][addr]

	case 0xA000 <= addr && addr < 0xC000:
		internal.Logger.Panicf("Reading from SRAM is not implemented yet")
	default:

	}
	return 0
}
