package motherboard

import (
	"github.com/duysqubix/gobc/internal"
)

const (
	// IO Addresses
	IO_START_ADDR uint16 = 0xFF00 // Start of IO addresses
	IO_END_ADDR   uint16 = 0xFF7F // End of IO addresses
	IO_P1_JOYP    uint16 = 0xFF00 // Joypad
	IO_SB         uint16 = 0xFF01 // Serial transfer data
	IO_SC         uint16 = 0xFF02 // Serial transfer control
	IO_DIV        uint16 = 0xFF04 // Divider Register
	IO_TIMA       uint16 = 0xFF05 // Timer counter
	IO_TMA        uint16 = 0xFF06 // Timer Modulo
	IO_TAC        uint16 = 0xFF07 // Timer Control
	IO_IF         uint16 = 0xFF0F // Interrupt Flag
	IO_NR10       uint16 = 0xFF10 // Sound Mode 1 register, Sweep register
	IO_NR11       uint16 = 0xFF11 // Sound Mode 1 register, Sound length/Wave pattern duty
	IO_NR12       uint16 = 0xFF12 // Sound Mode 1 register, Envelope
	IO_NR13       uint16 = 0xFF13 // Sound Mode 1 register, Frequency lo
	IO_NR14       uint16 = 0xFF14 // Sound Mode 1 register, Frequency hi
	IO_NR21       uint16 = 0xFF16 // Sound Mode 2 register, Sound length/Wave pattern duty
	IO_NR22       uint16 = 0xFF17 // Sound Mode 2 register, Envelope
	IO_NR23       uint16 = 0xFF18 // Sound Mode 2 register, Frequency lo
	IO_NR24       uint16 = 0xFF19 // Sound Mode 2 register, Frequency hi
	IO_NR30       uint16 = 0xFF1A // Sound Mode 3 register, Sound on/off
	IO_NR31       uint16 = 0xFF1B // Sound Mode 3 register, Sound length
	IO_NR32       uint16 = 0xFF1C // Sound Mode 3 register, Select output level
	IO_NR33       uint16 = 0xFF1D // Sound Mode 3 register, Frequency lo
	IO_NR34       uint16 = 0xFF1E // Sound Mode 3 register, Frequency hi
	IO_NR41       uint16 = 0xFF20 // Sound Mode 4 register, Sound length
	IO_NR42       uint16 = 0xFF21 // Sound Mode 4 register, Envelope
	IO_NR43       uint16 = 0xFF22 // Sound Mode 4 register, Polynomial counter
	IO_NR44       uint16 = 0xFF23 // Sound Mode 4 register, Counter/consecutive; Inital
	IO_NR50       uint16 = 0xFF24 // Channel control / ON-OFF / Volume
	IO_NR51       uint16 = 0xFF25 // Selection of Sound output terminal
	IO_NR52       uint16 = 0xFF26 // Sound on/off
	IO_WAVE_RAM1  uint16 = 0xFF30 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM2  uint16 = 0xFF31 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM3  uint16 = 0xFF32 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM4  uint16 = 0xFF33 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM5  uint16 = 0xFF34 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM6  uint16 = 0xFF35 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM7  uint16 = 0xFF36 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM8  uint16 = 0xFF37 // Waveform storage for arbitrary sound data
	IO_WAVE_RAM9  uint16 = 0xFF38 // Waveform storage for arbitrary sound data
	IO_WAVE_RAMA  uint16 = 0xFF39 // Waveform storage for arbitrary sound data
	IO_WAVE_RAMB  uint16 = 0xFF3A // Waveform storage for arbitrary sound data
	IO_WAVE_RAMC  uint16 = 0xFF3B // Waveform storage for arbitrary sound data
	IO_WAVE_RAMD  uint16 = 0xFF3C // Waveform storage for arbitrary sound data
	IO_WAVE_RAME  uint16 = 0xFF3D // Waveform storage for arbitrary sound data
	IO_WAVE_RAMF  uint16 = 0xFF3E // Waveform storage for arbitrary sound data

	IO_LCDC  uint16 = 0xFF40 // LCD Control
	IO_STAT  uint16 = 0xFF41 // LCD Status
	IO_SCY   uint16 = 0xFF42 // Scroll Y
	IO_SCX   uint16 = 0xFF43 // Scroll X
	IO_LY    uint16 = 0xFF44 // LCDC Y-Coordinate
	IO_LYC   uint16 = 0xFF45 // LY Compare
	IO_DMA   uint16 = 0xFF46 // DMA Transfer and Start Address
	IO_BGP   uint16 = 0xFF47 // BG Palette Data
	IO_OBP0  uint16 = 0xFF48 // Object Palette 0 Data
	IO_OBP1  uint16 = 0xFF49 // Object Palette 1 Data
	IO_WY    uint16 = 0xFF4A // Window Y Position
	IO_WX    uint16 = 0xFF4B // Window X Position
	IO_KEY1  uint16 = 0xFF4D // CGB Mode Only - Prepare Speed Switch
	IO_VBK   uint16 = 0xFF4F // CGB Mode Only - VRAM Bank
	IO_HDMA1 uint16 = 0xFF51 // CGB Mode Only - New DMA Source, High
	IO_HDMA2 uint16 = 0xFF52 // CGB Mode Only - New DMA Source, Low
	IO_HDMA3 uint16 = 0xFF53 // CGB Mode Only - New DMA Destination, High
	IO_HDMA4 uint16 = 0xFF54 // CGB Mode Only - New DMA Destination, Low
	IO_HDMA5 uint16 = 0xFF55 // CGB Mode Only - New DMA Length/Mode/Start
	IO_RP    uint16 = 0xFF56 // CGB Mode Only - Infrared Communications Port
	IO_BCPS  uint16 = 0xFF68 // CGB Mode Only - Background Color Palette Specification
	IO_BCPD  uint16 = 0xFF69 // CGB Mode Only - Background Color Palette Data
	IO_OCPS  uint16 = 0xFF6A // CGB Mode Only - Object Color Palette Specification
	IO_OCPD  uint16 = 0xFF6B // CGB Mode Only - Object Color Palette Data
	IO_OPRI  uint16 = 0xFF6C // CGB Mode Only - Object Priority
	IO_SVBK  uint16 = 0xFF70 // CGB Mode Only - WRAM Bank
	IO_PCM12 uint16 = 0xFF76 // CGB Mode Only - PCM Channel 1&2 Control
	IO_PCM34 uint16 = 0xFF77 // CGB Mode Only - PCM Channel 3&4 Control

	IE uint16 = 0xFFFF // Interrupt Enable

	FLAGC uint8 = 0x04 // Math operation raised carry
	FLAGH uint8 = 0x05 // Math operation raised half carry
	FLAGN uint8 = 0x06 // Math operation was a subtraction
	FLAGZ uint8 = 0x07 // Math operation result was zero

	CB_SHIFT OpCode = 0x100

	INTR_VBLANK    uint8 = 0x0 // VBlank interrupt      00000001 (bit 0)
	INTR_LCDSTAT   uint8 = 0x1 // LCD status interrupt  00000010 (bit 1)
	INTR_TIMER     uint8 = 0x2 // Timer interrupt       00000100 (bit 2)
	INTR_SERIAL    uint8 = 0x3 // Serial interrupt      00001000 (bit 3)
	INTR_HIGHTOLOW uint8 = 0x4 // Joypad interrupt      00010000 (bit 4)

	INTR_VBLANK_ADDR    uint16 = 0x0040 // VBlank interrupt Memory address
	INTR_LCDSTAT_ADDR   uint16 = 0x0048 // LCD status interrupt Memory address
	INTR_TIMER_ADDR     uint16 = 0x0050 // Timer interrupt Memory address
	INTR_SERIAL_ADDR    uint16 = 0x0058 // Serial interrupt Memory address
	INTR_HIGHTOLOW_ADDR uint16 = 0x0060 // Joypad interrupt Memory address

	LCDC_ENABLE uint8 = 7 // Bit 7 - LCD Display Enable             (0=Off, 1=On)
	LCDC_WINMAP uint8 = 6 // Bit 6 - Window Tile Map Display Select (0=9800-9BFF, 1=9C00-9FFF)
	LCDC_WINEN  uint8 = 5 // Bit 5 - Window Display Enable          (0=Off, 1=On)
	LCDC_BGMAP  uint8 = 4 // Bit 4 - BG & Window Tile Data Select   (0=8800-97FF, 1=8000-8FFF)
	LCDC_BGWIN  uint8 = 3 // Bit 3 - BG Tile Map Display Select     (0=9800-9BFF, 1=9C00-9FFF)
	LCDC_OBJSZ  uint8 = 2 // Bit 2 - OBJ (Sprite) Size              (0=8x8, 1=8x16)
	LCDC_OBJEN  uint8 = 1 // Bit 1 - OBJ (Sprite) Display Enable    (0=Off, 1=On)
	LCDC_BGEN   uint8 = 0 // Bit 0 - BG Display (for CGB see below) (0=Off, 1=On); CGB Object Priority Bit (0=OBJ Above BG, 1=OBJ Behind BG color 1-3)

	STAT_LYCINT uint8 = 6 // Bit 6 - LYC=LY Coincidence Interrupt (1=Enable) (Read/Write)
	STAT_OAMINT uint8 = 5 // Bit 5 - Mode 2 OAM Interrupt         (1=Enable) (Read/Write)
	STAT_VBLINT uint8 = 4 // Bit 4 - Mode 1 V-Blank Interrupt     (1=Enable) (Read/Write)
	STAT_HBLINT uint8 = 3 // Bit 3 - Mode 0 H-Blank Interrupt     (1=Enable) (Read/Write)
	STAT_LYC    uint8 = 2 // Bit 2 - LYC=LY Coincidence Flag  (0:LYC<>LY, 1:LYC=LY) (Read Only)
	STAT_MODE1  uint8 = 1 // Bit 1-0 - Mode Flag       (Mode 0-3, see below) (Read Only)
	STAT_MODE0  uint8 = 0 // Bit 1-0 - Mode Flag       (Mode 0-3, see below) (Read Only)

	STAT_MODE_HBLANK uint8 = 0x00 // Mode 0: During H-Blank
	STAT_MODE_VBLANK uint8 = 0x01 // Mode 1: During V-Blank
	STAT_MODE_OAM    uint8 = 0x02 // Mode 2: During Searching OAM-RAM
	STAT_MODE_TRANS  uint8 = 0x03 // Mode 3: During Transfering Data to LCD Driver

	TAC_ENABLE     uint8    = 0x04 // Timer enable (0b100)
	TAC_SPEED_1024 OpCycles = 1024 // CPU_CLOCK / 1024 (0b00)
	TAC_SPEED_16   OpCycles = 16   // CPU_CLOCK / 16 (0b01)
	TAC_SPEED_64   OpCycles = 64   // CPU_CLOCK / 64 (0b10)
	TAC_SPEED_256  OpCycles = 256  // CPU_CLOCK / 256 (0b11)

	TIMER_DIV_HZ uint32 = 16384 // 16384 Hz

	BOOTROM_START_ADDR uint16 = 0x0000 // Start of CPU addresses
	ROM_START_ADDR     uint16 = 0x0100 // Start of ROM addresses
)

