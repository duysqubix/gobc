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
	Cpu         *CPU                 // CPU
	Cartridge   *cartridge.Cartridge // Cartridge
	Ram         *InternalRAM         // Internal RAM
	BootRom     *BootRom             // Boot ROM
	Timer       *Timer               // Timer
	Cgb         bool                 // Color Gameboy
	CpuFreq     uint32               // CPU frequency
	Randomize   bool                 // Randomize RAM on startup
	Decouple    bool                 // Decouple Motherboard from other components, and all calls to read/write memory will be mocked
	Breakpoints *Breakpoints         // Breakpoints
}

type MotherboardParams struct {
	Filename    *pathlib.Path
	Randomize   bool
	ForceCgb    bool
	Breakpoints []uint16
	Decouple    bool
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
		Cartridge:   cart,
		Randomize:   params.Randomize,
		Decouple:    params.Decouple,
		Timer:       NewTimer(),
		Breakpoints: bp,
	}

	mb.Cgb = mb.Cartridge.CgbModeEnabled() || params.ForceCgb
	mb.CpuFreq = internal.DMG_CLOCK_SPEED

	if mb.Cgb {
		mb.CpuFreq = internal.CGB_CLOCK_SPEED
	}

	mb.Cpu = NewCpu(mb)
	mb.Ram = NewInternalRAM(mb.Cgb, params.Randomize)
	// mb.BootRom = NewBootRom(mb.Cartridge.CgbModeEnabled())

	if !mb.BootRomEnabled() {
		logger.Debugf("Boot ROM disabled")
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
			m.Ram.DumpState(df)
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

	m.Cpu.Mb.Timer.Tick(cycles, m.Cpu)

	cycles += m.handleInterrupts()

	return true, cycles
}

func (m *Motherboard) handleInterrupts() OpCycles {

	if m.Cpu.Interrupts.InterruptsEnabling {
		m.Cpu.Interrupts.InterruptsOn = true
		m.Cpu.Interrupts.InterruptsEnabling = false
		return 0
	}

	if !m.Cpu.Interrupts.InterruptsOn && !m.Cpu.Halted {
		return 0
	}

	req := m.Cpu.Interrupts.IF | 0xE0
	enabled := m.Cpu.Interrupts.IE

	if req > 0 {
		var i uint8
		for i = 0; i < 5; i++ {
			if internal.IsBitSet(req, i) && internal.IsBitSet(enabled, i) {
				m.Cpu.ServiceInterrupt(i)
				return 20
			}
		}
	}

	return 0
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
			activeBank := m.Ram.ActiveVramBank()
			// return m.Ram.GetItemVRAM(activeBank, addr-0x8000)
			return m.Ram.Vram[activeBank][addr-0x8000]
		}

		return m.Ram.Vram[0][addr-0x8000]

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
		return m.Ram.Wram[0][addr-0xC000]

	/*
	*
	* READ: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= addr && addr < 0xE000:
		if m.Cgb {
			bank := m.Ram.ActiveWramBank()
			return m.Ram.Wram[bank][addr-0xD000]
		}
		return m.Ram.Wram[1][addr-0xD000]

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
				bank := m.Ram.ActiveWramBank()
				return m.Ram.Wram[bank][addr]
			}
			return m.Ram.Wram[1][addr]
		}
		return m.Ram.Wram[0][addr]

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
			return m.Ram.IO[addr-0xFF00]
		}

	/*
	*
	* READ: HIGH RAM
	*
	 */
	case 0xFF80 <= addr && addr < 0xFFFF:
		return m.Ram.Hram[addr-0xFF80]

	/*
	*
	* READ: INTERRUPT ENABLE REGISTER
	*
	 */
	case addr == 0xFFFF:
		logger.Debugf("Reading from %#x on Interrupt Enable Register\n", addr)
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
		m.Cartridge.CartType.SetItem(addr, v)

	/*
	*
	* WRITE: SWITCHABLE ROM BANK
	*
	 */
	case 0x4000 <= addr && addr < 0x8000:
		m.Cartridge.CartType.SetItem(addr-0x4000, v)

	/*
	*
	* WRITE: VIDEO RAM
	*
	 */
	case 0x8000 <= addr && addr < 0xA000:
		if m.Cgb {
			bank := m.Ram.ActiveVramBank()
			m.Ram.Vram[bank][addr-0x8000] = v
		}
		m.Ram.Vram[0][addr-0x8000] = v

	/*
	*
	* WRITE: EXTERNAL RAM
	*
	 */
	case 0xA000 <= addr && addr < 0xC000:
		m.Cartridge.CartType.SetItem(addr-0xA000, v)

	/*
	*
	* WRITE: WORK RAM BANK 0
	*
	 */
	case 0xC000 <= addr && addr < 0xD000:
		m.Ram.Wram[0][addr-0xC000] = v

	/*
	*
	* WRITE: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= addr && addr < 0xE000:

		// check if CGB mode
		if m.Cgb {
			// check what bank to read from
			bank := m.Ram.ActiveWramBank()
			m.Ram.Wram[bank][addr-0xD000] = v
			break
		}
		m.Ram.Wram[1][addr-0xD000] = v

	/*
	*
	* WRITE: ECHO OF 8K INTERNAL RAM
	*
	 */
	case 0xE000 <= addr && addr < 0xFE00:
		addr = addr - 0x2000 - 0xC000
		m.Ram.Wram[0][addr] = v

	/*
	*
	* WRITE: SPRITE ATTRIBUTE TABLE (OAM)
	*
	 */
	case 0xFE00 <= addr && addr < 0xFEA0:
		m.Ram.Oam[addr-0xFE00] = v
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

		default:
			m.Ram.IO[addr-0xFF00] = v
		}

		/// prints serial output to terminal ///
		if v == 0x81 && addr == IO_SC {
			fmt.Printf("%c", m.Ram.IO[IO_SB-IO_START_ADDR])
		}
		////////////////////////////////////

	/*
	*
	* WRITE: HIGH RAM
	*
	 */
	case 0xFF80 <= addr && addr < 0xFFFF:
		m.Ram.Hram[addr-0xFF80] = v

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
