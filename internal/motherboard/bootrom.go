package motherboard

type BootRom struct {
	bootrom []uint8
}

func NewBootRom(cgb bool) *BootRom {
	var bootrom []uint8
	if cgb {
		bootrom = make([]uint8, 0x9000)
	} else {
		bootrom = make([]uint8, 0x100)
	}

	return &BootRom{
		bootrom: bootrom,
	}
}

func (br *BootRom) GetItem(addr uint16) uint8 {
	return br.bootrom[addr]
}
