// https://gbdev.io/pandocs/The_Cartridge_Header.html

package cartridge

import (
	"fmt"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
)

const (
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
	CBG_FLAG uint16 = 0x0143

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
	filename string   // filename of the ROM
	RomBanks [][]byte // slice of ROM banks
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
	fmt.Println("Filename: ", c.filename)
}
