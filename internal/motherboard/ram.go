package motherboard

import (
	"io"
	"math/rand"
)

func initWram(ram *[8][4096]uint8, random bool) {
	var fixed uint8 = 0xFF
	for i := 0; i < 8; i++ {
		for j := 0; j < 4096; j++ {
			if random {
				ram[i][j] = uint8(rand.Intn(256))
			} else {
				ram[i][j] = fixed
			}
		}
	}
}

func initHram(ram *[127]uint8, random bool) {
	var fixed uint8 = 0xFF
	for i := 0; i < 127; i++ {
		if random {
			ram[i] = uint8(rand.Intn(256))
		} else {
			ram[i] = fixed
		}
	}
}

func initIo(ram *[128]uint8, random bool) {
	var fixed uint8 = 0xFF
	for i := 0; i < 128; i++ {
		if random {
			ram[i] = uint8(rand.Intn(256))
		} else {
			ram[i] = fixed
		}
	}
}

func initVram(ram *[2][8192]uint8, random bool) {
	var fixed uint8 = 0xFF
	for i := 0; i < 2; i++ {
		for j := 0; j < 8192; j++ {
			if random {
				ram[i][j] = uint8(rand.Intn(256))
			} else {
				ram[i][j] = fixed
			}
		}
	}
}

func initOam(ram *[160]uint8, random bool) {
	var fixed uint8 = 0xFF

	for i := 0; i < 160; i++ {
		if random {
			ram[i] = uint8(rand.Intn(256))
		} else {
			ram[i] = fixed
		}
	}
}

type InternalRAM struct {
	Wram      [8][4096]uint8 // 8 banks of 4KB each -- [0,1] are always available, [2,3,4,5,6,7] are switchable in CGB Mode
	IO        [128]uint8     // 128 bytes of IO
	Hram      [127]uint8     // 127 bytes of High RAM
	Vram      [2][8192]uint8 // 2 banks of 8KB each -- [0] is always available, [1] is switchable in CGB Mode
	Oam       [160]uint8     // 160 bytes of OAM
	Randomize bool           // Randomize RAM on startup
}

func NewInternalRAM(cgb bool, randomize bool) *InternalRAM {
	ram := &InternalRAM{
		Randomize: randomize,
	}
	initWram(&ram.Wram, randomize)
	initIo(&ram.IO, randomize)
	initHram(&ram.Hram, randomize)
	initVram(&ram.Vram, randomize)
	initOam(&ram.Oam, randomize)

	return ram
}

func (r *InternalRAM) ActiveWramBank() uint8 {
	logger.Debug("Checking Active WRAM bank...")
	bank := r.GetItemIO(IO_SVBK) & 0x7 // force to 3 bits

	if bank == 0 || bank == 1 {
		return 1
	}
	return bank
}

func (r *InternalRAM) ActiveVramBank() uint8 {
	logger.Debug("Checking Active VRAM bank...")
	return r.GetItemIO(IO_VBK) & 0x1
}

func (r *InternalRAM) GetItemWRAM(bank uint8, addr uint16) uint8 {
	return r.Wram[bank][addr]
}

func (r *InternalRAM) GetItemIO(addr uint16) uint8 {
	return r.IO[addr]
}

func (r *InternalRAM) GetItemHRAM(addr uint16) uint8 {
	return r.Hram[addr]
}

func (r *InternalRAM) GetItemVRAM(bank uint8, addr uint16) uint8 {
	return r.Vram[bank][addr]
}

func (r *InternalRAM) GetItemOAM(addr uint16) uint8 {

	return r.Oam[addr]
}

func (r *InternalRAM) SetItemWRAM(bank uint8, addr uint16, value uint8) {
	r.Wram[bank][addr] = value
}

func (r *InternalRAM) SetItemIO(addr uint16, value uint8) {
	addr -= IO_START_ADDR
	r.IO[addr] = value
}

func (r *InternalRAM) SetItemHRAM(addr uint16, value uint8) {
	r.Hram[addr] = value
}

func (r *InternalRAM) SetItemVRAM(bank uint8, addr uint16, value uint8) {
	r.Vram[bank][addr] = value
}

func (r *InternalRAM) SetItemOAM(addr uint16, value uint8) {
	r.Oam[addr] = value
}

func (r *InternalRAM) DumpState(writer io.Writer) {
	// table := tablewriter.NewWriter(writer)
	// table.SetHeader([]string{"Memory Space", "Values"})
	// var report [][]string

	// // WRAM

}
