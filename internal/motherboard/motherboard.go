package motherboard

import (
	"fmt"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
)

var logger = internal.Logger

type Breakpoints struct {
	Enabled     bool     // Breakpoints enabled
	Addrs       []uint16 // Breakpoints addresses
	LastElement uint16   // Last element of breakpoints
}

type Motherboard struct {
	Cpu           *CPU                 // CPU
	Cartridge     *cartridge.Cartridge // Cartridge
	Memory        *Memory              // Internal RAM
	BootRom       *BootRom             // Boot ROM
	Timer         *Timer               // Timer
	Lcd           *LCD                 // LCD
	Input         *Input               // Input
	Cgb           bool                 // Color Gameboy
	CpuFreq       uint32               // CPU frequency
	Randomize     bool                 // Randomize RAM on startup
	BGPalette     *cgbPalette          // Background palette
	SpritePalette *cgbPalette          // Sprite palette

	hdmaActive  bool  // HDMA active
	hdmaLength  uint8 // HDMA length
	doubleSpeed bool  // Double speed mode

	// debugging
	Decouple     bool         // Decouple Motherboard from other components, and all calls to read/write memory will be mocked
	Breakpoints  *Breakpoints // Breakpoints
	PanicOnStuck bool         // Panic when CPU is stuck
	GuiPause     bool         // Pause GUI
}

type MotherboardParams struct {
	Filename     *pathlib.Path
	Randomize    bool
	ForceCgb     bool
	ForceDmg     bool
	Breakpoints  []uint16
	Decouple     bool
	PanicOnStuck bool
}

func NewMotherboard(params *MotherboardParams) *Motherboard {

	var cart *cartridge.Cartridge = cartridge.NewCartridge(params.Filename)

	var bp *Breakpoints
	if len(params.Breakpoints) > 0 {
		logger.Debug("Breakpoints enabled")

		last_element := params.Breakpoints[len(params.Breakpoints)-1]

		bp = &Breakpoints{
			Enabled:     true,
			Addrs:       params.Breakpoints,
			LastElement: last_element,
		}
	} else {
		bp = &Breakpoints{
			Enabled: false,
		}
	}

	mb := &Motherboard{
		Cartridge:     cart,
		Randomize:     params.Randomize,
		Decouple:      params.Decouple,
		Timer:         NewTimer(),
		Breakpoints:   bp,
		PanicOnStuck:  params.PanicOnStuck,
		BGPalette:     NewPalette(),
		SpritePalette: NewPalette(),
		Input:         NewInput(),
	}

	mb.Cgb = mb.Cartridge.CgbModeEnabled() || params.ForceCgb

	if mb.Cgb && params.ForceDmg {
		mb.Cgb = false
	}

	mb.CpuFreq = internal.DMG_CLOCK_SPEED

	if mb.Cgb {
		mb.CpuFreq = internal.CGB_CLOCK_SPEED
	}

	mb.Cpu = NewCpu(mb)
	mb.Memory = NewInternalRAM(mb.Cgb, params.Randomize)
	mb.Lcd = NewLCD(mb)
	mb.BootRom = NewBootRom(mb.Cgb)
	mb.BootRom.Enable()
	// mb.BootRom.Disable()

	if !mb.BootRomEnabled() {
		logger.Info("Boot ROM not enabled. Jumping to 0x100")
		mb.Cpu.Registers.PC = ROM_START_ADDR
	} else {
		logger.Info("Boot ROM enabled. Jumping to 0x0")
		mb.Cpu.Registers.PC = BOOTROM_START_ADDR
	}

	return mb
}

func (m *Motherboard) Reset() {
	m.Cpu.Reset()
	m.Memory.Reset()
	m.Lcd.Reset()
	m.BootRom.Enable()
	// m.BootRom.Disable()
	m.Timer.Reset()

	if !m.BootRomEnabled() {
		logger.Info("Boot ROM not enabled. Jumping to 0x100")
		m.Cpu.Registers.PC = ROM_START_ADDR
	} else {
		logger.Info("Boot ROM enabled. Jumping to 0x0")
		m.Cpu.Registers.PC = BOOTROM_START_ADDR
	}
}

