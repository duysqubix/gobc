package cartridge

type Mbc3Cartridge struct {
	parent  *Cartridge
	sram    bool
	battery bool
	rtc     bool
}

func (c *Mbc3Cartridge) SetItem(addr uint16, value uint8) {
	// implement
}

func (c *Mbc3Cartridge) GetItem(addr uint16) uint8 {
	// implement
	return 0
}
