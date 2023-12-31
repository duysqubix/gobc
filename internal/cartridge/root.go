package cartridge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/duysqubix/gobc/internal"
)

var logger = internal.Logger

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
	RAM_BANK_SIZE    uint16 = 8_192  // 8 KiB (1024*8)

	// Cartridge Types
	ROM_ONLY uint8 = 0x00
	MBC1     uint8 = 0x01
	MBC2     uint8 = 0x02
	MBC3     uint8 = 0x03
	MBC5     uint8 = 0x05
)

var CgbFlagMap = map[uint8]string{
	0x80: "CGB Only",
	0xC0: "CGB Supported",
}

var NewLicenseeCodeMap = map[uint8]string{
	0x00: "None",
	0x01: "Nintendo R&D1",
	0x08: "Capcom",
	0x13: "Electronic Arts",
	0x18: "Hudson Soft",
	0x19: "b-ai",
	0x20: "kss",
	0x22: "pow",
	0x24: "PCM Complete",
	0x25: "san-x",
	0x28: "Kemco Japan",
	0x29: "seta",
	0x30: "Viacom",
	0x31: "Nintendo",
	0x32: "Bandai",
	0x33: "Ocean/Acclaim",
	0x34: "Konami",
	0x35: "Hector",
	0x37: "Taito",
	0x38: "Hudson",
	0x39: "Banpresto",
	0x41: "Ubi Soft",
	0x42: "Atlus",
	0x44: "Malibu",
	0x46: "angel",
	0x47: "Bullet-Proof",
	0x49: "irem",
	0x50: "Absolute",
	0x51: "Acclaim",
	0x52: "Activision",
	0x53: "American sammy",
	0x54: "Konami",
	0x55: "Hi tech entertainment",
	0x56: "LJN",
	0x57: "Matchbox",
	0x58: "Mattel",
	0x59: "Milton Bradley",
	0x60: "Titus",
	0x61: "Virgin",
	0x64: "LucasArts",
	0x67: "Ocean",
	0x69: "Electronic Arts",
	0x70: "Infogrames",
	0x71: "Interplay",
	0x72: "Broderbund",
	0x73: "sculptured",
	0x75: "sci",
	0x78: "THQ",
	0x79: "Accolade",
	0x80: "misawa",
	0x83: "lozc",
	0x86: "tokuma Shoten Intermedia",
	0x87: "tsukuda ori",
	0x91: "Chunsoft",
	0x92: "Video system",
	0x93: "Ocean/Acclaim",
	0x95: "Varie",
	0x96: "Yonezawa/s'pal",
	0x97: "Kaneko",
	0x99: "Pack in soft",
	0xA4: "Konami (Yu-Gi-Oh!)",
}

var CartridgeTypeMap = map[uint8]string{
	0x00: "ROM ONLY",
	0x01: "MBC1",
	0x02: "MBC1+RAM",
	0x03: "MBC1+RAM+BATTERY",
	0x05: "MBC2",
	0x06: "MBC2+BATTERY",
	0x08: "ROM+RAM",
	0x09: "ROM+RAM+BATTERY",
	0x0B: "MMM01",
	0x0C: "MMM01+RAM",
	0x0D: "MMM01+RAM+BATTERY",
	0x0F: "MBC3+TIMER+BATTERY",
	0x10: "MBC3+TIMER+RAM+BATTERY",
	0x11: "MBC3",
	0x12: "MBC3+RAM",
	0x13: "MBC3+RAM+BATTERY",
	0x19: "MBC5",
	0x1A: "MBC5+RAM",
	0x1B: "MBC5+RAM+BATTERY",
	0x1C: "MBC5+RUMBLE",
	0x1D: "MBC5+RUMBLE+RAM",
	0x1E: "MBC5+RUMBLE+RAM+BATTERY",
	0x20: "MBC6",
	0x22: "MBC7+SENSOR+RUMBLE+RAM+BATTERY",
	0xFC: "POCKET CAMERA",
	0xFD: "BANDAI TAMA5",
	0xFE: "HuC3",
	0xFF: "HuC1+RAM+BATTERY",
}