func (m *Motherboard) BootRomEnabled() bool {
	return m.BootRom.IsEnabled
}

func (m *Motherboard) Tick() (bool, OpCycles) {
	defer func() {
		if r := recover(); r != nil {
			df := internal.StateDumpFile()
			logger.SetOutput(df)
			// dump CPU State
			fmt.Fprintln(df, "-----CRASH DETECTED-----")
			fmt.Fprintln(df, "----- CPU STATE -----")
			m.Cpu.DumpState(df)
			fmt.Fprintln(df, "----- MEMORY STATE -----")
			m.Memory.DumpState(df)
			fmt.Fprintln(df, "-----END CRASH-----")
			fmt.Println("Crash detected.. check log file for more details")
			logger.Panic()
		}
	}()
	var cycles OpCycles = 4

	if m.Cpu.Stopped || m.Cpu.IsStuck || m.GuiPause {
		return false, cycles
	}

	if !m.Cpu.Halted {
		cycles = m.Cpu.Tick()
	}

	m.Cpu.Mb.Timer.Tick(cycles, m.Cpu)
	m.Lcd.Tick(cycles)

	cycles += m.Cpu.handleInterrupts()
	return true, cycles
}

func (m *Motherboard) ButtonEvent(key Key) {
	result := m.Input.KeyEvent(key)
	if result != 0 {
		m.Cpu.SetInterruptFlag(INTR_HIGHTOLOW)
	}
}

func (m *Motherboard) GetItem(addr uint16) uint8 {
	if m.Decouple {
		logger.Warn("Decoupled Motherboard from other components. Memory read is mocked")
		return 0xDA // mock return value
	}

	// debugging
	switch {
	/*
	*
	* READ: ROM BANK 0
	*
	 */
	case addr < 0x4000: // ROM bank 0
		if m.BootRomEnabled() && (addr < 0x100 || (m.Cgb && 0x200 <= addr && addr < 0x900)) {
			return m.BootRom.GetItem(addr)
		} else {
			return m.Cartridge.CartType.GetItem(addr)
		}

	/*
	*
	* READ: SWITCHABLE ROM BANK
	*
	 */
	case 0x4000 <= addr && addr < 0x8000: // Switchable ROM bank
		return m.Cartridge.CartType.GetItem(addr)

	/*
	*
	* READ: VIDEO RAM
	*
	 */
	case 0x8000 <= addr && addr < 0xA000: // 8K Video RAM
		if m.Cgb {
			activeBank := m.Memory.ActiveVramBank()
			// return m.Memory.GetItemVRAM(activeBank, addr-0x8000)
			return m.Memory.Vram[activeBank][addr-0x8000]
		}

		return m.Memory.Vram[0][addr-0x8000]

	/*
	*
	* READ: EXTERNAL RAM
	*
	 */
	case 0xA000 <= addr && addr < 0xC000: // 8K External RAM (Cartridge)
		return m.Cartridge.CartType.GetItem(addr)

	/*
	*
	* READ: WORK RAM BANK 0
	*
	 */
	case 0xC000 <= addr && addr < 0xD000: // 4K Work RAM bank 0
		return m.Memory.Wram[0][addr-0xC000]

	/*
	*
	* READ: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= addr && addr < 0xE000:
		if m.Cgb {
			bank := m.Memory.ActiveWramBank()
			return m.Memory.Wram[bank][addr-0xD000]
		}
		return m.Memory.Wram[1][addr-0xD000]

	/*
	*
	* READ: ECHO OF 8K INTERNAL RAM
	*
	 */
	case 0xE000 <= addr && addr < 0xFE00:
		addr = addr - 0x2000 - 0xC000
		if addr >= 0x1000 {
			addr -= 0x1000
			if m.Cgb {
				bank := m.Memory.ActiveWramBank()
				return m.Memory.Wram[bank][addr]
			}
			return m.Memory.Wram[1][addr]
		}
		return m.Memory.Wram[0][addr]

	/*
	*
	* READ: SPRITE ATTRIBUTE TABLE (OAM)
	*
	 */
	case 0xFE00 <= addr && addr < 0xFEA0:
		return m.Memory.Oam[addr-0xFE00]

	/*
	*
	* READ: NOT USABLE
	*
	 */
	case 0xFEA0 <= addr && addr < 0xFF00:

	/*
	*
	* READ: I/O REGISTERS
	*
	 */
	case 0xFF00 <= addr && addr < 0xFF80:

		switch addr {

		case 0xFF00: /* P1 */
			return m.Memory.IO[IO_P1_JOYP-IO_START_ADDR]

		case 0xFF04: /* DIV */
			return uint8(m.Timer.DIV)

		case 0xFF05: /* TIMA */
			return uint8(m.Timer.TIMA)

		case 0xFF06: /* TMA */
			return uint8(m.Timer.TMA)

		case 0xFF07: /* TAC */
			return uint8(m.Timer.TAC)

		case 0xFF0F: /* IF */
			return m.Cpu.Interrupts.IF | 0xE0

		case 0xFF4D: /* KEY1 */
			// TODO: implement double speed mode
			return 0xFF

		case 0xFF50: /* Disable Boot ROM */
			return 0xFF
		case 0xFF68: /* BG Palette Index */
			if m.Cgb {
				return m.BGPalette.readIndex()
			}
			return 0x00

		case 0xFF69: /* BG Palette Data */
			if m.Cgb {
				return m.BGPalette.read()
			}
			return 0x00

		case 0xFF6A: /* Sprite Palette Index */
			if m.Cgb {
				return m.SpritePalette.readIndex()
			}
			return 0x00

		case 0xFF6B: /* Sprite Palette Data */
			if m.Cgb {
				return m.SpritePalette.read()
			}
			return 0x00

		default:
			return m.Memory.IO[addr-IO_START_ADDR]
		}

	/*
	*
	* READ: HIGH RAM
	*
	 */
	case 0xFF80 <= addr && addr < 0xFFFF:
		return m.Memory.Hram[addr-0xFF80]

	/*
	*
	* READ: INTERRUPT ENABLE REGISTER
	*
	 */
	case addr == 0xFFFF:
		return m.Cpu.Interrupts.IE

	default:
		logger.Panicf("Memory read error! Can't read from %#x\n", addr)
	}

	return 0xFF
}

