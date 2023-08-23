package motherboard

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
	IO_LCDC       uint16 = 0xFF40 // LCD Control
	IO_STAT       uint16 = 0xFF41 // LCD Status
	IO_SCY        uint16 = 0xFF42 // Scroll Y
	IO_SCX        uint16 = 0xFF43 // Scroll X
	IO_LY         uint16 = 0xFF44 // LCDC Y-Coordinate
	IO_LYC        uint16 = 0xFF45 // LY Compare
	IO_DMA        uint16 = 0xFF46 // DMA Transfer and Start Address
	IO_BGP        uint16 = 0xFF47 // BG Palette Data
	IO_OBP0       uint16 = 0xFF48 // Object Palette 0 Data
	IO_OBP1       uint16 = 0xFF49 // Object Palette 1 Data
	IO_WY         uint16 = 0xFF4A // Window Y Position
	IO_WX         uint16 = 0xFF4B // Window X Position
	IO_KEY1       uint16 = 0xFF4D // CGB Mode Only - Prepare Speed Switch
	IO_VBK        uint16 = 0xFF4F // CGB Mode Only - VRAM Bank
	IO_HDMA1      uint16 = 0xFF51 // CGB Mode Only - New DMA Source, High
	IO_HDMA2      uint16 = 0xFF52 // CGB Mode Only - New DMA Source, Low
	IO_HDMA3      uint16 = 0xFF53 // CGB Mode Only - New DMA Destination, High
	IO_HDMA4      uint16 = 0xFF54 // CGB Mode Only - New DMA Destination, Low
	IO_HDMA5      uint16 = 0xFF55 // CGB Mode Only - New DMA Length/Mode/Start
	IO_RP         uint16 = 0xFF56 // CGB Mode Only - Infrared Communications Port
	IO_BCPS       uint16 = 0xFF68 // CGB Mode Only - Background Color Palette Specification
	IO_BCPD       uint16 = 0xFF69 // CGB Mode Only - Background Color Palette Data
	IO_OCPS       uint16 = 0xFF6A // CGB Mode Only - Object Color Palette Specification
	IO_OCPD       uint16 = 0xFF6B // CGB Mode Only - Object Color Palette Data
	IO_OPRI       uint16 = 0xFF6C // CGB Mode Only - Object Priority
	IO_SVBK       uint16 = 0xFF70 // CGB Mode Only - WRAM Bank
	IO_PCM12      uint16 = 0xFF76 // CGB Mode Only - PCM Channel 1&2 Control
	IO_PCM34      uint16 = 0xFF77 // CGB Mode Only - PCM Channel 3&4 Control
	IO_IE         uint16 = 0xFFFF // Interrupt Enable

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

)

type OpCode uint16                                        // 16-bit opcodes
type OpCycles uint8                                       // Number of cycles an operation takes
type OpLogic func(mb *Motherboard, value uint16) OpCycles // Operation logic
type OpCodeMap map[OpCode]OpLogic                         // Map of opcodes to their logic
