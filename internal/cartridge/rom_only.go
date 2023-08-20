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
	// implement
	return 0
}
