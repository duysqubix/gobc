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
	Cpu          *CPU                 // CPU
	Cartridge    *cartridge.Cartridge // Cartridge
	Memory       *Memory              // Internal RAM
	BootRom      *BootRom             // Boot ROM
	Timer        *Timer               // Timer
	Lcd          *LCD                 // LCD
	Cgb          bool                 // Color Gameboy
	CpuFreq      uint32               // CPU frequency
	Randomize    bool                 // Randomize RAM on startup
	Decouple     bool                 // Decouple Motherboard from other components, and all calls to read/write memory will be mocked
	Breakpoints  *Breakpoints         // Breakpoints
	PanicOnStuck bool                 // Panic when CPU is stuck
	hdmaActive   bool                 // HDMA active
	hdmaLength   uint8                // HDMA length
}

type MotherboardParams struct {
	Filename     *pathlib.Path
	Randomize    bool
	ForceCgb     bool
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
		Cartridge:    cart,
		Randomize:    params.Randomize,
		Decouple:     params.Decouple,
		Timer:        NewTimer(),
		Breakpoints:  bp,
		PanicOnStuck: params.PanicOnStuck,
	}

	mb.Cgb = mb.Cartridge.CgbModeEnabled() || params.ForceCgb

	// if mb.Cgb {
	// 	logger.Errorf("CGB mode is not implemented yet")
	// 	// os.Exit(0)
	// 	mb.Cgb = false
	// }

	mb.CpuFreq = internal.DMG_CLOCK_SPEED

	if mb.Cgb {
		mb.CpuFreq = internal.CGB_CLOCK_SPEED
	}

	mb.Cpu = NewCpu(mb)
	mb.Memory = NewInternalRAM(mb.Cgb, params.Randomize)
	mb.Lcd = NewLCD(mb)
	mb.BootRom = NewBootRom(mb.Cgb)

	if !mb.BootRomEnabled() {
		logger.Info("Boot ROM not enabled. Jumping to 0x100")
		mb.Cpu.Registers.PC = 0x100
	}

	return mb
}

func (m *Motherboard) BootRomEnabled() bool {
	return m.BootRom != nil
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

	if m.Cpu.Stopped || m.Cpu.IsStuck {
		return false, cycles
	}

	if !m.Cpu.Halted {
		cycles = m.Cpu.Tick()
	}

	m.Lcd.Tick(cycles)
	m.Cpu.Mb.Timer.Tick(cycles, m.Cpu)
	cycles += m.Cpu.handleInterrupts()

	return true, cycles
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
		default:
			return m.Memory.IO[addr-0xFF00]
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
		internal.Logger.Panicf("Memory write error! Can't write %#x to %#x\n", value, addr)
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

		if m.BootRomEnabled() &&
			addr == 0xFF50 &&
			(v == 0x1 || v == 0x11) {
			logger.Debugf("Disabling boot rom")
			m.BootRom = nil
		}

		switch addr {

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

		case 0xFF50: /* Disable Boot ROM */
			if m.BootRomEnabled() {
				logger.Debugf("Disabling boot rom")
				m.BootRom = nil
				m.Cpu.Registers.PC = 0x100
			}
			return

		default:
			m.Memory.IO[addr-0xFF00] = v
		}

		/// prints serial output to terminal ///
		if v == 0x81 && addr == IO_SC {
			fmt.Printf("%c", m.Memory.IO[IO_SB-IO_START_ADDR])
		}
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
		m.SetItem(IO_HDMA5, uint16(m.hdmaLength))
	} else {
		m.hdmaActive = false
		m.SetItem(IO_HDMA5, 0xFF)
	}
}

func (m *Motherboard) performNewDMATransfer(length uint16) {

	// load the source and destination from RAM
	src := uint16(m.GetItem(IO_HDMA1))<<8 | uint16(m.GetItem(IO_HDMA2))
	dst := uint16(m.GetItem(IO_HDMA3))<<8 | uint16(m.GetItem(IO_HDMA4))
	dst += 0x8000

	// perform the transfer
	for i := uint16(0); i < length; i++ {
		m.SetItem(dst, uint16(m.GetItem(src)))
		src++
		dst++
	}

	// update the source and destination
	m.SetItem(IO_HDMA1, src>>8)
	m.SetItem(IO_HDMA2, src&0xFF)
	m.SetItem(IO_HDMA3, dst>>8)
	m.SetItem(IO_HDMA4, dst&0xFF)
}
