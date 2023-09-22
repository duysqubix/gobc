package motherboard

import (
	"github.com/duysqubix/gobc/internal"
)

type ScreenData [internal.GB_SCREEN_WIDTH][internal.GB_SCREEN_HEIGHT][3]uint8
type ScreenPriority [internal.GB_SCREEN_WIDTH][internal.GB_SCREEN_HEIGHT]bool

const (
	lcdMode2Bounds       = 456 - 80
	lcdMode3Bounds       = lcdMode2Bounds - 172
	spritePriorityOffset = 100
)

type LCD struct {

	// Matrix of pixel data which is used while the screen is rendering. When the screen is done rendering, this data is copied to the PreparedData matrix.
	// screenData ScreenData
	bgPriority ScreenPriority

	tileScanline    [internal.GB_SCREEN_WIDTH]uint8
	scanlineCounter OpCycles
	screenCleared   bool

	// PreparedData is a matrix of screen pixel data for a single frame which has been fully rendered
	PreparedData         ScreenData
	Mb                   *Motherboard
	CurrentPixelPosition uint8 // current pixel position in the scanline
	CurrentScanline      uint8 // current scanline being rendered
	WindowLY             uint8 // current window scanline being rendered
}

func NewLCD(mb *Motherboard) *LCD {
	return &LCD{
		Mb: mb,
	}
}

func (l *LCD) Reset() {
	// l.screenData = ScreenData{}
	l.bgPriority = ScreenPriority{}
	l.PreparedData = ScreenData{}
	l.scanlineCounter = 0
	l.screenCleared = false
}

func (l *LCD) Tick(cycles OpCycles) {
	l.updateGraphics(cycles)
}

func (l *LCD) ReportOnSTAT(bit uint8) []string {
	var bitOff = "OFF"
	if internal.IsBitSet(l.Mb.Memory.IO[IO_STAT-IO_START_ADDR], bit) {
		bitOff = "ON"
	}
	return []string{
		STATBitNames[bit],
		bitOff,
	}
}

func (l *LCD) ReportOnLCDC(bit uint8, on, off string) []string {
	var bitOff = off
	if internal.IsBitSet(l.Mb.Memory.IO[IO_LCDC-IO_START_ADDR], bit) {
		bitOff = on
	}
	return []string{
		LCDCBitNames[bit],
		bitOff,
	}
}

func (l *LCD) updateGraphics(cycles OpCycles) {

	if !l.isLCDEnabled() {
		return
	}
	l.scanlineCounter -= cycles

	l.setLCDStatus()
	if l.scanlineCounter <= 0 {
		l.Mb.Memory.IO[IO_LY-IO_START_ADDR]++ // directly change for optimized performance
		if l.Mb.Memory.IO[IO_LY-IO_START_ADDR] > 153 {
			// l.PreparedData = ScreenData{}
			// l.screenData = ScreenData{}
			// l.clearScreen()
			l.bgPriority = ScreenPriority{}
			l.Mb.Memory.IO[IO_LY-IO_START_ADDR] = 0
		}

		l.scanlineCounter += (456 * 1) // change 1 to 2 for double speed

		if l.Mb.Memory.IO[IO_LY-IO_START_ADDR] == internal.GB_SCREEN_HEIGHT {
			l.Mb.Cpu.SetInterruptFlag(INTR_VBLANK)
		}
	}
}

var prevLY = uint8(0)

