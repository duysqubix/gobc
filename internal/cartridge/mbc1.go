package cartridge

type Mbc1Cartridge struct {
	parent        *Cartridge
	romBankSelect uint8
	ramBankSelect uint8
	mode          bool
	hasBattery    bool
}

func (c *Mbc1Cartridge) SetItem(addr uint16, value uint8) {
	switch {
	case addr < 0x2000:
		if (value & 0x0f) == 0x0a {
			c.parent.RamBankEnabled = true
			logger.Debugf("RAM Bank enabled")
		} else {
			c.parent.RamBankEnabled = false
		}

	case 0x2000 <= addr && addr < 0x4000:
		value &= 0x1f
		if value == 0 {
			value = 1
		}
		c.romBankSelect = value

	case 0x4000 <= addr && addr < 0x6000:
		c.ramBankSelect = value & 0x3
		logger.Debugf("RAM Bank selected: %d", c.ramBankSelect)

	case 0x6000 <= addr && addr < 0x8000:
		c.parent.MemoryModel = value & 0x1
		c.mode = value&0x1 == 0x1
		logger.Debugf("Memory model: %d", c.parent.MemoryModel)

	case 0xA000 <= addr && addr < 0xC000:
		if !c.parent.RamBankEnabled {
			return
		}

		if c.parent.MemoryModel == 1 {
			c.parent.RamBankSelected = c.ramBankSelect
		} else {
			c.parent.RamBankSelected = 0
		}
		logger.Debugf("Writing %#x to %#x on RAM bank %d/%d (%d)\n", value, addr, c.parent.RamBankSelected, c.parent.RamBankCount, c.parent.RamBankSelected%c.parent.RamBankCount)
		c.parent.RamBanks[c.parent.RamBankSelected%c.parent.RamBankCount][addr-0xA000] = value
	default:
		logger.Panicf("Memory write error! Can't write %#x to %#x\n", value, addr)
	}
}

func (c *Mbc1Cartridge) GetItem(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		if c.parent.MemoryModel == 1 {
			c.parent.RomBankSelected = (c.ramBankSelect << 5) % c.parent.RomBanksCount
		} else {
			c.parent.RomBankSelected = 0
		}
		return c.parent.RomBanks[c.parent.RomBankSelected][addr]

	case 0x4000 <= addr && addr < 0x8000:
		c.parent.RomBankSelected = (c.ramBankSelect<<5)%c.parent.RomBanksCount | c.romBankSelect
		bank := c.parent.RomBankSelected % uint8(len(c.parent.RomBanks))
		return c.parent.RomBanks[bank][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		if !c.parent.RamBankEnabled {
			return 0xff
		}

		if c.parent.MemoryModel == 1 {
			c.parent.RamBankSelected = c.ramBankSelect
		} else {
			c.parent.RamBankSelected = 0
		}

		bank := c.parent.RamBankSelected % uint8(c.parent.RamBankCount)
		return c.parent.RamBanks[bank][addr-0xA000]
	default:
		logger.Errorf("Memory read error! Can't read from %#x\n", addr)
	}

	return 0
}