func (m *Motherboard) SetItem(addr uint16, value uint16) {
	if value >= 0x100 {
		logger.Fatalf("Memory write error! Can't write %#x to %#x\n", value, addr)
	}

	if m.Decouple {
		logger.Warn("Decoupled Motherboard from other components. Memory write is mocked")
		return
	}
	v := uint8(value)

	switch {
	/*
	*
	* WRITE: ROM BANK 0
	*
	 */
	case addr < 0x4000:
		if m.BootRomEnabled() && (addr < 0x100 || (m.Cgb && 0x200 <= addr && addr < 0x900)) {
			logger.Errorf("Can't write to ROM bank 0 when boot ROM is enabled")
			return
		}

		m.Cartridge.CartType.SetItem(addr, v)

	/*
	*
	* WRITE: SWITCHABLE ROM BANK
	*
	 */
	case 0x4000 <= addr && addr < 0x8000:
		m.Cartridge.CartType.SetItem(addr, v)

	/*
	*
	* WRITE: VIDEO RAM
	*
	 */
	case 0x8000 <= addr && addr < 0xA000:
		if m.Cgb {
			bank := m.Memory.ActiveVramBank()
			m.Memory.Vram[bank][addr-0x8000] = v
		}
		m.Memory.Vram[0][addr-0x8000] = v

	/*
	*
	* WRITE: EXTERNAL RAM
	*
	 */
	case 0xA000 <= addr && addr < 0xC000:
		m.Cartridge.CartType.SetItem(addr, v)

	/*
	*
	* WRITE: WORK RAM BANK 0
	*
	 */
	case 0xC000 <= addr && addr < 0xD000:
		m.Memory.Wram[0][addr-0xC000] = v

	/*
	*
	* WRITE: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= addr && addr < 0xE000:

		// check if CGB mode
		if m.Cgb {
			// check what bank to read from
			bank := m.Memory.ActiveWramBank()
			m.Memory.Wram[bank][addr-0xD000] = v
			break
		}
		m.Memory.Wram[1][addr-0xD000] = v

	/*
	*
	* WRITE: ECHO OF 8K INTERNAL RAM
	*
	 */
	case 0xE000 <= addr && addr < 0xF000:
		m.Memory.Wram[0][addr-0x2000-0xC000] = v

	case 0xF000 <= addr && addr < 0xFE00:
		// fmt.Printf("ECHO OF 8K INTERNAL RAM: %#x, %d\n", addr, addr-0x2000-0xD000)
		m.Memory.Wram[1][addr-0x2000-0xD000] = v

	/*
	*
	* WRITE: SPRITE ATTRIBUTE TABLE (OAM)
	*
	 */
	case 0xFE00 <= addr && addr < 0xFEA0:
		m.Memory.Oam[addr-0xFE00] = v
	/*
	*
	* WRITE: NOT USABLE
	*
	 */
	case 0xFEA0 <= addr && addr < 0xFF00:

	/*
	*
	* WRITE: I/O REGISTERS
	*
	 */
	case 0xFF00 <= addr && addr < 0xFF80:

		switch addr {
		case 0xFF00: /* P1 */
			m.Memory.IO[IO_P1_JOYP-IO_START_ADDR] = m.Input.Pull(v)

		case 0xFF04: /* DIV */
			m.Timer.TimaCounter = 0
			m.Timer.DivCounter = 0
			m.Timer.DIV = 0
			return

		case 0xFF05: /* TIMA */
			m.Timer.TIMA = uint32(v)
			return

		case 0xFF06: /* TMA */
			m.Timer.TMA = uint32(v)
			return

		case 0xFF07: /* TAC */
			currentFreq := m.Timer.TAC & 0x03
			m.Timer.TAC = uint32(v) | 0xF8
			newFreq := m.Timer.TAC & 0x03
			if currentFreq != newFreq {
				m.Timer.TimaCounter = 0
			}
			return

		case 0xFF0F: /* IF */
			m.Cpu.Interrupts.IF = v
			return

		case 0xFF41: /* STAT */
			// do not set bits 0-1, they are read_only bits, bit 7 always reads 1
			stat := (m.Memory.IO[IO_STAT-IO_START_ADDR] & 0x83) | (v & 0xFC)
			m.Memory.IO[IO_STAT-IO_START_ADDR] = stat

		case 0xFF44: /* LY */
			m.Memory.IO[IO_LY-IO_START_ADDR] = 0

		case 0xFF46: /* DMA */
			m.doDMATransfer(v)

		case 0xFF4D: /* KEY1 */
			//TODO: implement double speed mode

		case 0xFF4F: /* VBK */
			if m.Cgb && !m.hdmaActive {
				m.Memory.IO[IO_VBK-IO_START_ADDR] = v & 0x01
			}

		case 0xFF50: /* Disable Boot ROM */
			if !m.BootRomEnabled() {
				logger.Warnf("Writing to 0xFF50 when boot ROM is disabled")
			}

			if m.BootRomEnabled() {
				logger.Debugf("CGB: %t, Value: %#x", m.Cgb, v)
				if m.Cgb && v == 0x11 || !m.Cgb && v == 0x1 {

					logger.Warnf("Disabling boot rom")
					m.BootRom.Disable()
					m.Cpu.Registers.PC = ROM_START_ADDR - 2 // PC will be incremented by 2
				}
			}
			return

		case 0xFF55: /* HDMA5 */
			if m.Cgb {
				m.doNewDMATransfer(v)
			}

		case 0xFF68: /* BG Palette Index */
			if m.Cgb {
				m.BGPalette.updateIndex(v)
			}
			return

		case 0xFF69: /* BG Palette Data */
			if m.Cgb {
				m.BGPalette.write(v)
			}

			return

		case 0xFF6A: /* Sprite Palette Index */
			if m.Cgb {
				m.SpritePalette.updateIndex(v)
			}
			return

		case 0xFF6B: /* Sprite Palette Data */
			if m.Cgb {
				m.SpritePalette.write(v)
			}
			return

		case 0xFF70: /* WRAM Bank */
			if m.Cgb {
				m.Memory.IO[IO_SVBK-IO_START_ADDR] = v & 0x07
			}

		default:
			m.Memory.IO[addr-IO_START_ADDR] = v
		}

		/// prints serial output to terminal ///
		// if v == 0x81 && addr == IO_SC {
		// 	fmt.Printf("%c", m.Memory.IO[IO_SB-IO_START_ADDR])
		// }
		////////////////////////////////////

	/*
	*
	* WRITE: HIGH RAM
	*
	 */
	case 0xFF80 <= addr && addr < 0xFFFF:
		m.Memory.Hram[addr-0xFF80] = v

	/*
	*
	* WRITE: INTERRUPT ENABLE REGISTER
	*
	 */
	case addr == IE:
		m.Cpu.Interrupts.IE = v
	default:
		internal.Logger.Panicf("Memory write error! Can't write `%#x` to `%#x`\n", value, addr)
	}

}