func (l *LCD) setLCDStatus() {

	status := l.Mb.Memory.IO[IO_STAT-IO_START_ADDR]
	if !l.isLCDEnabled() {
		// clear the screen
		l.clearScreen()
		l.scanlineCounter = 456 // total cycles per scanline

		l.Mb.Memory.IO[IO_LY-IO_START_ADDR] = 0

		// reset status
		status &= 252
		internal.ResetBit(&status, 0)
		internal.ResetBit(&status, 1)

		// write status to memory
		l.Mb.Memory.IO[IO_STAT-IO_START_ADDR] = status
		return
	}

	l.screenCleared = false

	currentLine := l.Mb.Memory.IO[IO_LY-IO_START_ADDR]
	l.CurrentScanline = currentLine
	currentMode := status & 0b11

	var mode uint8
	rqstInterrupt := false

	switch {

	case currentLine >= 144:
		mode = STAT_MODE_VBLANK
		internal.SetBit(&status, STAT_MODE0)
		internal.ResetBit(&status, STAT_MODE1)
		rqstInterrupt = internal.IsBitSet(status, STAT_VBLINT)

	case l.scanlineCounter >= lcdMode2Bounds:
		mode = STAT_MODE_OAM
		internal.ResetBit(&status, STAT_MODE0)
		internal.SetBit(&status, STAT_MODE1)
		rqstInterrupt = internal.IsBitSet(status, STAT_OAMINT)

	case l.scanlineCounter >= lcdMode3Bounds:
		mode = STAT_MODE_TRANS
		internal.SetBit(&status, STAT_MODE0)
		internal.SetBit(&status, STAT_MODE1)
		if mode != currentMode {
			// draw scanline when we start mode 3. In the real gameboy
			// this would be done through mode 3 by readong OAM and VRAM
			// to generate the picture
			l.drawScanline(currentLine)
		}
	default:
		mode = STAT_MODE_HBLANK
		internal.ResetBit(&status, STAT_MODE1)
		internal.ResetBit(&status, STAT_MODE0)
		rqstInterrupt = internal.IsBitSet(status, STAT_HBLINT)
		if mode != currentMode {
			// fmt.Printf("Mode is now HBLANK\n")
			l.Mb.DoHDMATransfer() // do HDMATransfer when we start mode 0
		}
	}

	if rqstInterrupt && mode != currentMode && prevLY != currentLine {
		l.Mb.Cpu.SetInterruptFlag(INTR_LCDSTAT)
	}

	// // check if LYC == LY (coincedence flag)
	if currentLine == l.Mb.Memory.IO[IO_LYC-IO_START_ADDR] {
		internal.SetBit(&status, STAT_LYC)
		if internal.IsBitSet(status, STAT_LYCINT) && prevLY != currentLine {
			// l.Mb.GuiPause = true
			l.Mb.Cpu.SetInterruptFlag(INTR_LCDSTAT)

		}
	} else {
		internal.ResetBit(&status, STAT_LYC)
	}
	if prevLY != currentLine {
		prevLY = currentLine
	}
	// write status to memory
	l.Mb.Memory.IO[IO_STAT-IO_START_ADDR] = status
}

func (l *LCD) isLCDEnabled() bool {
	return internal.IsBitSet(l.Mb.Memory.IO[IO_LCDC-IO_START_ADDR], LCDC_ENABLE)
}

func (l *LCD) drawScanline(scanline uint8) {
	// internal.ResetBit(&l.Mb.Memory.IO[IO_LCDC-IO_START_ADDR], LCDC_WINEN)
	// internal.ResetBit(&l.Mb.Memory.IO[IO_LCDC-IO_START_ADDR], LCDC_OBJEN)

	control := l.Mb.Memory.IO[IO_LCDC-IO_START_ADDR]

	// disable window  for debugging

	// LCDC bit 0 clears tiles on DMG but controls priority on CBG
	if l.Mb.Cgb || internal.IsBitSet(control, LCDC_BGEN) {
		l.renderTiles(control, scanline)
	}

	if internal.IsBitSet(control, LCDC_OBJEN) {
		l.renderSprites(control, int32(scanline))
	}
}

type tileSettings struct {
	UsingWindow bool   // true if window is enabled
	Unsigned    bool   // true if using unsigned tile numbers
	TileData    uint16 // address of tile data
	BgMemory    uint16 // address of background tile map
}

func (l *LCD) getTileSettings(lcdControl uint8, windowY uint8) tileSettings {
	var usingWindow bool = false
	var tileData uint16 = uint16(0x8800)
	var bgMemory uint16 = uint16(0x9800)
	var unsigned bool = false

	if internal.IsBitSet(lcdControl, LCDC_WINEN) {
		// is current scanline we are drawing within the window?
		if windowY <= l.Mb.Memory.IO[IO_LY-IO_START_ADDR] {
			usingWindow = true
		}
	}
	// test if we are using unsigned bytes
	if internal.IsBitSet(lcdControl, LCDC_BGMAP) {
		tileData = 0x8000
		unsigned = true
	}

	// work out where to look in the background memory
	var testBit uint8 = 3
	if usingWindow {
		testBit = 6
	}

	if internal.IsBitSet(lcdControl, testBit) {
		bgMemory = 0x9C00
	}

	return tileSettings{
		TileData:    tileData,
		BgMemory:    bgMemory,
		UsingWindow: usingWindow,
		Unsigned:    unsigned,
	}
}

