package motherboard

import (
	"io"
	"math/rand"
)

type Tile [16]uint8

func (t *Tile) ParseTile() PaletteTile {
	var tile PaletteTile
	for i := 0; i < 15; i++ {
		for j := 0; j < 8; j++ {
			tile[i>>1][j] = (((t[i+1] >> j) & 0x1) << 1) | ((t[i] >> j) & 0x1)
		}
		i++
	}
	return tile
}

type PaletteTile [8][8]uint8

func (p *PaletteTile) IsBlank() bool {
	var testValue uint8 = 0xff
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if testValue != p[i][j] {
				return false
			}
		}
	}

	return true
}

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
	// ram[IO_LY-IO_START_ADDR] = 0x90
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

type Memory struct {
	Wram      WRAM // 8 banks of 4KB each -- [0,1] are always available, [2,3,4,5,6,7] are switchable in CGB Mode
	IO        IO   // 128 bytes of IO
	Hram      HRAM // 127 bytes of High RAM
	Vram      VRAM // 2 banks of 8KB each -- [0] is always available, [1] is switchable in CGB Mode
	Oam       OAM  // 160 bytes of OAM
	Randomize bool // Randomize RAM on startup
	Cgb       bool // CGB Mode
}

type IO [0x80]uint8
type HRAM [0x7f]uint8
type WRAM [0x8][0x1000]uint8
type OAM [0xa0]uint8
type VRAM [0x2][0x2000]uint8

func NewInternalRAM(cgb bool, randomize bool) *Memory {
	ram := &Memory{
		Randomize: randomize,
		Cgb:       cgb,
	}
	initWram(&ram.Wram, randomize)
	initIo(&ram.IO, randomize)
	initHram(&ram.Hram, randomize)
	initVram(&ram.Vram, randomize)
	initOam(&ram.Oam, randomize)

	return ram
}

// //////// VRAM //////////

func (r *Memory) TileData() []uint8 {
	return r.Vram[r.ActiveVramBank()][:6144]
}

func (r *Memory) TileMap() []uint8 {
	return r.Vram[r.ActiveVramBank()][6144:]
}

func (r *Memory) ActiveVramBank() uint8 {

	if r.Cgb {
		return r.IO[IO_VBK-IO_START_ADDR] & 0x1
	}
	return 0
}

////////// WRAM //////////

func (r *Memory) ActiveWramBank() uint8 {
	bank := r.IO[IO_SVBK-IO_START_ADDR] & 0x7 // force to 3 bits

	if bank == 0 || bank == 1 {
		return 1
	}
	return bank
}

func (r *Memory) DumpState(writer io.Writer) {

}
