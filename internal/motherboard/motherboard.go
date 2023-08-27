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
	Cgb         bool                 // Color Gameboy
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

	var cart *cartridge.Cartridge
	if params.Filename != nil {
		cart = cartridge.NewCartridge(params.Filename)
	} else {
		cart = &cartridge.Cartridge{}
	}

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
		Breakpoints: bp,
	}

	mb.Cgb = mb.Cartridge.CgbModeEnabled() || params.ForceCgb
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
	if m.Cpu.Stopped || m.Cpu.Halted || m.Cpu.IsStuck {
		return false, 0
	}

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
	cycles := m.Cpu.Tick()

	return true, cycles
}

func (m *Motherboard) GetItem(addr *uint16) uint8 {
	addr_copy := *addr

	// logger.Debugf("Reading from %#x on Motherboard\n", *addr)
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
	case *addr < 0x4000: // ROM bank 0
		logger.Debugf("Reading from %#x on ROM bank 0", *addr)
		if m.BootRomEnabled() && (*addr < 0x100 || (m.Cgb && 0x200 <= *addr && *addr < 0x900)) {
			return m.BootRom.GetItem(addr_copy)
		} else {
			return m.Cartridge.CartType.GetItem(addr_copy)
		}

	/*
	*
	* READ: SWITCHABLE ROM BANK
	*
	 */
	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank
		logger.Debugf("Reading from %#x on Switchable ROM bank", *addr)
		addr_copy -= 0x4000
		return m.Cartridge.CartType.GetItem(addr_copy)

	/*
	*
	* READ: VIDEO RAM
	*
	 */
	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM
		logger.Debugf("Reading from %#x on Video RAM", *addr)
		addr_copy -= 0x8000
		if m.Cgb {
			return m.Ram.GetItemVRAM(m.Ram.ActiveVramBank(), addr_copy)
		}

		return m.Ram.GetItemVRAM(0, addr_copy)

	/*
	*
	* READ: EXTERNAL RAM
	*
	 */
	case 0xA000 <= *addr && *addr < 0xC000: // 8K External RAM (Cartridge)
		logger.Debugf("Reading from %#x on External RAM", *addr)
		addr_copy -= 0xA000
		return m.Cartridge.CartType.GetItem(addr_copy)

	/*
	*
	* READ: WORK RAM BANK 0
	*
	 */
	case 0xC000 <= *addr && *addr < 0xD000: // 4K Work RAM bank 0
		logger.Debugf("Reading from %#x on Work RAM Bank 0", *addr)
		addr_copy -= 0xC000
		return m.Ram.GetItemWRAM(0, addr_copy)

	/*
	*
	* READ: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= *addr && *addr < 0xE000:
		addr_copy -= 0xD000
		logger.Debugf("Reading from %#x on Work RAM Bank=[%d]", *addr, m.Ram.ActiveWramBank())
		// check if CGB mode
		if m.Cgb {
			// check what bank to read from
			bank := m.Ram.ActiveWramBank()
			return m.Ram.GetItemWRAM(bank, addr_copy)
		}
		logger.Debugf("%d\n", 1)
		return m.Ram.GetItemWRAM(1, addr_copy)

	/*
	*
	* READ: ECHO OF 8K INTERNAL RAM
	*
	 */
	case 0xE000 <= *addr && *addr < 0xFE00:
		logger.Debugf("Reading from %#x on Echo of 8K Internal RAM", *addr)
		addr_copy = addr_copy - 0x2000 - 0xC000
		if addr_copy >= 0x1000 {
			addr_copy -= 0x1000
			if m.Cgb {
				bank := m.Ram.ActiveWramBank()
				return m.Ram.GetItemWRAM(bank, addr_copy)
			}
			return m.Ram.GetItemWRAM(1, addr_copy)
		}
		return m.Ram.GetItemWRAM(0, addr_copy)

	/*
	*
	* READ: SPRITE ATTRIBUTE TABLE (OAM)
	*
	 */
	case 0xFE00 <= *addr && *addr < 0xFEA0:
		logger.Debugf("Reading from %#x on Sprite Attribute Table (OAM)", *addr)

	/*
	*
	* READ: NOT USABLE
	*
	 */
	case 0xFEA0 <= *addr && *addr < 0xFF00:
		logger.Warningf("Reading from %#x on Not Usable", *addr)

	/*
	*
	* READ: I/O REGISTERS
	*
	 */
	case 0xFF00 <= *addr && *addr < 0xFF80:
		addr_copy -= 0xFF00
		logger.Debugf("Reading from %#x on IO", *addr)
		return m.Ram.GetItemIO(addr_copy)

	/*
	*
	* READ: HIGH RAM
	*
	 */
	case 0xFF80 <= *addr && *addr < 0xFFFF:
		logger.Debugf("Reading from %#x on High RAM", *addr)
		addr_copy -= 0xFF80
		return m.Ram.GetItemHRAM(addr_copy)

	/*
	*
	* READ: INTERRUPT ENABLE REGISTER
	*
	 */
	case *addr == 0xFFFF:
		logger.Debugf("Reading from %#x on Interrupt Enable Register\n", *addr)

	default:
		logger.Panicf("Memory read error! Can't read from %#x\n", *addr)
	}

	return 0xFF
}