func (l *LCD) renderTiles(lcdControl uint8, scanline uint8) {
	scrollY := l.Mb.Memory.IO[IO_SCY-IO_START_ADDR]
	scrollX := l.Mb.Memory.IO[IO_SCX-IO_START_ADDR]
	windowY := l.Mb.Memory.IO[IO_WY-IO_START_ADDR]
	windowX := l.Mb.Memory.IO[IO_WX-IO_START_ADDR] - 7

	ts := l.getTileSettings(lcdControl, windowY)

	var yPos uint8
	if !ts.UsingWindow {
		yPos = scrollY + scanline
	} else {
		yPos = scanline - windowY
	}

	tileRow := uint16(yPos/8) * 32
	palette := l.Mb.Memory.IO[IO_BGP-IO_START_ADDR]
	l.tileScanline = [internal.GB_SCREEN_WIDTH]uint8{}

	for pixel := uint8(0); pixel < internal.GB_SCREEN_WIDTH; pixel++ {

		xPos := pixel + scrollX

		// translate current x pos to window space if necessary
		if ts.UsingWindow && pixel >= windowX {
			xPos = pixel - windowX
		}

		l.CurrentPixelPosition = xPos
		// which of the 32 horizontal tiles does this xPos fall within?
		tileCol := uint16(xPos / 8)

		tileAddress := ts.BgMemory + tileRow + tileCol

		//deduce tile id in memory
		tileLocation := ts.TileData

		var tileNum int16
		if ts.Unsigned {
			tileNum = int16(l.Mb.Memory.Vram[0][tileAddress-0x8000])
			tileLocation = tileLocation + uint16(tileNum*16)
		} else {
			tileNum = int16(int8(l.Mb.Memory.Vram[0][tileAddress-0x8000]))
			tileLocation = uint16(int32(tileLocation) + int32((tileNum+128)*16))
		}

		// Attributes used in CGB mode TODO: check in CGB mode
		//
		//    Bit 0-2  Background Palette number  (BGP0-7)
		//    Bit 3    Tile VRAM Bank number      (0=Bank 0, 1=Bank 1)
		//    Bit 5    Horizontal Flip            (0=Normal, 1=Mirror horizontally)
		//    Bit 6    Vertical Flip              (0=Normal, 1=Mirror vertically)
		//    Bit 7    BG-to-OAM Priority         (0=Use OAM priority bit, 1=BG Priority)
		//

		bank := 0

		tileAttr := l.Mb.Memory.Vram[1][tileAddress-0x8000]
		if l.Mb.Cgb && internal.IsBitSet(tileAttr, 3) {
			bank = 1
		}

		priority := internal.IsBitSet(tileAttr, 7)

		var line uint16
		if l.Mb.Cgb && internal.IsBitSet(tileAttr, 6) {
			// vertical flip
			line = uint16((7-yPos)%8) * 2
		} else {
			line = uint16(yPos%8) * 2
		}

		data1 := l.Mb.Memory.Vram[bank][tileLocation+uint16(line)-0x8000]
		data2 := l.Mb.Memory.Vram[bank][tileLocation+uint16(line)+1-0x8000]

		if l.Mb.Cgb && internal.IsBitSet(tileAttr, 5) {
			// horizontal flip
			xPos -= 7
		}

		colorBit := uint8(int8((xPos%8)-7) * -1)
		colorNum := (internal.BitValue(data2, colorBit) << 1) | internal.BitValue(data1, colorBit)
		if lcdControl != oldlcdc {
			oldlcdc = lcdControl
			// // logger.Debug(l.CurrentScanline)
			// lys.Add(int(l.CurrentScanline))
			// slice := lys.ToSlice()
			// sort.Ints(slice)
			// // logger.Debug(slice)
		}

		if (scanline == 53) && (pixel == 48) {
			// Map Address: 0x9945
			// Tile Address: 0:8090
			// tileNum: 0x09
			// logger.Debugf("Map Address: %#x, Tile Address: %#x, tileNum: %#x, unsignedTile: %t, LCDC: %08b", tileAddress, tileLocation, tileNum, ts.Unsigned, lcdControl)

			// logger.Debugf("Scanline: %d, Pixel: %d, xPos: %d, yPos: %d, offSet: %#x, oldoffSet: %#x, tileLocation: %#x, tileData: %#x,  tileAddress: %#x, tileAttr: %#x, data1: %#x, data2: %#x, LCDC: %08b, STAT: %08b, IE: %08b, IF: %08b: BGMem: %#x, Unsigned: %t\n", scanline, pixel, xPos, yPos, offSet, oldOffSet, tileLocation, ts.TileData, tileAddress, tileAttr, data1, data2, lcdControl, l.Mb.Memory.IO[IO_STAT-IO_START_ADDR], l.Mb.Cpu.Interrupts.IE, l.Mb.Cpu.Interrupts.IF, ts.BgMemory, ts.Unsigned)
			// logger.Debugf("Cycles: %d)", l.scanlineCounter)
		}

		// if data1 != 0x00 || data2 != 0x00 {
		// 	fmt.Printf("-----------------\n"+
		// 		"data1: %#x, data2: %#x, tileLocation: %#x, line: %#x, tileAddress: %#x, tileAttr: %#x\n"+
		// 		"Unsigned: %t, tileNum: %#x, tileData: %#x, yPos: %#x, xPos: %#x, tileCol: %#x\n"+
		// 		"tileRow: %#x, scrollX: %#x, scrollY: %#x, windowX: %#x, windowY: %#x, pixel: %#x, colorBit: %#x, colorNum: %#x\n",
		// 		data1, data2, tileLocation, line, tileAddress, tileAttr, ts.Unsigned, tileNum, ts.TileData, yPos, xPos, tileCol,
		// 		tileRow, scrollX, scrollY, windowX, windowY, pixel, colorBit, colorNum,
		// 	)
		// 	l.setTilePixel(pixel, scanline, tileAttr, colorNum, palette, priority)

		// }
		l.setTilePixel(pixel, scanline, tileAttr, colorNum, palette, priority)

	}
}