func (m *Motherboard) DoHDMATransfer() {
	if !m.hdmaActive {
		return
	}
	m.performNewDMATransfer(0x10)
	if m.hdmaLength > 0 {
		m.hdmaLength--
		m.Memory.IO[IO_HDMA5-IO_START_ADDR] = m.hdmaLength
	} else {
		m.hdmaActive = false
		m.Memory.IO[IO_HDMA5-IO_START_ADDR] = 0xFF
	}
}

func (m *Motherboard) performNewDMATransfer(length uint16) {

	// load the source and destination from RAM
	source := (uint16(m.Memory.IO[IO_HDMA1-IO_START_ADDR])<<8 | uint16(m.Memory.IO[IO_HDMA2-IO_START_ADDR])) & 0xFFF0
	destination := (uint16(m.Memory.IO[IO_HDMA3-IO_START_ADDR])<<8 | uint16(m.Memory.IO[IO_HDMA4-IO_START_ADDR])) & 0x1FF0
	destination += 0x8000

	// copy the data
	for i := uint16(0); i < length; i++ {
		m.SetItem(destination, uint16(m.GetItem(source)))
		source++
		destination++
	}

	// update the source and destination in RAM
	m.Memory.IO[IO_HDMA1-IO_START_ADDR] = uint8(source >> 8)
	m.Memory.IO[IO_HDMA2-IO_START_ADDR] = uint8(source & 0xFF)
	m.Memory.IO[IO_HDMA3-IO_START_ADDR] = uint8(destination >> 8)
	m.Memory.IO[IO_HDMA4-IO_START_ADDR] = uint8(destination & 0xF0)
}

// Perform a DMA transfer.
func (m *Motherboard) doDMATransfer(value byte) {
	address := uint16(value) << 8 // (data * 100)

	var i uint16
	for i = 0; i < 0xA0; i++ {
		m.SetItem(0xFE00+i, uint16(m.GetItem(address+i)))
	}
}

// Start a CGB DMA transfer.
func (m *Motherboard) doNewDMATransfer(value byte) {
	// if m.hdmaActive && bits.Val(value, 7) == 0 {
	if m.hdmaActive && !internal.IsBitSet(value, 7) {
		// Abort a HDMA transfer
		m.hdmaActive = false
		m.Memory.Hram[0x55] |= 0x80 // Set bit 7
		return
	}

	length := ((uint16(value) & 0x7F) + 1) * 0x10

	// The 7th bit is DMA mode
	if value>>7 == 0 {
		// Mode 0, general purpose DMA
		m.performNewDMATransfer(length)
		m.Memory.Hram[0x55] = 0xFF
	} else {
		// Mode 1, H-Blank DMA
		m.hdmaLength = byte(value)
		m.hdmaActive = true
	}
}