type OpCode uint16                                        // 16-bit opcodes
type OpCycles int64                                       // Number of cycles an operation takes
type OpLogic func(mb *Motherboard, value uint16) OpCycles // Operation logic
type OpCodeMap map[OpCode]OpLogic                         // Map of opcodes to their logic
const (
	// ButtonA is the A button on the GameBoy.
	ButtonA = 0
	// ButtonB is the B button on the GameBoy.
	ButtonB = 1
	// ButtonSelect is the select button on the GameBoy.
	ButtonSelect = 2
	// ButtonStart is the start button on the GameBoy.
	ButtonStart = 3
	// ButtonRight is the right dpad direction on the GameBoy.
	ButtonRight = 4
	// ButtonLeft is the left dpad direction on the GameBoy.
	ButtonLeft = 5
	// ButtonUp is the up dpad direction on the GameBoy.
	ButtonUp = 6
	// ButtonDown is the down dpad direction on the GameBoy.
	ButtonDown = 7
)

func MemoryMapName(addr uint16) string {
	switch {
	case addr < 0x4000:
		return "ROM Bank 0"
	case 0x4000 <= addr && addr < 0x8000:
		return "Switchable ROM Bank"
	case 0x8000 <= addr && addr < 0x9800:
		return "VRAM Tile Data"
	case 0x9800 <= addr && addr < 0xA000:
		return "VRAM Tile Maps"
	case 0xA000 <= addr && addr < 0xC000:
		return "Switchable RAM Bank"
	case 0xC000 <= addr && addr < 0xE000:
		return "Internal RAM"
	case 0xE000 <= addr && addr < 0xFE00:
		return "Echo RAM"
	case 0xFE00 <= addr && addr < 0xFEA0:
		return "OAM"
	case 0xFEA0 <= addr && addr < 0xFF00:
		return "Not Usable"
	case 0xFF00 <= addr && addr < 0xFF80:
		return "IO"
	case 0xFF80 <= addr && addr < 0xFFFF:
		return "High RAM"
	case addr == 0xFFFF:
		return "Interrupt Enable Register"
	default:
		return "Unknown"
	}
}

