package cartridge

type Mbc3Cartridge struct {
	parent     *Cartridge
	hasBattery bool
	hasRTC     bool
}

func (c *Mbc3Cartridge) Init() {
	romName := c.parent.Filename

	// load save file if exists
	if c.hasBattery {
		LoadSRAM(romName, &c.parent.RamBanks, c.parent.RamBankCount)
	}
}

func (c *Mbc3Cartridge) SetItem(addr uint16, value uint8) {
	switch {
	case addr < 0x2000:
		if (value & 0b00001111) == 0b1010 {
			c.parent.RamBankEnabled = true
		} else if value == 0 {
			c.parent.RamBankEnabled = false
		} else {
			c.parent.RamBankEnabled = false
			logger.Debugf("Unexpected value for RAM bank enable: %#x", value)
		}

	case 0x2000 <= addr && addr < 0x4000:
		// value &= 0b01111111 // passes MBC30 test allowing upto 256 banks (4MB ROM)
		if value == 0 {
			value = 1
		}
		// logger.Debugf("Switching to ROM bank %d", value)
		c.parent.RomBankSelected = uint16(value)

	case 0x4000 <= addr && addr < 0x6000:
		c.parent.RamBankSelected = uint16(value)

	case 0x6000 <= addr && addr < 0x8000:
		if c.hasRTC {

		} else {
			logger.Debugf("RTC not present. Game attempted to write to RTC register %#x: %#x", addr, value)
		}

	case 0xA000 <= addr && addr < 0xC000:
		if c.parent.RamBankEnabled {
			if c.parent.RamBankSelected <= 0x07 {
				c.parent.RamBanks[c.parent.RamBankSelected%c.parent.RamBankCount][addr-0xA000] = value
			} else if 0x08 <= c.parent.RamBankSelected && c.parent.RamBankSelected <= 0x0C {
				// rtc set register}
			} else {
				logger.Errorf("Invalid RAM bank selected: %#x", c.parent.RamBankSelected)

			}
		}
	default:
		logger.Errorf("invalid address: %#x", addr)
	}
}

func (c *Mbc3Cartridge) GetItem(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		return c.parent.RomBanks[0][addr]

	case 0x4000 <= addr && addr < 0x8000:
		// logger.Debugf("Reading from ROM bank %#x", c.romBankSelect)
		return c.parent.RomBanks[c.parent.RomBankSelected%c.parent.RomBanksCount][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		if !c.parent.RamBankEnabled {
			return 0xFF
		}

		if c.hasRTC && 0x08 <= c.parent.RamBankSelected && c.parent.RamBankSelected <= 0x0C {
			// rtc get register
			return 0xFF
		} else {
			return c.parent.RamBanks[c.parent.RamBankSelected%c.parent.RamBankCount][addr-0xA000]
		}
	default:
		logger.Errorf("Read error! Can't read from %#x\n", addr)
	}
	return 0xff
}
