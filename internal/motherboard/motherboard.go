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

func (m *Motherboard) SetItem(addr *uint16, value *uint16) {
	if 0x0 < *addr && *addr < 0x100 {
		internal.Panicf("Memory write error! Can't write to %+v\n", *addr)
	}

	switch {
	case 0x0000 <= *addr && *addr < 0x4000: // ROM bank 0
		// doesn't change the data. This is for MBC commands
		fmt.Printf("Writing %#x to %#x to Cartridge ROM bank\n", *value, *addr)

	case 0x4000 <= *addr && *addr < 0x8000: // Switchable ROM bank
		// doesn't change the data. This is for MBC commands
		fmt.Printf("Writing %+v to switchable Cartridge ROM bank\n", *value)

	case 0x8000 <= *addr && *addr < 0xA000: // 8K Video RAM
		fmt.Printf("Writing %+v to Video RAM\n", *value)

	case 0xA000 <= *addr && *addr < 0xC000: // 8K Switchable RAM bank
		fmt.Printf("Writing %#x to %#x on Switchable RAM bank\n", *value, *addr)

	case 0xC000 <= *addr && *addr < 0xE000: // 8K Internal RAM
		fmt.Printf("Writing %+v to Internal RAM\n", *value)

	case 0xE000 <= *addr && *addr < 0xFE00: // Echo of 8K Internal RAM
		fmt.Printf("Writing %+v to Echo of Internal RAM\n", *value)

	case 0xFE00 <= *addr && *addr < 0xFEA0: // Sprite Attribute Table (OAM)
		fmt.Printf("Writing %+v to Sprite Attribute Table\n", *value)

	case 0xFEA0 <= *addr && *addr < 0xFF00: // Not Usable
		fmt.Printf("Writing %+v to Not Usable\n", *value)

	case 0xFF00 <= *addr && *addr < 0xFF4C: // I/O Registers
		fmt.Printf("Writing %+v to I/O Registers\n", *value)

	case 0xFF4C <= *addr && *addr < 0xFF80: // Not Usable
		fmt.Printf("Writing %+v to Not Usable\n", *value)
	case 0xFF80 <= *addr && *addr < 0xFFFF: // Internal RAM
		fmt.Printf("Writing %+v to Internal RAM\n", *value)

	case *addr == 0xFFFF: // Interrupt Enable Register
		fmt.Printf("Writing %+v to Interrupt Enable Register\n", *value)
	default:
		internal.Panicf("Memory write error! Can't write to %+v\n", *addr)
	}

}

func (m *Motherboard) Cpu() *cpu.Cpu {
	return m.cpu
}
