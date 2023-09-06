package cartridge

type Mbc3Cartridge struct {
	parent  *Cartridge
	sram    bool
	battery bool
	rtc     bool
}

func (c *Mbc3Cartridge) SetItem(addr uint16, value uint8) {
	switch {
	case addr < 0x2000:
		switch {
		case value&0xf == 0xa:
			c.parent.RamBankEnabled = true
		case value == 0:
			c.parent.RamBankEnabled = false

		default:
			c.parent.RamBankEnabled = false
			logger.Debugf("Unexpected command for MBC3: Address: %X, Value: %X", addr, value)
		}

	case 0x2000 <= addr && addr < 0x4000:
		value &= 0x7F
		if value == 0 {
			value = 1
		}
		c.parent.RomBankSelected = uint8(value)

	case 0x4000 <= addr && addr < 0x6000:
		c.parent.RamBankSelected = value

		if c.parent.RtcEnabled {
			// TODO: Handle RTC here
		} else {
			logger.Debugf("RTC not present. Game tried to issue RTC command at address: %X, value: %X", addr, value)
		}

	case 0xA000 <= addr && addr < 0xC000:
		if c.parent.RamBankEnabled {
			switch {
			case c.parent.RamBankSelected <= 0x03:
				c.parent.RamBanks[c.parent.RamBankSelected][addr-0xA000] = value
			case 0x08 <= c.parent.RamBankSelected && c.parent.RamBankSelected <= 0x0C:
				// TODO: Handle RTC here
			default:
				logger.Errorf("Unexpected RAM bank selected: %X", c.parent.RamBankSelected)
			}

		}

	default:
		logger.Errorf("Invalid writing address for MBC3: %X", addr)

	}
}

func (c *Mbc3Cartridge) GetItem(addr uint16) uint8 {
	rombank_n := c.parent.RomBankSelected % c.parent.RomBanksCount
	switch {
	case addr < 0x4000:
		return c.parent.RomBanks[0][addr]

	case 0x4000 <= addr && addr < 0x8000:

		return c.parent.RomBanks[rombank_n][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:

		if !c.parent.RamBankEnabled {
			return 0xFF
		}

		// TODO: Future handle RTC here

		return c.parent.RamBanks[rombank_n][addr-0xA000]
	default:

	}
	return 0
}
