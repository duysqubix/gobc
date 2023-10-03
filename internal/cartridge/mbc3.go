package cartridge

import (
	"bytes"
	"encoding/binary"
)

type Mbc3Cartridge struct {
	parent     *Cartridge
	hasBattery bool
	hasRTC     bool
	latchGate1 bool
	latchGate2 bool
}

func (c *Mbc3Cartridge) Init() {

	// load save file if exists
	if c.hasBattery {
		LoadSRAM(c.parent.Filename, &c.parent.RamBanks, c.parent.RamBankCount)
	}

	if c.hasRTC {
		c.parent.RtcEnabled = true
	}
}

func (c *Mbc3Cartridge) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, c.hasBattery) // Has Battery
	binary.Write(buf, binary.LittleEndian, c.hasRTC)     // Has RTC
	binary.Write(buf, binary.LittleEndian, c.latchGate1) // Latch Gate 1
	binary.Write(buf, binary.LittleEndian, c.latchGate2) // Latch Gate 2
	binary.Write(buf, binary.LittleEndian, Grtc.Serialize().Bytes())

	logger.Debug("Serialized MBC3 state")
	return buf
}

func (c *Mbc3Cartridge) Deserialize(data *bytes.Buffer) error {
	if err := binary.Read(data, binary.LittleEndian, &c.hasBattery); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &c.hasRTC); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &c.latchGate1); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &c.latchGate2); err != nil {
		return err
	}

	if err := Grtc.Deserialize(data); err != nil {
		return err
	}

	return nil
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
			if (!c.latchGate1 && value == 0) && !c.latchGate2 {
				c.latchGate1 = true
				c.latchGate2 = false
				return
			} else if (!c.latchGate2 && value == 1) && c.latchGate1 {
				c.latchGate2 = true
				c.latchGate1 = true
				return
			} else if c.latchGate1 && c.latchGate2 {
				// logger.Debugf("Latching RTC")
				Grtc.Latch()
				c.latchGate1 = false
				c.latchGate2 = false
				return
			} else {
				c.latchGate1 = false
				c.latchGate2 = false
				return
			}

		} else {
			logger.Debugf("RTC not present. Game attempted to write to RTC register %#x: %#x", addr, value)

		}

	case 0xA000 <= addr && addr < 0xC000:
		if c.parent.RamBankEnabled {
			if c.parent.RamBankSelected <= 0x07 {
				c.parent.RamBanks[c.parent.RamBankSelected%c.parent.RamBankCount][addr-0xA000] = value
			} else if 0x08 <= c.parent.RamBankSelected && c.parent.RamBankSelected <= 0x0C {
				Grtc.SetItem(c.parent.RamBankSelected, value)
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

		if c.hasRTC && (0x08 <= c.parent.RamBankSelected && c.parent.RamBankSelected <= 0x0C) {
			return Grtc.GetItem(c.parent.RamBankSelected)
		} else {
			return c.parent.RamBanks[c.parent.RamBankSelected%c.parent.RamBankCount][addr-0xA000]
		}
	default:
		logger.Errorf("Read error! Can't read from %#x\n", addr)
	}
	return 0xff
}
