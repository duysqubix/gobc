package motherboard

import (
	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
)

var logger = internal.Logger

type Breakpoints struct {
	Enabled     bool
	Addrs       []uint16
	LastElement uint16
}

type Motherboard struct {
	Cpu         *CPU
	Cartridge   *cartridge.Cartridge
	Cbg         bool
	Randomize   bool
	Decouple    bool // Decouple Motherboard from other components, and all calls to read/write memory will be mocked
	Breakpoints *Breakpoints
}

type MotherboardParams struct {
	Filename    *pathlib.Path
	Randomize   bool
	Cbg         bool
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
		Cbg:         params.Cbg,
		Cartridge:   cart,
		Randomize:   params.Randomize,
		Decouple:    params.Decouple,
		Breakpoints: bp,
	}
	mb.Cpu = NewCpu(mb)
	return mb
}

func (m *Motherboard) Tick() bool {
	if m.Cpu.Stopped {
		return false
	}
	m.Cpu.Tick()
	return true
}

func (m *Motherboard) GetItem(addr *uint16) uint8 {
	logger.Debugf("Reading from %#x on Motherboard\n", *addr)
	if m.Decouple {
		logger.Warn("Decoupled Motherboard from other components. Memory read is mocked")
		return 0xDA // mock return value
	}

	// debugging
	switch {
	case 0x0000 <= *addr && *addr < 0x4000: // ROM bank 0
		// doesn't change the data. This is for MBC commands
		return m.Cartridge.CartType.GetItem(*addr)

	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank
		// doesn't change the data. This is for MBC commands
		// fmt.Printf("Reading from %#x from Cartridge ROM bank\n", *addr)

	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM
		// fmt.Printf("Reading from %#x on Video RAM\n", *addr)

	case 0xA000 <= *addr && *addr < 0xC000: // 8K Switchable RAM bank
		// fmt.Printf("Reading from %#x on Switchable RAM bank\n", *addr)

	case 0xC000 <= *addr && *addr < 0xE000: // 8K Internal RAM
		// fmt.Printf("Reading from %#x on Internal RAM\n", *addr)

	case 0xE000 <= *addr && *addr < 0xFE00: // Echo of 8K Internal RAM
		// fmt.Printf("Reading from %#x on Echo of Internal RAM\n", *addr)

	case 0xFE00 <= *addr && *addr < 0xFEA0: // Sprite Attribute Table (OAM)
		// fmt.Printf("Reading from %#x on Sprite Attribute Table\n", *addr)

	case 0xFEA0 <= *addr && *addr < 0xFF00: // Not Usable
		// fmt.Printf("Reading from %#x on Not Usable\n", *addr)

	case 0xFF00 <= *addr && *addr < 0xFF4C: // I/O Registers
		// fmt.Printf("Reading from %#x on I/O Registers\n", *addr)

	case 0xFF4C <= *addr && *addr < 0xFF80: // Not Usable
		// fmt.Printf("Reading from %#x on Not Usable\n", *addr)

	case 0xFF80 <= *addr && *addr < 0xFFFF: // Internal RAM
		// fmt.Printf("Reading from %#x on Internal RAM\n", *addr)

	case *addr == 0xFFFF: // Interrupt Enable Register
		// fmt.Printf("Reading from %#x on Interrupt Enable Register\n", *addr)
	default:
		// internal.Panicf("Memory read error! Can't read from %#x\n", *addr)
	}

	return 0xFF
}

func (m *Motherboard) SetItem(addr *uint16, value *uint16) {
	logger.Debugf("Writing %#x to %#x on Motherboard\n", *value, *addr)
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
		// doesn't change the data. This is for MBC commands
		// fmt.Printf("Writing %#x to %#x to Cartridge ROM bank\n", *value, *addr)
		m.Cartridge.CartType.SetItem(*addr, v)

	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank
		// doesn't change the data. This is for MBC commands
		// fmt.Printf("Writing %#x to %#x to Cartridge ROM bank\n", *value, *addr)

	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM
		// fmt.Printf("Writing %#x to %#x on Video RAM\n", *value, *addr)

	case 0xA000 <= *addr && *addr < 0xC000: // 8K Switchable RAM bank
		// fmt.Printf("Writing %#x to %#x on Switchable RAM bank\n", *value, *addr)

	case 0xC000 <= *addr && *addr < 0xE000: // 8K Internal RAM
		// fmt.Printf("Writing %#x to %#x on Internal RAM\n", *value, *addr)

	case 0xE000 <= *addr && *addr < 0xFE00: // Echo of 8K Internal RAM
		// fmt.Printf("Writing %#x to %#x on Echo of 8K Internal RAM\n", *value, *addr)

	case 0xFE00 <= *addr && *addr < 0xFEA0: // Sprite Attribute Table (OAM)
		// fmt.Printf("Writing %#x to %#x on Sprite Attribute Table (OAM)\n", *value, *addr)

	case 0xFEA0 <= *addr && *addr < 0xFF00: // Not Usable
		// fmt.Printf("Writing %#x to %#x on Not Usable\n", *value, *addr)

	case 0xFF00 <= *addr && *addr < 0xFF4C: // I/O Registers
		// fmt.Printf("Writing %#x to %#x on I/O Registers\n", *value, *addr)

	case 0xFF4C <= *addr && *addr < 0xFF80: // Not Usable
		// fmt.Printf("Writing %#x to %#x on Not Usable\n", *value, *addr)

	case 0xFF80 <= *addr && *addr < 0xFFFF: // Internal RAM
		// fmt.Printf("Writing %#x to %#x on Internal RAM\n", *value, *addr)

	case *addr == 0xFFFF: // Interrupt Enable Register
		// fmt.Printf("Writing %#x to %#x on Interrupt Enable Register\n", *value, *addr)
		m.Cpu.Interrupts.IE = v
	default:
		internal.Logger.Panicf("Memory write error! Can't write `%#x` to `%#x`\n", *value, *addr)
	}

}
