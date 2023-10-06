package cartridge

import (
	"bytes"
	"encoding/binary"

	"github.com/duysqubix/gobc/internal"
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

var hiBanks *internal.Set = internal.NewSet()
var loBanks *internal.Set = internal.NewSet()

func (c *Mbc5Cartridge) SetItem(addr uint16, value uint8) {

	switch {
	case addr < 0x2000:
		// RAM Enable
		if value&0xF == 0xA {
			c.parent.RamBankEnabled = true
			logger.Debugf("RAM Bank enabled")
		} else {
			c.parent.RamBankEnabled = false
		}

	case 0x2000 <= addr && addr < 0x3000:
		// ROM Bank Number (lower 8 bits)
		c.romBankLow = value
		// if oldBank != c.GetRomBank() {
		// 	logger.Debugf("Switching to ROM bank %d", c.GetRomBank())
		// 	oldBank = c.GetRomBank()
		// }
		if !loBanks.Contains(int(value)) {
			loBanks.Add(int(value))
			loBanks.Sort()
			logger.Debugf("RombanksLow: %s", loBanks.Print())
		}

	case 0x3000 <= addr && addr < 0x4000:
		// ROM Bank Number (upper 1 bit)
		// c.parent.RomBankSelected &= 0x0FFF
		// c.parent.RomBankSelected |= (uint16(value&0x01) << 8)
		c.romBankHi = value & 0x01

		if !hiBanks.Contains(int(value)) {
			hiBanks.Add(int(value))
			logger.Debugf("RombanksHi: %s", hiBanks.Print())
		}
	case 0x4000 <= addr && addr < 0x6000:
		// RAM Bank Number 4bits
		c.parent.RamBankSelected = uint16(value & 0x0F)

	case 0xA000 <= addr && addr < 0xC000:
		// External RAM
		if c.parent.RamBankEnabled {
			c.parent.RamBanks[c.parent.RamBankSelected&c.parent.RamBankCount][addr-0xA000] = value
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
		bank := c.GetRomBank() & c.parent.RomBanksCount

		if bank != 0 {
			logger.Debugf("Reading from ROM bank %d", bank)
		}
		return c.parent.RomBanks[bank][addr-0x4000]

	case 0xA000 <= addr && addr < 0xC000:
		// External RAM
		if c.parent.RamBankEnabled {
			return c.parent.RamBanks[c.parent.RamBankSelected&c.parent.RamBankCount][addr-0xA000]
		}
		return 0xFF
	}

	return 0xFF
}
