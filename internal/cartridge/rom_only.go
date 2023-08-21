package cartridge

type RomOnlyCartridge struct {
	parent  *Cartridge
	sram    bool
	battery bool
	rtc     bool
}

func (c *RomOnlyCartridge) SetItem(addr uint16, value uint8) {
	// implement
}

func (c *RomOnlyCartridge) GetItem(addr uint16) uint8 {
	switch {
	case 0x0000 <= addr && addr < 0x4000:
		return c.parent.RomBanks[0][addr]

	case 0x4000 <= addr && addr < 0x8000:
		rombank_n := c.parent.RomBankSelected % c.parent.RomBanksCount
		addr -= 0x4000
		return c.parent.RomBanks[rombank_n][addr]

	case 0xA000 <= addr && addr < 0xC000:
		if !

	}
	return 0
}