type tuple struct {
	name  string
	value uint16
}

func (t *tuple) String() string {
	return fmt.Sprintf("%s (%d, 16KiB banks)", t.name, t.value)
}

var RomSizeMap = map[uint8]tuple{
	0x00: {"32 KiB", 2},
	0x01: {"64 KiB", 4},
	0x02: {"128 KiB", 8},
	0x03: {"256 KiB", 16},
	0x04: {"512 KiB", 32},
	0x05: {"1 MiB", 64},
	0x06: {"2 MiB", 128},
	0x07: {"4 MiB", 256},
	0x08: {"8 MiB", 512},
	0x52: {"1.1 MiB", 72},
	0x53: {"1.2 MiB", 80},
	0x54: {"1.5 MiB", 96},
}

var RamSizeMap = map[uint8]tuple{
	0x00: {"0", 0},
	0x01: {"-", 0},
	0x02: {"8 KiB", 1},
	0x03: {"32 KiB", 4},
	0x04: {"128 KiB", 16},
	0x05: {"64 KiB", 8},
}

var DestinationCodeMap = map[uint8]string{
	0x00: "Japanese",
	0x01: "Overseas",
}

var OldLicenseeCodeMap = map[uint8]string{
	0x00: "None",
	0x01: "Nintendo",
	0x08: "Capcom",
	0x09: "Hot-B",
	0x0A: "Jaleco",
	0x0B: "Coconuts",
	0x0C: "Elite Systems",
	0x13: "Electronic Arts",
	0x18: "Hudson Soft",
	0x19: "ITC Entertainment",
	0x1A: "Yanoman",
	0x1D: "Japan Clary",
	0x1F: "Virgin",
	0x24: "PCM Complete",
	0x25: "San-X",
	0x28: "Kotobuki Systems",
	0x29: "Seta",
	0x30: "Infogrames",
	0x31: "Nintendo",
	0x32: "Bandai",
	0x33: "Use New Licensee Code",
	0x34: "Konami",
	0x35: "Hector",
	0x38: "Capcom",
	0x39: "Banpresto",
	0x3C: "Entertainment I",
	0x3E: "Gremlin Graphics",
	0x41: "Ubisoft",
	0x42: "Atlus",
	0x44: "Malibu",
	0x46: "Angel",
	0x47: "Spectrum Holoby",
	0x49: "Irem",
	0x4A: "Virgin",
	0x4D: "Malibu",
	0x4F: "U.S. Gold",
	0x50: "Absolute",
	0x51: "Acclaim",
	0x52: "Activision",
	0x53: "American Sammy",
	0x54: "GameTek",
	0x55: "Park Place",
	0x56: "LJN",
	0x57: "Matchbox",
	0x59: "Milton Bradley",
	0x5A: "Mindscape",
	0x5B: "Romstar",
	0x5C: "Naxat Soft",
	0x5D: "Tradewest",
	0x60: "Titus",
	0x61: "Virgin",
	0x67: "Ocean",
	0x69: "Electronic Arts",
	0x6E: "Elite Systems",
	0x6F: "Electro Brain",
	0x70: "Infogrames",
	0x71: "Interplay",
	0x72: "Broderbund",
	0x73: "Sculptered Soft",
	0x75: "The Sales Curve",
	0x78: "THQ",
	0x79: "Accolade",
	0x7A: "Triffix Entertainment",
	0x7C: "Microprose",
	0x7F: "Kemco",
	0x80: "Misawa Entertainment",
	0x83: "Lozc",
	0x86: "Tokuma Shoten Intermedia",
	0x8B: "Bullet-Proof Software",
	0x8C: "Vic Tokai",
	0x8E: "Ape",
	0x8F: "I'Max",
	0x91: "Chun Soft",
	0x92: "Video System",
	0x93: "Tsuburava",
	0x95: "Varie",
	0x96: "Yonezawa/S'Pal",
	0x97: "Kaneko",
	0x99: "Arc",
	0x9A: "Nihon Bussan",
	0x9B: "Tecmo",
	0x9C: "Imagineer",
	0x9D: "Banpresto",
	0x9F: "Nova",
	0xA1: "Hori Electric",
	0xA2: "Bandai",
	0xA4: "Konami",
	0xA6: "Kawada",
	0xA7: "Takara",
	0xA9: "Technos Japan",
	0xAA: "Broderbund",
	0xAC: "Toei Animation",
	0xAD: "Toho",
	0xAF: "Namco",
	0xB0: "Acclaim",
	0xB1: "Ascii or Nexoft",
	0xB2: "Bandai",
	0xB4: "Enix",
	0xB6: "HAL",
	0xB7: "SNK",
	0xB9: "Pony Canyon",
	0xBA: "Culture Brain",
	0xBB: "Sunsoft",
	0xBD: "Sony Imagesoft",
	0xBF: "Sammy",
	0xC0: "Taito",
	0xC2: "Kemco",
	0xC3: "Square",
	0xC4: "Tokuma Shoten Intermedia",
	0xC5: "Data East",
	0xC6: "Tonkin House",
	0xC8: "Koei",
	0xC9: "UFL",
	0xCA: "Ultra",
	0xCB: "Vap",
	0xCC: "Use",
	0xCD: "Meldac",
	0xCE: "Pony Canyon or",
	0xCF: "Angel",
	0xD0: "Taito",
	0xD1: "Sofel",
	0xD2: "Quest",
	0xD3: "Sigma Enterprises",
	0xD4: "Ask Kodansha",
	0xD6: "Naxat Soft",
	0xD7: "Copya Systems",
	0xD9: "Banpresto",
	0xDA: "Tomy",
	0xDB: "LJN",
	0xDD: "NCS",
	0xDE: "Human",
	0xDF: "Altron",
	0xE0: "Jaleco",
	0xE1: "Towachiki",
	0xE2: "Uutaka",
	0xE3: "Varie",
	0xE5: "Epoch",
	0xE7: "Athena",
	0xE8: "Asmik",
	0xE9: "Natsume",
	0xEA: "King Records",
	0xEB: "Atlus",
	0xEC: "Epic/Sony Records",
	0xEE: "IGS",
	0xF0: "A-Wave",
	0xF3: "Extreme Entertainment",
	0xFF: "LJN",
}

