package motherboard

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/rand"
)

type Tile [16]uint8

func (t *Tile) ParseTile() PaletteTile {
	var tile PaletteTile
	for i := 0; i < 15; i++ {
		for j := 0; j < 8; j++ {
			tile[63-((i>>1)*8+j)] = (((t[i+1] >> j) & 0x1) << 1) | ((t[i] >> j) & 0x1)
		}
		i++
	}

	return tile
}

// type PaletteTile [8][8]uint8
type PaletteTile [64]uint8

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

func initIo(ram *IO, cgb bool) {
	if cgb {
		copy(ram[:], ioinitCGB[:])
	} else {
		copy(ram[:], ioinitDMG[:])
	}

	// delete once LCD is implemented
	// ram[IO_LY-IO_START_ADDR] = 0x90

	ram[IO_P1_JOYP-IO_START_ADDR] = 0xCF
}

func initVram(ram *VRAM, random bool) {
	var fixed uint8 = 0x00
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
	Wram      WRAM         // 8 banks of 4KB each -- [0,1] are always available, [2,3,4,5,6,7] are switchable in CGB Mode
	IO        IO           // 128 bytes of IO
	Hram      HRAM         // 127 bytes of High RAM
	Vram      VRAM         // 2 banks of 8KB each -- [0] is always available, [1] is switchable in CGB Mode
	Oam       OAM          // 160 bytes of OAM
	Randomize bool         // Randomize RAM on startup
	Cgb       bool         // CGB Mode
	Mb        *Motherboard // Motherboard
}

type IO [0x80]uint8
type HRAM [0x7f]uint8
type WRAM [0x8][0x1000]uint8
type OAM [0xa0]uint8

type VRAM [0x2][0x2000]uint8

func (r *Memory) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, r.Wram) // WRAM
	binary.Write(buf, binary.LittleEndian, r.Vram) // VRAM
	binary.Write(buf, binary.LittleEndian, r.Oam)  // OAM
	binary.Write(buf, binary.LittleEndian, r.IO)   // IO
	binary.Write(buf, binary.LittleEndian, r.Hram) // HRAM

	logger.Debug("Serialized memory state")
	return buf
}

func (r *Memory) Deserialize(data *bytes.Buffer) error {
	// Read the data from the buffer
	if err := binary.Read(data, binary.LittleEndian, &r.Wram); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &r.Vram); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &r.Oam); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &r.IO); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &r.Hram); err != nil {
		return err
	}

	return nil
}

func NewInternalRAM(mb *Motherboard, randomize bool) *Memory {
	ram := &Memory{
		Randomize: randomize,
		Cgb:       mb.Cgb,
		Mb:        mb,
	}
	initWram(&ram.Wram, randomize)
	initIo(&ram.IO, mb.Cgb)
	initHram(&ram.Hram, randomize)
	initVram(&ram.Vram, randomize)
	initOam(&ram.Oam, randomize)

	return ram
}

func (r *Memory) Reset() {
	initWram(&r.Wram, r.Randomize)
	initIo(&r.IO, r.Cgb)
	initHram(&r.Hram, r.Randomize)
	initVram(&r.Vram, r.Randomize)
	initOam(&r.Oam, r.Randomize)
}

// //////// IO //////////
// var setVbktrace uint8 = 0
// var getVbktrace uint8 = 0

func (r *Memory) SetIO(addr uint16, value uint8) {
	r.IO[addr-IO_START_ADDR] = value
}

func (r *Memory) GetIO(addr uint16) uint8 {
	return r.IO[addr-IO_START_ADDR]
}

// //////// VRAM //////////

func (r *Memory) TileData(bank uint8) []uint8 {
	return r.Vram[bank][:0x17ff]
}

