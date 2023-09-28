// https://gbdev.io/pandocs/The_Cartridge_Header.html

package cartridge

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/olekukonko/tablewriter"
)

// CartridgeTable is a table of cartridge types
// type CartridgeType struct {
// 	MBC     uint8
// 	SRAM    bool
// 	Battery bool
// 	RTC     bool
// }

type CartridgeType interface {
	SetItem(uint16, uint8)
	GetItem(uint16) uint8
}

// var CARTRIDGE_TABLE = map[uint8]CartridgeType{
// 	// #    MBC     SRAM    Battery RTC
// 	0x00: {ROM_ONLY, false, false, false}, // ROM ONLY
// 0x01: {MBC1, false, false, false},     // MBC1
// 0x02: {MBC1, true, false, false},      // MBC1+RAM
// 0x03: {MBC1, true, true, false},       // MBC1+RAM+BATTERY
// 0x05: {MBC2, false, false, false},     // MBC2
// 0x06: {MBC2, false, true, false},      // MBC2+BATTERY
// 0x08: {ROM_ONLY, true, false, false},  // ROM+RAM
// 0x09: {ROM_ONLY, true, true, false},   // ROM+RAM+BATTERY
// 0x0F: {MBC3, false, true, true},       // MBC3+TIMER+BATTERY
// 0x10: {MBC3, true, true, true},        // MBC3+TIMER+RAM+BATTERY
// 0x11: {MBC3, false, false, false},     // MBC3
// 0x12: {MBC3, true, false, false},      // MBC3+RAM
// 0x13: {MBC3, true, true, false},       // MBC3+RAM+BATTERY
// 0x19: {MBC5, false, false, false},     // MBC5
// 0x1A: {MBC5, true, false, false},      // MBC5+RAM
// 0x1B: {MBC5, true, true, false},       // MBC5+RAM+BATTERY
// 0x1C: {MBC5, false, false, true},      // MBC5+RUMBLE
// 0x1D: {MBC5, true, false, true},       // MBC5+RUMBLE+RAM
// 0x1E: {MBC5, true, true, false},       // MBC5+RUMBLE+RAM+BATTERY
// }

var CARTRIDGE_TABLE = map[uint8]func(*Cartridge) CartridgeType{
	// ROM ONLY
	0x00: func(c *Cartridge) CartridgeType {
		return &RomOnlyCartridge{parent: c}
	},

	0x01: func(c *Cartridge) CartridgeType {
		return &Mbc1Cartridge{
			parent:        c,
			romBankSelect: 1,
			mode:          false,
		}
	},

	// MBC1+RAM
	0x02: func(c *Cartridge) CartridgeType {
		return &Mbc1Cartridge{
			parent:        c,
			romBankSelect: 1,
			mode:          false,
		}
	},

	// MBC1+RAM+BATTERY
	0x03: func(c *Cartridge) CartridgeType {
		return &Mbc1Cartridge{
			parent:        c,
			romBankSelect: 1,
			mode:          false,
		}
	},

	// MBC3+TIMER+RAM+BATTERY
	0x10: func(c *Cartridge) CartridgeType {
		return &Mbc3Cartridge{
			parent:     c,
			hasBattery: true,
			hasRTC:     true,
		}
	},
	// MBC3+RAM+BATTERY
	0x13: func(c *Cartridge) CartridgeType {
		return &Mbc3Cartridge{
			parent:     c,
			hasBattery: true,
			hasRTC:     false,
		}

	},
	// MBC5+RAM+BATTERY
	0x1b: func(c *Cartridge) CartridgeType {
		return &Mbc5Cartridge{
			parent:     c,
			hasBattery: true,
		}
	},
}

type Cartridge struct {
	filename  string        // filename of the ROM
	CartType  CartridgeType // type of cartridge
	Randomize bool          // whether to randomize RAM banks on startup

	// ROM Banks
	// RomBanks        [128][MEMORY_BANK_SIZE]uint8 // slice of ROM banks
	RomBanks        [][]uint8 // slice of ROM banks
	RomBanksCount   uint16    // number of ROM banks
	RomBankSelected uint16    // currently selected ROM bank

	// RAM Banks
	RamBanks           [16][RAM_BANK_SIZE]uint8 // slice of RAM banks
	RamBankCount       uint16                   // number of RAM banks
	RamBankSelected    uint16                   // currently selected RAM bank
	RamBankEnabled     bool                     // whether RAM bank is supported
	RamBankInitialized bool                     // whether RAM bank has been initialized

	// RTC
	RtcEnabled bool // whether RTC is enabled

	MemoryModel uint8 // 0 = 16/8, 1 = 4/32
}

