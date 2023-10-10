package cartridge

import (
	"bytes"
	"encoding/binary"
)

type Mbc5Cartridge struct {
	parent     *Cartridge
	hasBattery bool
	hasRumble  bool
	romBankLow uint8
	romBankHi  uint8
}

func (c *Mbc5Cartridge) Init() {
	c.hasBattery = true
	c.hasRumble = false
	c.romBankLow = 1
	c.romBankHi = 0
	logger.Debugf("Initializing MBC5, with ROM bank %d", c.GetRomBank())
}

func (c *Mbc5Cartridge) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, c.hasBattery) // Has Battery
	binary.Write(buf, binary.LittleEndian, c.hasRumble)  // Has Rumble
	binary.Write(buf, binary.LittleEndian, c.romBankLow) // ROM Bank Low
	binary.Write(buf, binary.LittleEndian, c.romBankHi)  // ROM Bank Hi
	return buf

}

func (c *Mbc5Cartridge) Deserialize(data *bytes.Buffer) error {
	if err := binary.Read(data, binary.LittleEndian, &c.hasBattery); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &c.hasRumble); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &c.romBankLow); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &c.romBankHi); err != nil {
		return err
	}

	return nil
}

func (c *Mbc5Cartridge) GetRomBank() uint16 {
	return (uint16(c.romBankHi) << 8) | uint16(c.romBankLow)
}

func (c *Mbc5Cartridge) SetItem(addr uint16, value uint8) {

	switch {
	case addr < 0x2000:
		// RAM Enable
		c.parent.RamBankEnabled = (value & 0x0f) == 0x0a

	case 0x2000 <= addr && addr < 0x3000:
		// ROM Bank Number (lower 8 bits)
		c.romBankLow = value

	case 0x3000 <= addr && addr < 0x4000:
		// ROM Bank Number (upper 1 bit)
		c.romBankHi = value & 0x01

	case 0x4000 <= addr && addr < 0x6000:
		// RAM Bank Number 4bits
		c.parent.RamBankSelected = uint16(value & 0x0f)
	case 0xA000 <= addr && addr < 0xC000:
		// External RAM
		if c.parent.RamBankEnabled {
			ramBank := c.parent.RamBankSelected & c.parent.RamBankCount
			c.parent.RamBanks[ramBank][addr-0xA000] = value
		}
	}

}

func (c *Mbc5Cartridge) GetItem(addr uint16) uint8 {
	switch {
	case addr < 0x4000:
		// ROM Bank 0
		return c.parent.RomBanks[0][addr]
	case 0x4000 <= addr && addr < 0x8000:
		// Switchable ROM Bank
		romBank := c.GetRomBank() & (c.parent.RomBanksCount - 1)
		// logger.Debugf("Reading from ROM bank %d", bank)

		return c.parent.RomBanks[romBank][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		// External RAM
		if c.parent.RamBankEnabled {
			ramBank := c.parent.RamBankSelected & c.parent.RamBankCount

			return c.parent.RamBanks[ramBank][addr-0xA000]
		}
		return 0xFF
	}

	return 0xFF
}
