package motherboard

import (
	"bytes"
	"encoding/binary"

	"github.com/duysqubix/gobc/internal"
)

const (
	// PaletteGreyscale is the default greyscale gameboy colour palette.
	PaletteGreyscale = byte(iota)
	// PaletteOriginal is more authentic looking green tinted gameboy
	// colour palette  as it would have been on the GameBoy
	PaletteOriginal
	// PaletteBGB used by default in the BGB emulator.
	PaletteBGB

	//NyX4-GB
	PaletteNyX4GB

	//Crimson
	PaletteCrimson
)

// CurrentPalette is the global current DMG palette.
var CurrentPalette = PaletteBGB

// Palettes is an mapping from colour palettes to their colour values
// to be used by the emulator.
var Palettes = [][][]byte{
	// PaletteGreyscale
	{
		{0xFF, 0xFF, 0xFF},
		{0xCC, 0xCC, 0xCC},
		{0x77, 0x77, 0x77},
		{0x00, 0x00, 0x00},
	},
	// PaletteOriginal
	{
		{0x9B, 0xBC, 0x0F},
		{0x8B, 0xAC, 0x0F},
		{0x30, 0x62, 0x30},
		{0x0F, 0x38, 0x0F},
	},
	// PaletteBGB
	{
		{0xE0, 0xF8, 0xD0},
		{0x88, 0xC0, 0x70},
		{0x34, 0x68, 0x56},
		{0x08, 0x18, 0x20},
	},
	//NyX4-GB
	{
		{0x8c, 0xab, 0xa1},
		{0x6d, 0x7a, 0x80},
		{0x0f, 0x2a, 0x3f},
		{0x08, 0x14, 0x1e},
	},
	//Crimson
	{
		{0xef, 0xf9, 0xd6},
		{0xba, 0x50, 0x44},
		{0x7a, 0x1c, 0x4b},
		{0x1b, 0x03, 0x26},
	},
	//ColdFire
	{
		{0xf6, 0xc6, 0xa8},
		{0xd1, 0x7c, 0x7c},
		{0x5b, 0x76, 0x8d},
		{0x46, 0x42, 0x5e},
	},
}

// GetPaletteColour returns the colour based on the colour index and the currently
// selected palette.
func GetPaletteColour(index byte) (uint8, uint8, uint8) {
	col := Palettes[CurrentPalette][index]
	return col[0], col[1], col[2]
}

// NewPalette makes a new CGB colour palette.
func NewPalette() *cgbPalette {
	pal := make([]byte, 0x40)
	for i := range pal {
		pal[i] = 0xFF
	}

	return &cgbPalette{Palette: pal}
}

func ChangePallete() {
	CurrentPalette = (CurrentPalette + 1) % byte(len(Palettes))
}

// Palette for cgb containing information tracking the palette colour info.
type cgbPalette struct {
	// Palette colour information.
	Palette []byte
	// Current index the palette is referencing.
	Index byte
	// If to auto increment on write.
	Inc bool
}

func (pal *cgbPalette) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, pal.Palette) // Palette
	binary.Write(buf, binary.LittleEndian, pal.Index)   // Index
	binary.Write(buf, binary.LittleEndian, pal.Inc)     // Inc
	logger.Debug("Serialized palette state")
	return buf
}

func (pal *cgbPalette) Deserialize(data *bytes.Buffer) error {
	// Read the data from the buffer
	if err := binary.Read(data, binary.LittleEndian, &pal.Palette); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &pal.Index); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &pal.Inc); err != nil {
		return err
	}

	return nil
}

// Update the index the palette is indexing and set
// auto increment if bit 7 is set.
func (pal *cgbPalette) updateIndex(value byte) {
	pal.Index = value & 0x3F
	pal.Inc = internal.IsBitSet(value, 7)
}

// Read the palette information stored at the current index.
func (pal *cgbPalette) read() byte {
	return pal.Palette[pal.Index]
}

// Read the current index.
func (pal *cgbPalette) readIndex() byte {
	return pal.Index
}

// Write a value to the palette at the current index.
func (pal *cgbPalette) write(value byte) {
	pal.Palette[pal.Index] = value
	if pal.Inc {
		pal.Index = (pal.Index + 1) & 0x3F
	}
}

// Get the rgb colour for a palette at a colour number.
func (pal *cgbPalette) get(palette byte, num byte) (uint8, uint8, uint8) {
	idx := (palette * 8) + (num * 2)
	colour := uint16(pal.Palette[idx]) | uint16(pal.Palette[idx+1])<<8
	r := uint8(colour & 0x1F)
	g := uint8((colour >> 5) & 0x1F)
	b := uint8((colour >> 10) & 0x1F)
	return colArr[r], colArr[g], colArr[b]
}

// Mapping of the 5 bit colour value to a 8 bit value.
var colArr = []uint8{
	0x0, // 0
	0x8,
	0x10,
	0x18,
	0x20,
	0x29, // 5
	0x31,
	0x39,
	0x41,
	0x4a,
	0x52, // 10
	0x5a,
	0x62,
	0x6a,
	0x73,
	0x7b, // 15
	0x83,
	0x8b,
	0x94,
	0x9c,
	0xa4, // 20
	0xac,
	0xb4,
	0xbd,
	0xc5,
	0xcd, // 25
	0xd5,
	0xde,
	0xe6,
	0xee,
	0xf6, // 30
	0xff,
}