var LCDCBitNames = map[uint8]string{
	LCDC_ENABLE: "LCD ENABLE",
	LCDC_WINMAP: "WIN MAP ADDRESS",
	LCDC_WINEN:  "WIN ENABLE",
	LCDC_BGMAP:  "BG ADDR MODE",
	LCDC_BGWIN:  "BG MAP ADDRESS",
	LCDC_OBJSZ:  "OBJ SIZE",
	LCDC_OBJEN:  "OBJ ENABLE",
	LCDC_BGEN:   "BG ENABLE/PRIO",
}

var STATBitNames = map[uint8]string{
	STAT_LYCINT: "LYC INT",
	STAT_OAMINT: "OAM INT",
	STAT_VBLINT: "VBL INT",
	STAT_HBLINT: "HBL INT",
	STAT_LYC:    "LYC=LY",
	STAT_MODE1:  "MODE 1",
	STAT_MODE0:  "MODE 0",
}

func InterruptFlagDump(v uint8) string {
	var msg string = ""

	for i := uint8(0); i < 5; i++ {
		if internal.IsBitSet(v, i) {
			switch i {
			case INTR_VBLANK:
				msg += "VBLANK, "
			case INTR_LCDSTAT:
				msg += "LCDSTAT, "
			case INTR_TIMER:
				msg += "TIMER, "
			case INTR_SERIAL:
				msg += "SERIAL, "
			case INTR_HIGHTOLOW:
				msg += "HIGHTOLOW, "
			}
		}
	}
	return msg
}