func LoadRomBanks(rom_data []byte, dummy_data bool) [][]uint8 {
	logger.Infof("Processing ROM file of size %d bytes", len(rom_data))
	var rom_banks [][]uint8

	if dummy_data {
		bank := make([]byte, MEMORY_BANK_SIZE)
		for j := range bank {
			bank[j] = 0xff // fill with 0xff
		}
		bank[CARTRIDGE_TYPE_ADDR] = 0x0
		rom_banks = append(rom_banks, bank)
		return rom_banks
	}

	rom_len := len(rom_data)

	for i := 0; i < rom_len; i += int(MEMORY_BANK_SIZE) {
		end := i + int(MEMORY_BANK_SIZE)

		// prevent exceeding slice bounds
		if end > rom_len {
			end = rom_len
		}

		bank := make([]uint8, MEMORY_BANK_SIZE)
		for j := range bank {
			bank[j] = 0xff
		}
		rom_banks = append(rom_banks, rom_data[i:end])
	}

	return rom_banks
}

func LoadRomBanksV2(rom_data []byte, dummy_data bool) [128][MEMORY_BANK_SIZE]uint8 {
	logger.Infof("Processing ROM file of size %d bytes", len(rom_data))

	var romBanksFlat [int(MEMORY_BANK_SIZE) * 128]uint8
	if dummy_data {
		for j := range romBanksFlat {
			romBanksFlat[j] = 0xff // fill with 0xff
		}
		romBanksFlat[CARTRIDGE_TYPE_ADDR] = 0x0
	} else {

		copy(romBanksFlat[:], rom_data)

		// fill rest of romBanksFlat with 0xff
		for i := len(rom_data); i < int(MEMORY_BANK_SIZE)*128; i++ {
			romBanksFlat[i] = 0xff
		}
	}

	// convert romBanksFlat into 128 banks of 16KiB each
	var rom_banks [128][MEMORY_BANK_SIZE]uint8
	for i := 0; i < 128; i++ {
		for j := 0; j < int(MEMORY_BANK_SIZE); j++ {
			rom_banks[i][j] = romBanksFlat[i*int(MEMORY_BANK_SIZE)+j]
		}
	}

	return rom_banks
}

func NewCartridge(filename *pathlib.Path) *Cartridge {
	var rom_data []byte
	var err error
	// var rom_banks [128][MEMORY_BANK_SIZE]uint8
	var rom_banks [][]uint8
	var fname string

	if filename != nil {
		rom_data, err = filename.ReadFile()
		fname = filename.Name()
		if err != nil {
			internal.Logger.Panicf("Error reading ROM file: %s", err)
		}
		rom_banks = LoadRomBanks(rom_data, false)

	} else {
		logger.Warn("No ROM file specified, running tests")
		rom_banks = LoadRomBanks(nil, true)
		fname = ""
	}

	var ramBankCount uint16

	switch rom_banks[0][SRAM_SIZE_ADDR] {
	case 0x00:
		ramBankCount = 0
	case 0x01:
		logger.Panicf("RAM size is unused")
	case 0x02:
		ramBankCount = 1
	case 0x03:
		ramBankCount = 4
	case 0x04:
		ramBankCount = 16
	case 0x05:
		ramBankCount = 8
	default:
		logger.Panicf("Invalid RAM size: %02X", rom_banks[0][SRAM_SIZE_ADDR])
	}

	var romBankCount uint16
	switch rom_banks[0][ROM_SIZE_ADDR] {
	case 0x00:
		romBankCount = 2
	case 0x01:
		romBankCount = 4
	case 0x02:
		romBankCount = 8
	case 0x03:
		romBankCount = 16
	case 0x04:
		romBankCount = 32
	case 0x05:
		romBankCount = 64
	case 0x06:
		romBankCount = 128
	case 0x07:
		romBankCount = 256
	case 0x08:
		romBankCount = 512
	}
	logger.Debugf("Detected ROM bank count: %d, Calculated Number of ROM Banks: %d", romBankCount, len(rom_data)/int(MEMORY_BANK_SIZE))
	if romBankCount != uint16(len(rom_data)/int(MEMORY_BANK_SIZE)) {
		logger.Fatalf("ROM bank count mismatch. Expected %d, got %d", romBankCount, len(rom_data)/int(MEMORY_BANK_SIZE))
	}

	cart := Cartridge{
		RomBanks:        rom_banks,
		filename:        fname,
		RomBanksCount:   romBankCount,
		RomBankSelected: 1,
		RamBankSelected: 0,
		RamBankCount:    ramBankCount,
		MemoryModel:     0,
		Randomize:       false,
	}

	cart_type_addr := rom_banks[0][CARTRIDGE_TYPE_ADDR]
	cartTypeConstructor := CARTRIDGE_TABLE[rom_banks[0][CARTRIDGE_TYPE_ADDR]]

	if cartTypeConstructor == nil {
		logger.Errorf("Cartridge type not supported: %02X", cart_type_addr)
		os.Exit(0)
	}
	cart.CartType = cartTypeConstructor(&cart)

	calc_checksum, valid := cart.ValidateChecksum()
	if !valid {
		logger.Fatalf("Checksum invalid. Expected %02X, got %02X", cart.RomBanks[0][HEADER_CHECKSUM_ADDR], calc_checksum)
	}

	// initialize RAM banks to maximum size of 128KiB
	// cart.initRambanks()

	logger.Info("Cartridge RAM Initialized")
	cart.Dump(os.Stdout)
	logger.Infof("ROM file loaded successfully: %s", filename)
	logger.Infof("Cartridge Initialized: %s", reflect.TypeOf(cart.CartType))
	logger.Infof("ROM Banks: %d, Size: %dKb", cart.RomBanksCount, cart.RomBanksCount*16)
	logger.Infof("RAM Banks: %d, Size: %dKb", cart.RamBankCount, cart.RamBankCount*8)
	return &cart
}

