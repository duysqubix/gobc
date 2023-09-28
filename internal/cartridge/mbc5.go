package cartridge

type Mbc5Cartridge struct {
	parent     *Cartridge
	hasBattery bool
	hasRumble  bool
}

func (c *Mbc5Cartridge) SetItem(addr uint16, value uint8) {

}

func (c *Mbc5Cartridge) GetItem(addr uint16) uint8 {
	return 0
}
