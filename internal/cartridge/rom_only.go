package cartridge

type RomOnlyCartridge struct {
	parent *Cartridge
}

func (c *RomOnlyCartridge) SetItem(addr uint16, value uint8) {
	// do nothing
	// can't write to ROM and RAM doesn't exist on these carts
}

func (c *RomOnlyCartridge) GetItem(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		return c.parent.RomBanks[0][addr]

	case 0x4000 <= addr && addr < 0x8000:

		return c.parent.RomBanks[1][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		// RAM doesn't exist, but if an attempt is made, return 0xFF
		return 0xFF
	default:

	}
	return 0
}