func (c *Cartridge) ValidateChecksum() (uint8, bool) {
	var checksum uint8 = 0

	for i := TITLE_START_ADDR; i <= MASK_ROM_VERSION_NUMBER_ADDR; i++ {
		checksum -= c.RomBanks[0][i] + 1
	}

	return checksum, checksum == c.RomBanks[0][HEADER_CHECKSUM_ADDR]

}

func (c *Cartridge) CgbModeEnabled() bool {
	if c.RomBanksCount == 0 {
		// no ROM banks loaded -- only possible if we're running tests
		return false
	}

	return c.RomBanks[0][CBG_FLAG_ADDR] == 0x80 || c.RomBanks[0][CBG_FLAG_ADDR] == 0xC0
}

func (c *Cartridge) Dump(writer io.Writer) {
	title := c.RomBanks[0][TITLE_START_ADDR : TITLE_END_ADDR+1]
	license1 := NewLicenseeCodeMap[c.RomBanks[0][NEW_LICENSEE_CODE_START_ADDR]]
	license2 := NewLicenseeCodeMap[c.RomBanks[0][NEW_LICENSEE_CODE_END_ADDR]]

	cartridge_type := CartridgeTypeMap[c.RomBanks[0][CARTRIDGE_TYPE_ADDR]]
	rom_size := RomSizeMap[c.RomBanks[0][ROM_SIZE_ADDR]]
	ram_size := RamSizeMap[c.RomBanks[0][SRAM_SIZE_ADDR]]

	oldlicense1 := OldLicenseeCodeMap[c.RomBanks[0][OLD_LICENSEE_CODE_ADDR]]

	sbg_mode_enabled := "No"
	if c.RomBanks[0][SGB_FLAG_ADDR] == 0x03 {
		sbg_mode_enabled = "Yes"
	}

	cgb_mode := c.RomBanks[0][CBG_FLAG_ADDR]
	var cgb_mode_desc string
	if c.CgbModeEnabled() {
		cgb_mode_desc = CgbFlagMap[cgb_mode]
	} else {
		cgb_mode_desc = "CGB Not Supported"
	}

	_, valid := c.ValidateChecksum()
	report := [][]string{
		{"Filename", c.filename},
		{"Title", string(title)},
		{"CBG Mode", cgb_mode_desc},
		{"SBG Mode", sbg_mode_enabled},
		{"New Licensee Code", fmt.Sprintf("%s, %s", license1, license2)},
		{"Cartridge Type", cartridge_type},
		{"ROM Size", rom_size.String()},
		{"RAM Size", ram_size.String()},
		{"Old Licensee Code", oldlicense1},
		{"Header Checksum", fmt.Sprintf("$%02X", c.RomBanks[0][HEADER_CHECKSUM_ADDR])},
		{"Header Checksum Valid", fmt.Sprintf("%t", valid)},
		{"Global Checksum", fmt.Sprintf("$%02X", c.RomBanks[0][GLOBAL_CHECKSUM_START_ADDR])},
	}

	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Attribute", "Value"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, v := range report {
		table.Append(v)
	}

	table.Render()
}

