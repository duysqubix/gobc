package motherboard

import (
	"fmt"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cpu"
)

type Motherboard struct {
	cpu *cpu.Cpu
	//Ram Ram
	//Cartridge Cartridge
}

func NewMotherboard() *Motherboard {
	return &Motherboard{
		cpu: cpu.NewCpu(),
	}
}

func (m *Motherboard) GetItem(addr *uint16) uint8 {
	// debugging
	switch {
	case 0x0000 <= *addr && *addr < 0x4000: // ROM bank 0
		// doesn't change the data. This is for MBC commands
		fmt.Printf("Reading from %#x from Cartridge ROM bank\n", *addr)

	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank
		// doesn't change the data. This is for MBC commands
		fmt.Printf("Reading from %#x from Cartridge ROM bank\n", *addr)

	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM
		fmt.Printf("Reading from %#x on Video RAM\n", *addr)

	case 0xA000 <= *addr && *addr < 0xC000: // 8K Switchable RAM bank
		fmt.Printf("Reading from %#x on Switchable RAM bank\n", *addr)

	case 0xC000 <= *addr && *addr < 0xE000: // 8K Internal RAM
		fmt.Printf("Reading from %#x on Internal RAM\n", *addr)

	case 0xE000 <= *addr && *addr < 0xFE00: // Echo of 8K Internal RAM
		fmt.Printf("Reading from %#x on Echo of Internal RAM\n", *addr)

	case 0xFE00 <= *addr && *addr < 0xFEA0: // Sprite Attribute Table (OAM)
		fmt.Printf("Reading from %#x on Sprite Attribute Table\n", *addr)

	case 0xFEA0 <= *addr && *addr < 0xFF00: // Not Usable
		fmt.Printf("Reading from %#x on Not Usable\n", *addr)

	case 0xFF00 <= *addr && *addr < 0xFF4C: // I/O Registers
		fmt.Printf("Reading from %#x on I/O Registers\n", *addr)

	case 0xFF4C <= *addr && *addr < 0xFF80: // Not Usable
		fmt.Printf("Reading from %#x on Not Usable\n", *addr)

	case 0xFF80 <= *addr && *addr < 0xFFFF: // Internal RAM
		fmt.Printf("Reading from %#x on Internal RAM\n", *addr)

	case *addr == 0xFFFF: // Interrupt Enable Register
		fmt.Printf("Reading from %#x on Interrupt Enable Register\n", *addr)
	default:
		internal.Panicf("Memory read error! Can't read from %#x\n", *addr)
	}

	return 0x01
}

func (m *Motherboard) SetItem(addr *uint16, value *uint16) {

	// preventing overflow of 8 bits
	// writing to memory should only be 8 bits
	if *value >= 0x100 {
		internal.Panicf("Memory write error! Can't write %#x to %#x\n", *value, *addr)
	}

	switch {
	case 0x0000 <= *addr && *addr < 0x4000: // ROM bank 0
		// doesn't change the data. This is for MBC commands
		fmt.Printf("Writing %#x to %#x to Cartridge ROM bank\n", *value, *addr)

	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank
		// doesn't change the data. This is for MBC commands
		fmt.Printf("Writing %#x to %#x to Cartridge ROM bank\n", *value, *addr)

	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM
		fmt.Printf("Writing %#x to %#x on Video RAM\n", *value, *addr)

	case 0xA000 <= *addr && *addr < 0xC000: // 8K Switchable RAM bank
		fmt.Printf("Writing %#x to %#x on Switchable RAM bank\n", *value, *addr)

	case 0xC000 <= *addr && *addr < 0xE000: // 8K Internal RAM
		fmt.Printf("Writing %#x to %#x on Internal RAM\n", *value, *addr)

	case 0xE000 <= *addr && *addr < 0xFE00: // Echo of 8K Internal RAM
		fmt.Printf("Writing %#x to %#x on Echo of 8K Internal RAM\n", *value, *addr)

	case 0xFE00 <= *addr && *addr < 0xFEA0: // Sprite Attribute Table (OAM)
		fmt.Printf("Writing %#x to %#x on Sprite Attribute Table (OAM)\n", *value, *addr)

	case 0xFEA0 <= *addr && *addr < 0xFF00: // Not Usable
		fmt.Printf("Writing %#x to %#x on Not Usable\n", *value, *addr)

	case 0xFF00 <= *addr && *addr < 0xFF4C: // I/O Registers
		fmt.Printf("Writing %#x to %#x on I/O Registers\n", *value, *addr)

	case 0xFF4C <= *addr && *addr < 0xFF80: // Not Usable
		fmt.Printf("Writing %#x to %#x on Not Usable\n", *value, *addr)

	case 0xFF80 <= *addr && *addr < 0xFFFF: // Internal RAM
		fmt.Printf("Writing %#x to %#x on Internal RAM\n", *value, *addr)

	case *addr == 0xFFFF: // Interrupt Enable Register
		fmt.Printf("Writing %#x to %#x on Interrupt Enable Register\n", *value, *addr)
	default:
		internal.Panicf("Memory write error! Can't write `%#x` to `%#x`\n", *value, *addr)
	}

}

func (m *Motherboard) Cpu() *cpu.Cpu {
	return m.cpu
}
