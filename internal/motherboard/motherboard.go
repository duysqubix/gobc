package motherboard

import (
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
	Cbg         bool                 // Color Gameboy
	Randomize   bool                 // Randomize RAM on startup
	Decouple    bool                 // Decouple Motherboard from other components, and all calls to read/write memory will be mocked
	Breakpoints *Breakpoints         // Breakpoints
}

type MotherboardParams struct {
	Filename    *pathlib.Path
	Randomize   bool
	ForceCbg    bool
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

	mb.Cbg = mb.Cartridge.CbgModeEnabled() || params.ForceCbg
	mb.Cpu = NewCpu(mb)
	mb.Ram = NewInternalRAM(mb.Cbg, params.Randomize)

	return mb
}

func (m *Motherboard) Tick() (bool, OpCycles) {
	if m.Cpu.Stopped || m.Cpu.Halted || m.Cpu.IsStuck {
		return false, 0
	}
	cycles := m.Cpu.Tick()
	return true, cycles
}

func (m *Motherboard) GetItem(addr *uint16) uint8 {
	// logger.Debugf("Reading from %#x on Motherboard\n", *addr)
	if m.Decouple {
		logger.Warn("Decoupled Motherboard from other components. Memory read is mocked")
		return 0xDA // mock return value
	}

	// debugging
	switch {
	case 0x0000 <= *addr && *addr < 0x4000: // ROM bank 0
		return m.Cartridge.CartType.GetItem(*addr)

	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank

	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM

	case 0xA000 <= *addr && *addr < 0xC000: // 8K External RAM (Cartridge)

	case 0xC000 <= *addr && *addr < 0xD000: // 4K Work RAM bank 0
		return m.Ram.GetItemWRAM(0, *addr)

	case 0xD000 <= *addr && *addr < 0xE000: // 4K Work RAM bank 1 (or switchable bank 1)
		logger.Debugf("Reading from %#x on Work RAM", *addr)
		(*addr) -= 0xD000
		// check if CGB mode
		if m.Cbg {
			// check what bank to read from
			bank := m.Ram.ActiveWramBank()
			logger.Debugf("Bank: %d", bank)
			return m.Ram.GetItemWRAM(bank, *addr)
		}
		logger.Debugf("%d\n", 1)
		return m.Ram.GetItemWRAM(1, *addr)

	case 0xE000 <= *addr && *addr < 0xFE00: // Echo of 8K Internal RAM

	case 0xFE00 <= *addr && *addr < 0xFEA0: // Sprite Attribute Table (OAM)

	case 0xFEA0 <= *addr && *addr < 0xFF00: // Not Usable

	case 0xFF00 <= *addr && *addr < 0xFF80: // I/O Registers
		logger.Debugf("Reading from %#x on IO", *addr)

		return m.Ram.GetItemIO(*addr)

	case 0xFF80 <= *addr && *addr < 0xFFFF: // High RAM

	case *addr == 0xFFFF: // Interrupt Enable Register
	default:
		logger.Panicf("Memory read error! Can't read from %#x\n", *addr)
	}

	return 0xFF
}

func (m *Motherboard) SetItem(addr *uint16, value *uint16) {
	// logger.Debugf("Writing %#x to %#x on Motherboard\n", *value, *addr)
	// preventing overflow of 8 bits
	// writing to memory should only be 8 bits
	if *value >= 0x100 {
		internal.Logger.Panicf("Memory write error! Can't write %#x to %#x\n", *value, *addr)
	}

	if m.Decouple {
		logger.Warn("Decoupled Motherboard from other components. Memory write is mocked")
		return
	}

	v := uint8(*value)
	switch {
	case 0x0000 <= *addr && *addr < 0x4000: // ROM bank 0
		m.Cartridge.CartType.SetItem(*addr, v)

	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank

	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM

	case 0xA000 <= *addr && *addr < 0xC000: // 8K External RAM (Cartridge)

	case 0xC000 <= *addr && *addr < 0xD000: // 4K Work RAM bank 0
		m.Ram.SetItemWRAM(0, *addr, v)

	case 0xD000 <= *addr && *addr < 0xE000: // 4K Work RAM bank 1 (or switchable bank 1)
		logger.Debugf("Writing %#x to %#x on Work RAM", v, *addr)
		(*addr) -= 0xD000
		// check if CGB mode
		if m.Cbg {
			// check what bank to read from
			bank := m.Ram.ActiveWramBank()
			logger.Errorf("Bank: %d", bank)
			m.Ram.SetItemWRAM(bank, *addr, v)
			break
		}
		logger.Debugf("Bank: %d", 1)
		m.Ram.SetItemWRAM(1, *addr, v)

	case 0xE000 <= *addr && *addr < 0xFE00: // Echo of 8K Internal RAM

	case 0xFE00 <= *addr && *addr < 0xFEA0: // Sprite Attribute Table (OAM)

	case 0xFEA0 <= *addr && *addr < 0xFF00: // Not Usable

	case 0xFF00 <= *addr && *addr < 0xFF80: // I/O Registers
		logger.Debugf("Writing %#x to %#x on IO", v, *addr)
		m.Ram.SetItemIO(*addr, v)

	case 0xFF80 <= *addr && *addr < 0xFFFF: // High RAM

	case *addr == 0xFFFF: // Interrupt Enable Register
		// fmt.Printf("Writing %#x to %#x on Interrupt Enable Register\n", *value, *addr)
		m.Cpu.Interrupts.IE = v
	default:
		internal.Logger.Panicf("Memory write error! Can't write `%#x` to `%#x`\n", *value, *addr)
	}

}