// writes the ROM into a file that is human readle
// [bank#]/[addr]: [opcode] [value] [description]
func (c *Cartridge) DumpInstructionSet(writer io.Writer, include_nop bool) {

	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Bank", "Address", "Opcode", "Value", "Description", "Notes"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	// var str string
	var data [][]string
	var opcode uint16
	var notes string
	for cntr, bank := range c.RomBanks {
		addr_start := 0x00
		if cntr > 0 {
			addr_start = 0x00
		}
		for addr := addr_start; addr < len(bank); addr++ {

			oplen := internal.OPCODE_LENGTHS[bank[addr]]
			if bank[addr] == 0x00 && !include_nop {
				continue
			}

			opcode = uint16(bank[addr])
			if opcode == 0xcb {
				addr++
				opcode = uint16(bank[addr]) + 0x100
				notes = "CB Prefix"
			}

			switch oplen {
			case 2:
				orig_addr := addr
				notes = "8bit Immediate"
				opcode = uint16(bank[addr])
				// immediate 8bit
				addr++
				value := bank[addr]
				data = append(data, []string{
					fmt.Sprintf("Bank_%d", cntr),
					fmt.Sprintf("$%04X", orig_addr),
					fmt.Sprintf("$%02X", opcode),
					fmt.Sprintf("$%02X", value),
					internal.OPCODE_NAMES[opcode],
					notes,
				})
			case 3:
				// immediate 16bit
				orig_addr := addr
				notes = "16bit Immediate"
				opcode = uint16(bank[addr])
				addr++
				h := bank[addr]
				addr++
				l := bank[addr]
				value := (uint16(l) << 8) | uint16(h) // swapped to show correct value
				data = append(data, []string{
					fmt.Sprintf("Bank_%d", cntr),
					fmt.Sprintf("$%04X", orig_addr),
					fmt.Sprintf("$%02X", opcode),
					fmt.Sprintf("$%04X", value),
					internal.OPCODE_NAMES[opcode],
					notes,
				})
			default:
				data = append(data, []string{
					fmt.Sprintf("Bank_%d", cntr),
					fmt.Sprintf("$%04X", addr),
					fmt.Sprintf("$%02X", opcode),
					"",
					internal.OPCODE_NAMES[opcode],
					notes,
				})
			}
		}
	}

	for _, row := range data {
		table.Append(row)
	}
	table.Render()
}

func (c *Cartridge) RawHeaderDump() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Address", "Value", "Description"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _i, v := range c.RomBanks[0][HEADER_START_ADDR : HEADER_END_ADDR+1] {
		var desc string
		i := uint16(_i) + HEADER_START_ADDR
		switch {
		case i >= ENTRY_POINT_START_ADDR && i <= ENTRY_POINT_END_ADDR:
			desc = "Entry Point"
		case i >= NINTENDO_LOGO_START_ADDR && i <= NINTENDO_LOGO_END_ADDR:
			desc = "Nintendo Logo"
		case i >= TITLE_START_ADDR && i <= TITLE_END_ADDR:
			desc = "Title"
		case i == CBG_FLAG_ADDR:
			desc = "CBG Flag"
		case i >= MANUFACTURER_CODE_START_ADDR && i <= MANUFACTURER_CODE_END_ADDR:
			desc = "Manufacturer Code"
		case i >= NEW_LICENSEE_CODE_START_ADDR && i <= NEW_LICENSEE_CODE_END_ADDR:
			desc = "New Licensee Code"
		case i == SGB_FLAG_ADDR:
			desc = "SGB Flag"
		case i == CARTRIDGE_TYPE_ADDR:
			desc = "Cartridge Type"
		case i == ROM_SIZE_ADDR:
			desc = "ROM Size"
		case i == SRAM_SIZE_ADDR:
			desc = "SRAM Size"
		case i == DESTINATION_CODE_ADDR:
			desc = "Destination Code"
		case i == OLD_LICENSEE_CODE_ADDR:
			desc = "Old Licensee Code"
		case i == MASK_ROM_VERSION_NUMBER_ADDR:
			desc = "Mask ROM Version Number"
		case i == HEADER_CHECKSUM_ADDR:
			desc = "Header Checksum"
		case i >= GLOBAL_CHECKSUM_START_ADDR && i <= GLOBAL_CHECKSUM_END_ADDR:
			desc = "Global Checksum"
		default:
			desc = "Unkown"
		}

		table.Append([]string{fmt.Sprintf("$%04X", i), fmt.Sprintf("$%02X", v), desc})
	}

	table.Render()
}