var oldlcdc = uint8(0)

type Set struct {
	list map[int]bool
}

func (s *Set) Add(value int) {
	s.list[value] = true
}

func (s *Set) Contains(value int) bool {
	_, ok := s.list[value]
	return ok
}

func (s *Set) Remove(value int) {
	delete(s.list, value)
}

func (s *Set) ToSlice() []int {
	keys := make([]int, 0, len(s.list))
	for key := range s.list {
		keys = append(keys, key)
	}
	return keys
}

func NewSet() *Set {
	return &Set{make(map[int]bool)}
}

var lys = NewSet()

// var beginLogging bool = false

func (l *LCD) setTilePixel(x, y, tileAttr, colorNum, palette uint8, priority bool) {
	l.tileScanline[x] = colorNum

	if l.Mb.Cgb {
		cgbPalette := tileAttr & 0x7
		r, g, b := l.Mb.BGPalette.get(cgbPalette, colorNum)
		l.setPixel(x, y, r, g, b, true)
		l.bgPriority[x][y] = priority
	} else {
		r, g, b := l.getColour(colorNum, palette)
		l.setPixel(x, y, r, g, b, true)
	}

}

func (l *LCD) setPixel(x, y, r, g, b uint8, priority bool) {
	if (priority && !l.bgPriority[x][y]) || l.tileScanline[x] == 0 {
		l.PreparedData[x][y][0] = r
		l.PreparedData[x][y][1] = g
		l.PreparedData[x][y][2] = b
	}
}

// Get the RGB colour value for a colour num at an address using the current palette.
func (l *LCD) getColour(colourNum byte, palette byte) (uint8, uint8, uint8) {
	hi := colourNum<<1 | 1
	lo := colourNum << 1
	index := (internal.BitValue(palette, hi) << 1) | internal.BitValue(palette, lo)
	r, g, b := GetPaletteColour(index)
	return r, g, b
}

