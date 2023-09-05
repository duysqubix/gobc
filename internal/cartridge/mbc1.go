package cartridge

type Mbc1Cartridge struct {
	parent              *Cartridge
	sram                bool
	battery             bool
	rtc                 bool
	bankSelectRegister1 uint8
	bankSelectRegister2 uint8
}

func (c *Mbc1Cartridge) SetItem(addr uint16, value uint8) {

	switch {
	case addr < 0x2000:
		if (value & 0x0f) == 0x0a {
			c.parent.RamBankEnabled = true
		} else {
			c.parent.RamBankEnabled = false
		}

	case 0x2000 <= addr && addr < 0x4000:
		value &= 0x1f
		if value == 0 {
			value = 1
		}
		c.bankSelectRegister1 = value

	case 0x4000 <= addr && addr < 0x6000:
		c.bankSelectRegister2 = value & 0x3

	case 0x6000 <= addr && addr < 0x8000:
		c.parent.MemoryModel = value & 0x1
	case 0xA000 <= addr && addr < 0xC000:
		if c.parent.MemoryModel == 1 {
			c.parent.RamBankSelected = c.bankSelectRegister2
		} else {
			c.parent.RamBankSelected = 0
		}
		c.parent.RamBanks[c.parent.RamBankSelected][addr-0xA000] = value
	default:
		logger.Panicf("Memory write error! Can't write %#x to %#x\n", value, addr)
	}

}

func (c *Mbc1Cartridge) GetItem(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		if c.parent.MemoryModel == 1 {
			c.parent.RomBankSelected = (c.bankSelectRegister2 << 5) % c.parent.RomBanksCount
		} else {
			c.parent.RomBankSelected = 0
		}
		return c.parent.RomBanks[c.parent.RomBankSelected][addr]

	case 0x4000 <= addr && addr < 0x8000:
		c.parent.RomBankSelected = (c.bankSelectRegister2<<5)%c.parent.RomBanksCount | c.bankSelectRegister1
		bank := c.parent.RomBankSelected % uint8(len(c.parent.RomBanks))
		return c.parent.RomBanks[bank][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		if !c.parent.RamBankEnabled {
			return 0xff
		}

		if c.parent.MemoryModel == 1 {
			c.parent.RamBankSelected = c.bankSelectRegister2
		} else {
			c.parent.RamBankSelected = 0
		}

		bank := c.parent.RamBankSelected % uint8(len(c.parent.RamBanks))
		return c.parent.RamBanks[bank][addr-0xA000]
	default:
		logger.Errorf("Memory read error! Can't read from %#x\n", addr)
	}

	return 0
}
