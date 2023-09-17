package cartridge

type RomOnlyCartridge struct {
	parent  *Cartridge
	sram    bool
	battery bool
	rtc     bool
}

func (c *RomOnlyCartridge) SetItem(addr uint16, value uint8) {
	switch {
	case addr < 0x4000:
		c.parent.RomBanks[0][addr] = value

	case 0x4000 <= addr && addr < 0x8000:
		c.parent.RomBanks[1][addr-0x4000] = value

	case 0xA000 <= addr && addr < 0xC000:
		c.parent.RamBanks[0][addr-0xA000] = value
	}
}

func (c *RomOnlyCartridge) GetItem(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		return c.parent.RomBanks[0][addr]

	case 0x4000 <= addr && addr < 0x8000:

		return c.parent.RomBanks[1][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		return c.parent.RamBanks[0][addr-0xA000]
	default:

	}
	return 0
}