func LoadSRAM(romName string, rambanks *[16][RAM_BANK_SIZE]uint8, ramBankCount uint16) {
	saveName := romName + ".sav"
	file, err := os.Open(saveName)
	if err != nil {
		logger.Info("No save file found")
		return
	}
	defer file.Close()

	for i := uint16(0); i < ramBankCount; i++ {
		if _, err := file.Read(rambanks[i][:]); err != nil {
			logger.Errorf("Error reading save file: %v", err)
		}
	}
	absPath, err := filepath.Abs(file.Name())
	if err != nil {
		logger.Errorf("Error getting absolute path: %v", err)
	}
	logger.Infof("Loaded %d bytes from %s", ramBankCount*RAM_BANK_SIZE, absPath)
}

func SaveSRAM(romName string, rambanks *[16][RAM_BANK_SIZE]uint8, ramBankCount uint16) {
	saveName := romName + ".sav"
	file, err := os.Create(saveName)
	if err != nil {
		logger.Errorf("Error creating save file: %v", err)
	}
	defer file.Close()

	for i := uint16(0); i < ramBankCount; i++ {
		if _, err := file.Write(rambanks[i][:]); err != nil {
			logger.Errorf("Error writing save file: %v", err)
		}
	}
	absPath, err := filepath.Abs(file.Name())
	if err != nil {
		logger.Errorf("Error getting absolute path: %v", err)
	}

	logger.Infof("Saved %d bytes to %s", ramBankCount*RAM_BANK_SIZE, absPath)
}