func (l *LCD) renderSprites(lcdControl uint8, scanline int32) {
	var ySize int32 = 8

	if internal.IsBitSet(lcdControl, LCDC_OBJSZ) {
		ySize = 16

	}

	// Load the two palettes which sprites can be drawn in

	var palette1 = l.Mb.Memory.IO[IO_OBP0-IO_START_ADDR]
	var palette2 = l.Mb.Memory.IO[IO_OBP1-IO_START_ADDR]

	var minx [internal.GB_SCREEN_WIDTH]int32
	var lineSprites = 0
	for sprite := uint16(0); sprite < 40; sprite++ {
		// Load sprite data from memory.
		index := sprite * 4

		// If this is true the scanline is out of the area we care about
		yPos := int32(l.Mb.GetItem(uint16(0xFE00+index))) - 16
		if scanline < yPos || scanline >= (yPos+ySize) {
			continue
		}

		// Only 10 sprites are allowed to be displayed on each line
		if lineSprites >= 10 {
			break
		}
		lineSprites++

		xPos := int32(l.Mb.GetItem(uint16(0xFE00+index+1))) - 8
		tileLocation := l.Mb.GetItem(uint16(0xFE00 + index + 2))
		attributes := l.Mb.GetItem(uint16(0xFE00 + index + 3))

		yFlip := internal.IsBitSet(attributes, 6)
		xFlip := internal.IsBitSet(attributes, 5)
		priority := !internal.IsBitSet(attributes, 7)

		// Bank the sprite data in is (CGB only)
		var bank uint16 = 0
		if l.Mb.Cgb && internal.IsBitSet(attributes, 3) {
			bank = 1
		}

		// Set the line to draw based on if the sprite is flipped on the y
		line := scanline - yPos
		if yFlip {
			line = ySize - line - 1
		}

		// Load the data containing the sprite data for this line
		dataAddress := (uint16(tileLocation) * 16) + uint16(line*2) + (bank * 0x2000)
		// data1 := gb.Memory.VRAM[dataAddress]
		// data2 := gb.Memory.VRAM[dataAddress+1]
		data1 := l.Mb.Memory.Vram[bank][dataAddress-0x8000]
		data2 := l.Mb.Memory.Vram[bank][dataAddress+1-0x8000]

		// Draw the line of the sprite
		for tilePixel := byte(0); tilePixel < 8; tilePixel++ {
			pixel := int16(xPos) + int16(7-tilePixel)
			if pixel < 0 || pixel >= internal.GB_SCREEN_WIDTH {
				continue
			}

			// Check if the pixel has priority.
			//  - In DMG this is determined by the sprite with the smallest X coordinate,
			//    then the first sprite in the OAM.
			//  - In CGB this is determined by the first sprite appearing in the OAM.
			// We add a fixed 100 to the xPos so we can use the 0 value as the absence of a sprite.
			if minx[pixel] != 0 && (l.Mb.Cgb || minx[pixel] <= xPos+spritePriorityOffset) {
				continue
			}

			colourBit := tilePixel
			if xFlip {
				colourBit = byte(int8(colourBit-7) * -1)
			}

			// Find the colour value by combining the data bits
			// colourNum := (bits.Val(data2, colourBit) << 1) | bits.Val(data1, colourBit)
			colourNum := (internal.BitValue(data2, colourBit) << 1) | internal.BitValue(data1, colourBit)

			// Colour 0 is transparent for sprites
			if colourNum == 0 {
				continue
			}

			if l.Mb.Cgb {
				cgbPalette := attributes & 0x7
				red, green, blue := l.Mb.SpritePalette.get(cgbPalette, colourNum)
				l.setPixel(byte(pixel), byte(scanline), red, green, blue, priority)
			} else {
				// Determine the colour palette to use
				var palette = palette1
				if internal.IsBitSet(attributes, 4) {
					palette = palette2
				}
				red, green, blue := l.getColour(colourNum, palette)
				l.setPixel(byte(pixel), byte(scanline), red, green, blue, priority)
			}

			// Store the xpos of the sprite for this pixel for priority resolution
			minx[pixel] = xPos + spritePriorityOffset
		}
	}

}

func (l *LCD) clearScreen() {
	// if l.screenCleared {
	// 	return
	// }

	// set every pixel to white

	for x := 0; x < len(l.PreparedData); x++ {
		for y := 0; y < len(l.PreparedData[x]); y++ {
			l.PreparedData[x][y][0] = 0xFF
			l.PreparedData[x][y][1] = 0xFF
			l.PreparedData[x][y][2] = 0xFF
		}
	}

	// l.PreparedData = l.PreparedData
	l.screenCleared = true
}
