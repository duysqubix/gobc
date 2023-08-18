// https://gbdev.io/pandocs/The_Cartridge_Header.html

package cartridge

import (
	"fmt"
	"os"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/olekukonko/tablewriter"
)

const (

	// Header Range
	HEADER_START_ADDR uint16 = 0x0100
	HEADER_END_ADDR   uint16 = 0x014F

	// Entry Point
	ENTRY_POINT_START_ADDR uint16 = 0x0100
	ENTRY_POINT_END_ADDR   uint16 = 0x0103

	// Nintendo Logo
	NINTENDO_LOGO_START_ADDR uint16 = 0x0104
	NINTENDO_LOGO_END_ADDR   uint16 = 0x0133

	// Title
	TITLE_START_ADDR uint16 = 0x0134
	TITLE_END_ADDR   uint16 = 0x0142

	/* CGB Flag
	$80 = Game supports CGB functions, but works on old gameboys also.
	$C0 = Game works on CGB only (physically the same as $80), hardware ignores bit 6.
	*/
	CBG_FLAG_ADDR uint16 = 0x0143

	// Manufacturer Code
	MANUFACTURER_CODE_START_ADDR uint16 = 0x013F
	MANUFACTURER_CODE_END_ADDR   uint16 = 0x0142

	// New Licensee Code
	NEW_LICENSEE_CODE_START_ADDR uint16 = 0x0144
	NEW_LICENSEE_CODE_END_ADDR   uint16 = 0x0145

	// SGB Flag
	SGB_FLAG_ADDR uint16 = 0x0146

	/* Cartridge Type
	Code 	Type
	--------------
	$00 	ROM ONLY
	$01 	MBC1
	$02 	MBC1+RAM
	$03 	MBC1+RAM+BATTERY
	$05 	MBC2
	$06 	MBC2+BATTERY
	$08 	ROM+RAM
	$09 	ROM+RAM+BATTERY
	$0B 	MMM01
	$0C 	MMM01+RAM
	$0D 	MMM01+RAM+BATTERY
	$0F 	MBC3+TIMER+BATTERY
	$10 	MBC3+TIMER+RAM+BATTERY
	$11 	MBC3
	$12 	MBC3+RAM
	$13 	MBC3+RAM+BATTERY
	$19 	MBC5
	$1A 	MBC5+RAM
	$1B 	MBC5+RAM+BATTERY
	$1C 	MBC5+RUMBLE
	$1D 	MBC5+RUMBLE+RAM
	$1E 	MBC5+RUMBLE+RAM+BATTERY
	$20 	MBC6
	$22 	MBC7+SENSOR+RUMBLE+RAM+BATTERY
	$FC 	POCKET CAMERA
	$FD 	BANDAI TAMA5
	$FE 	HuC3
	$FF 	HuC1+RAM+BATTERY
	*/
	CARTRIDGE_TYPE_ADDR uint16 = 0x0147

	/* ROM Size
	Code 	ROM Size 	Comment
	---------------------------
	$00 	32 KiB 	    2 (no banking)
	$01 	64 KiB 	    4
	$02 	128 KiB 	8
	$03 	256 KiB 	16
	$04 	512 KiB 	32
	$05 	1 MiB 	    64
	$06 	2 MiB 	    128
	$07 	4 MiB 	    256
	$08 	8 MiB 	    512
	$52 	1.1 MiB 	72 (96 banking)
	$53 	1.2 MiB 	80 (104 banking)
	$54 	1.5 MiB 	96 (120 banking)
	*/
	ROM_SIZE_ADDR uint16 = 0x0148 // (32KiB * (1 << ROM_SIZE_FLAG)) = ROM Size

	/* RAM Size
	Code 	SRAM Size 	Comment
	---------------------------
	$00 	0 	        No RAM
	$01 	- 	        Unused
	$02 	8 KiB 	    1 bank
	$03 	32 KiB 	    4 banks of 8 KiB each
	$04 	128 KiB 	16 banks of 8 KiB each
	$05 	64 KiB 	    8 banks of 8 KiB each
	*/
	SRAM_SIZE_ADDR uint16 = 0x0149

	/* Destination Code
	Code 	Destination
	---------------------------
	$00 	Japanese
	$01 	Overseas only
	*/
	DESTINATION_CODE_ADDR uint16 = 0x014A

	// Old Licensee Code
	OLD_LICENSEE_CODE_ADDR uint16 = 0x014B

	// Mask ROM Version number
	MASK_ROM_VERSION_NUMBER_ADDR uint16 = 0x014C

	/* Header Checksum
	Contains an 8bit checksum across the cartridge header bytes 0134-014C.
	Computed as follows in pseudo code:

	var checksum = 0
		for i = 0134h to 014Ch
			checksum = checksum - rom[i] - 1

	*/
	HEADER_CHECKSUM_ADDR uint16 = 0x014D

	// Global Checksum
	GLOBAL_CHECKSUM_START_ADDR uint16 = 0x014E
	GLOBAL_CHECKSUM_END_ADDR   uint16 = 0x014F

	// Memory Bank Size
	MEMORY_BANK_SIZE uint16 = 16_384 // 16 KiB (1024*16)
)

type Cartridge struct {
	filename string    // filename of the ROM
	RomBanks [][]uint8 // slice of ROM banks
}

func NewCartridge(filename *pathlib.Path) *Cartridge {
	rom_data, err := filename.ReadFile()
	if err != nil {
		internal.Panicf("Error reading ROM file: %s", err)
	}

	var rom_banks [][]uint8

	for i := 0; i < len(rom_data); i += int(MEMORY_BANK_SIZE) {
		end := i + int(MEMORY_BANK_SIZE)

		// prevent exceeding slice bounds
		if end > len(rom_data) {
			end = len(rom_data)
		}

		rom_banks = append(rom_banks, rom_data[i:end])
	}
	cart := Cartridge{
		RomBanks: rom_banks,
		filename: filename.Name(),
	}

	calc_checksum, valid := cart.ValidateChecksum()
	if !valid {
		internal.Panicf("Checksum invalid. Expected %02X, got %02X", cart.RomBanks[0][HEADER_CHECKSUM_ADDR], calc_checksum)
	}
	return &cart
}

func (c *Cartridge) ValidateChecksum() (uint8, bool) {
	var checksum uint8 = 0

	for i := TITLE_START_ADDR; i <= MASK_ROM_VERSION_NUMBER_ADDR; i++ {
		checksum -= c.RomBanks[0][i] + 1
	}

	return checksum, checksum == c.RomBanks[0][HEADER_CHECKSUM_ADDR]

}

func (c *Cartridge) Dump() {
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

	_, valid := c.ValidateChecksum()
	report := [][]string{
		{"Filename", c.filename},
		{"Title", string(title)},
		{"CBG Mode", CbgFlagMap[c.RomBanks[0][CBG_FLAG_ADDR]]},
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

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Attribute", "Value"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, v := range report {
		table.Append(v)
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
