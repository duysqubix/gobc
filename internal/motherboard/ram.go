package motherboard

import (
	"io"
	"math/rand"
)

func initWram(ram *WRAM, random bool) {
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

func initHram(ram *HRAM, random bool) {
	var fixed uint8 = 0xFF
	for i := 0; i < 127; i++ {
		if random {
			ram[i] = uint8(rand.Intn(256))
		} else {
			ram[i] = fixed
		}
	}
}

func initIo(ram *IO, random bool) {
	var fixed uint8 = 0xFF
	for i := 0; i < 128; i++ {
		if random {
			ram[i] = uint8(rand.Intn(256))
		} else {
			ram[i] = fixed
		}
	}

	// delete once LCD is implemented
	ram[IO_LY] = 0x90
}

func initVram(ram *VRAM, random bool) {
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

func initOam(ram *OAM, random bool) {
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
	Wram      WRAM // 8 banks of 4KB each -- [0,1] are always available, [2,3,4,5,6,7] are switchable in CGB Mode
	IO        IO   // 128 bytes of IO
	Hram      HRAM // 127 bytes of High RAM
	Vram      VRAM // 2 banks of 8KB each -- [0] is always available, [1] is switchable in CGB Mode
	Oam       OAM  // 160 bytes of OAM
	Randomize bool // Randomize RAM on startup
}

type IO [128]uint8
type HRAM [127]uint8
type WRAM [8][4096]uint8
type OAM [160]uint8

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

// //////// IO //////////
func (r *InternalRAM) GetItemIO(addr uint16) uint8 {
	return r.IO[addr]
}

func (r *InternalRAM) SetItemIO(addr uint16, value uint8) {
	r.IO[addr] = value
}

func (r *InternalRAM) GetIO_LCDC(bit uint8) bool {
	return r.GetItemIO(IO_LCDC)>>bit&0x1 == 1
}

// //////// VRAM //////////

type VRAM [2][8192]uint8
type Tile [16]uint8

func (r *InternalRAM) ActiveVramBank() uint8 {
	logger.Debug("Checking Active VRAM bank...")
	return r.GetItemIO(IO_VBK) & 0x1
}

func (r *InternalRAM) GetItemVRAM(bank uint8, addr uint16) uint8 {
	return r.Vram[bank][addr]
}

func (r *InternalRAM) SetItemVRAM(bank uint8, addr uint16, value uint8) {
	r.Vram[bank][addr] = value
}

////////// WRAM //////////

func (r *InternalRAM) ActiveWramBank() uint8 {
	logger.Debug("Checking Active WRAM bank...")
	bank := r.GetItemIO(IO_SVBK) & 0x7 // force to 3 bits

	if bank == 0 || bank == 1 {
		return 1
	}
	return bank
}

func (r *InternalRAM) GetItemWRAM(bank uint8, addr uint16) uint8 {
	return r.Wram[bank][addr]
}

func (r *InternalRAM) SetItemWRAM(bank uint8, addr uint16, value uint8) {
	r.Wram[bank][addr] = value
}

// //////// HRAM //////////
func (r *InternalRAM) GetItemHRAM(addr uint16) uint8 {
	return r.Hram[addr]
}

func (r *InternalRAM) SetItemHRAM(addr uint16, value uint8) {
	r.Hram[addr] = value
}

////////// OAM //////////

func (r *InternalRAM) GetItemOAM(addr uint16) uint8 {

	return r.Oam[addr]
}

func (r *InternalRAM) SetItemOAM(addr uint16, value uint8) {
	r.Oam[addr] = value
}

func (r *InternalRAM) DumpState(writer io.Writer) {

}