// TileMap returns a
func (r *Memory) TileMap(addressingMode uint8, bgAddressingMode uint8) []uint8 {
	// tileMap := r.Vram[0][0x1800:]
	var tiles [131072]uint8 // 2, 32x32 tile maps
	var tileAddrStart int
	var tileIndexMax int
	var tileIndexOffset int
	// var Mode8000 bool = internal.IsBitSet(r.IO[IO_LCDC-IO_START_ADDR], LCDC_BGMAP)
	tileCntr := 0

	if bgAddressingMode == 1 { // 0x9800 tile map
		tileIndexMax = 0x9C00 - 0x9800
		tileIndexOffset = 0x1800
	} else { // 0x9C00 tile map
		tileIndexMax = 0xA000 - 0x9C00
		tileIndexOffset = 0x1C00
	}

	for tileIndex := 0; tileIndex < tileIndexMax; tileIndex++ {
		// starting with 0x9800 tile
		var tileOffset uint8 = r.Vram[0][tileIndexOffset+tileIndex]
		if addressingMode == 0 {
			// turn indexValue into a signed int
			tileAddrStart = 0x1000 + int(int8(tileOffset))*16
		} else {
			tileAddrStart = 0x0000 + (int(uint8(tileOffset)) * 16)
		}
		// now we have the tile address, we can copy the tile data into the tiles array
		for i := 0; i < 16; i++ {
			tiles[tileCntr] = r.Vram[0][tileAddrStart+i]
			tileCntr++
		}
	}

	return tiles[:]
}

func (r *Memory) ActiveVramBank() uint8 {

	if r.Cgb {
		return r.IO[IO_VBK-IO_START_ADDR] & 0x1
	}
	return 0
}

func (r *Memory) SetVram(bank uint8, addr uint16, value uint8) {
	r.Vram[bank][addr-0x8000] = value
}

func (r *Memory) GetVram(bank uint8, addr uint16) uint8 {
	return r.Vram[bank][addr-0x8000]
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

var ioinitDMG = [0x80]uint8{
	0xCF, 0x00, 0x7E, 0x00, 0xAB, 0x00, 0x00, 0xF8, // 0xFF00 - 0xFF07
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xE1, // 0xFF08 - 0xFF0F
	0x80, 0xBF, 0xF3, 0xFF, 0xBF, 0x00, 0x3F, 0x00, // 0xFF10 - 0xFF17
	0xFF, 0xBF, 0x7F, 0xFF, 0x9F, 0xFF, 0xBF, 0x00, // 0xFF18 - 0xFF1F
	0xFF, 0x00, 0x00, 0xBF, 0x77, 0xF3, 0xF1, 0x00, // 0xFF20 - 0xFF27
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF28 - 0xFF2F
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF30 - 0xFF37
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF38 - 0xFF3F
	0x91, 0x85, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFC, // 0xFF40 - 0xFF47
	0x07, 0x07, 0x00, 0x00, 0x00, 0xFF, 0x00, 0xFF, // 0xFF48 - 0xFF4F
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, // 0xFF50 - 0xFF57
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF58 - 0xFF5F
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF60 - 0xFF67
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, // 0xFF68 - 0xFF6F
	0xFF,
}

var ioinitCGB = [0x80]uint8{
	0xCF, 0x00, 0x7F, 0x00, 0x06, 0x00, 0x00, 0xF8, // 0xFF00 - 0xFF07
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xE1, // 0xFF08 - 0xFF0F
	0x80, 0xBF, 0xF3, 0xFF, 0xBF, 0x00, 0x3F, 0x00, // 0xFF10 - 0xFF17
	0xFF, 0xBF, 0x7F, 0xFF, 0x9F, 0xFF, 0xBF, 0x00, // 0xFF18 - 0xFF1F
	0xFF, 0x00, 0x00, 0xBF, 0x77, 0xF3, 0xF1, 0x00, // 0xFF20 - 0xFF27
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF28 - 0xFF2F
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF30 - 0xFF37
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF38 - 0xFF3F
	0x91, 0x06, 0x00, 0x00, 0x06, 0x00, 0x00, 0xFC, // 0xFF40 - 0xFF47
	0x07, 0x07, 0x00, 0x00, 0x00, 0xFF, 0x00, 0xFF, // 0xFF48 - 0xFF4F
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, // 0xFF50 - 0xFF57
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF58 - 0xFF5F
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0xFF60 - 0xFF67
	0x08, 0x08, 0x08, 0x08, 0x00, 0x00, 0x00, 0x00, // 0xFF68 - 0xFF6F
	0xFF,
}
