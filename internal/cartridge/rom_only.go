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
	switch {
	case addr < 0x4000:
		if value == 0 {
			value = 1
		}
		c.parent.RomBankSelected = uint8(value & 0b1)
	case 0x4000 <= addr && addr < 0xC000:
		rombank := c.parent.RomBankSelected % c.parent.RomBanksCount
		c.parent.RomBanks[rombank][addr-0x4000] = value
	}
}

func (c *RomOnlyCartridge) GetItem(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		return c.parent.RomBanks[0][addr]

	case 0x4000 <= addr && addr < 0x8000:
		rombank_n := c.parent.RomBankSelected % c.parent.RomBanksCount
		return c.parent.RomBanks[rombank_n][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		internal.Logger.Panicf("Reading from SRAM is not implemented yet")
	default:

	}
	return 0
}