func (m *Motherboard) SetItem(addr *uint16, value *uint16) {
	if *value >= 0x100 {
		internal.Logger.Panicf("Memory write error! Can't write %#x to %#x\n", *value, *addr)
	}

	if m.Decouple {
		logger.Warn("Decoupled Motherboard from other components. Memory write is mocked")
		return
	}
	v := uint8(*value)
	addr_copy := *addr

	switch {
	/*
	*
	* WRITE: ROM BANK 0
	*
	 */
	case *addr < 0x4000:
		logger.Debugf("Writing %#x to %#x on ROM bank 0", v, *addr)
		m.Cartridge.CartType.SetItem(addr_copy, v)

	/*
	*
	* WRITE: SWITCHABLE ROM BANK
	*
	 */
	case 0x4000 <= *addr && *addr < 0x8000:
		logger.Debugf("Writing %#x to %#x on Switchable ROM bank", v, *addr)
		addr_copy -= 0x4000
		m.Cartridge.CartType.SetItem(addr_copy, v)

	/*
	*
	* WRITE: VIDEO RAM
	*
	 */
	case 0x8000 <= *addr && *addr < 0xA000:
		addr_copy -= 0x8000
		logger.Debugf("Writing %#x to %#x on Video RAM", v, *addr)
		if m.Cgb {
			m.Ram.SetItemVRAM(m.Ram.ActiveVramBank(), addr_copy, v)
		}
		m.Ram.SetItemVRAM(0, addr_copy, v)

	/*
	*
	* WRITE: EXTERNAL RAM
	*
	 */
	case 0xA000 <= *addr && *addr < 0xC000:
		addr_copy -= 0xA000
		logger.Debugf("Writing %#x to %#x on External RAM", v, *addr)
		m.Cartridge.CartType.SetItem(addr_copy, v)

	/*
	*
	* WRITE: WORK RAM BANK 0
	*
	 */
	case 0xC000 <= *addr && *addr < 0xD000:
		addr_copy -= 0xC000
		logger.Debugf("Writing %#x to %#x on Work RAM BANK 0", v, addr_copy)

		m.Ram.SetItemWRAM(0, addr_copy, v)

	/*
	*
	* WRITE: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= *addr && *addr < 0xE000:
		addr_copy -= 0xD000
		logger.Debugf("Writing %#x to %#x on Work RAM Bank=[%d]", *addr, v, m.Ram.ActiveWramBank())

		// check if CGB mode
		if m.Cgb {
			// check what bank to read from
			bank := m.Ram.ActiveWramBank()
			m.Ram.SetItemWRAM(bank, addr_copy, v)
			break
		}
		m.Ram.SetItemWRAM(1, addr_copy, v)

	/*
	*
	* WRITE: ECHO OF 8K INTERNAL RAM
	*
	 */
	case 0xE000 <= *addr && *addr < 0xFE00:
		addr_copy = addr_copy - 0x2000 - 0xC000
		logger.Debugf("Writing %#x to %#x on Echo of 8K Internal RAM", v, *addr)
		m.Ram.SetItemWRAM(0, addr_copy, v)

	/*
	*
	* WRITE: SPRITE ATTRIBUTE TABLE (OAM)
	*
	 */
	case 0xFE00 <= *addr && *addr < 0xFEA0:
		logger.Debugf("Writing %#x to %#x on Sprite Attribute Table (OAM)", v, *addr)

	/*
	*
	* WRITE: NOT USABLE
	*
	 */
	case 0xFEA0 <= *addr && *addr < 0xFF00:
		logger.Warningf("Writing %#x to %#x on Not Usable", v, *addr)

	/*
	*
	* WRITE: I/O REGISTERS
	*
	 */
	case 0xFF00 <= *addr && *addr < 0xFF80:
		logger.Debugf("Writing %#x to %#x on IO", v, *addr)

		if m.BootRomEnabled() &&
			*addr == 0xFF50 &&
			(v == 0x1 || v == 0x11) {
			logger.Debugf("Disabling boot rom")
			m.BootRom = nil
		}

		addr_copy -= 0xFF00
		m.Ram.SetItemIO(addr_copy, v)

	/*
	*
	* WRITE: HIGH RAM
	*
	 */
	case 0xFF80 <= *addr && *addr < 0xFFFF:
		logger.Debugf("Writing %#x to %#x on High RAM", v, addr_copy)
		addr_copy -= 0xFF80
		m.Ram.SetItemHRAM(addr_copy, v)

	/*
	*
	* WRITE: INTERRUPT ENABLE REGISTER
	*
	 */
	case *addr == 0xFFFF:
		logger.Debugf("Writing %#x to %#x on Interrupt Enable Register\n", *value, *addr)
		m.Cpu.Interrupts.IE = v
	default:
		internal.Logger.Panicf("Memory write error! Can't write `%#x` to `%#x`\n", *value, *addr)
	}

}
