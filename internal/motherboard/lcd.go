package motherboard

import (
	"github.com/duysqubix/gobc/internal"
)

type ScreenData [internal.GB_SCREEN_WIDTH][internal.GB_SCREEN_HEIGHT][3]uint8
type ScreenPriority [internal.GB_SCREEN_WIDTH][internal.GB_SCREEN_HEIGHT]bool

const (
	lcdMode2Bounds = 456 - 80
	lcdMode3Bounds = lcdMode2Bounds - 172
)



type LCD struct {

	// Matrix of pixel data which is used while the screen is rendering. When the screen is done rendering, this data is copied to the PreparedData matrix.
	screenData ScreenData
	bgPriority ScreenPriority

	tileScanline    [internal.GB_SCREEN_WIDTH]uint8
	scanlineCounter OpCycles
	screenCleared   bool

	// PreparedData is a matrix of screen pixel data for a single frame which has been fully rendered
	PreparedData ScreenData
	Mb           *Motherboard
}

func NewLCD(mb *Motherboard) *LCD {
	return &LCD{
		Mb: mb,
	}
}

func (l *LCD) Tick(cycles OpCycles) {
	l.updateGraphics(cycles)
}

func (l *LCD) updateGraphics(cycles OpCycles) {
	l.setLCDStatus()

	if !l.isLCDEnabled() {
		return
	}

	l.scanlineCounter -= cycles

	if l.scanlineCounter <= 0 {
		l.Mb.Memory.IO[IO_LY-IO_START_ADDR]++ // directly change for optimized performance
		if l.Mb.GetItem(IO_LY) > 153 {
			l.PreparedData = l.screenData
			l.screenData = ScreenData{}
			l.bgPriority = ScreenPriority{}
			l.Mb.SetItem(IO_LY, 0)
		}

		currentLine := l.Mb.GetItem(IO_LY)
		l.scanlineCounter += (456 * 1) // change 1 to 2 for double speed

		if currentLine < internal.GB_SCREEN_HEIGHT {
			l.Mb.Cpu.SetInterruptFlag(INTR_VBLANK)
		}
	}

}

func (l *LCD) setLCDStatus() {

	status := l.Mb.GetItem(IO_STAT)

	if !l.isLCDEnabled() {
		// clear the screen
		l.clearScreen()
		l.scanlineCounter = 456

		l.Mb.SetItem(IO_LY, 0)

		// reset status
		status &= (1 << STAT_LYCINT) |
			(1 << STAT_OAMINT) |
			(1 << STAT_VBLINT) |
			(1 << STAT_HBLINT) |
			(1 << STAT_LYC) |
			(0 << STAT_MODE1) |
			(0 << STAT_MODE0)

		// write status to memory
		// l.Mb.Memory.IO[IO_STAT-IO_START_ADDR] = status
		l.Mb.SetItem(IO_STAT, uint16(status))
	}

	l.screenCleared = false

	currentLine := l.Mb.GetItem(IO_LY)
	currentMode := status & 0x3

	var mode uint8
	rqstInterrupt := false

	switch {
	case currentLine >= 144:
		mode = STAT_MODE_VBLANK
		status |= (0 << STAT_MODE1) | (1 << STAT_MODE0)
		rqstInterrupt = internal.IsBitSet(status, STAT_VBLINT)

	case l.scanlineCounter >= lcdMode2Bounds:
		mode = STAT_MODE_OAM
		status |= (1 << STAT_MODE1) | (0 << STAT_MODE0)
		rqstInterrupt = internal.IsBitSet(status, STAT_OAMINT)

	case l.scanlineCounter >= lcdMode3Bounds:
		mode = STAT_MODE_TRANS
		status |= (1 << STAT_MODE1) | (1 << STAT_MODE0)
		if mode != currentMode {
			// draw scanline when we start mode 3. In the real gameboy
			// this would be done through mode 3 by readong OAM and VRAM
			// to generate the picture
			l.drawScanline()
		}
	default:
		mode = STAT_MODE_HBLANK
		status |= (0 << STAT_MODE1) | (0 << STAT_MODE0)
		rqstInterrupt = internal.IsBitSet(status, STAT_HBLINT)
		if mode != currentMode {
			l.Mb.DoHDMATransfer() // do HDMATransfer when we start mode 0
		}
	}

	if rqstInterrupt && mode != currentMode {
		l.Mb.Cpu.SetInterruptFlag(INTR_LCDSTAT)
	}

	// check if LYC == LY (coincedence flag)
	if currentLine == l.Mb.GetItem(IO_LYC) {
		internal.SetBit(&status, STAT_LYC)
		if internal.IsBitSet(status, STAT_LYCINT) {
			l.Mb.Cpu.SetInterruptFlag(INTR_LCDSTAT)
		}
	} else {
		internal.ResetBit(&status, STAT_LYC)
	}

	// write status to memory
	l.Mb.SetItem(IO_STAT, uint16(status))
}

func (l *LCD) isLCDEnabled() bool {
	return internal.IsBitSet(l.Mb.Memory.IO[IO_LCDC-IO_START_ADDR], LCDC_ENABLE)
}

func (l *LCD) drawScanline() {
}

func (l *LCD) clearScreen() {
	if l.screenCleared {
		return
	}

	// set every pixel to white

	for x := 0; x < len(l.screenData); x++ {
		for y := 0; y < len(l.screenData[x]); y++ {
			l.screenData[x][y][0] = 0xFF
			l.screenData[x][y][1] = 0xFF
			l.screenData[x][y][2] = 0xFF
		}
	}

	l.PreparedData = l.screenData
	l.screenCleared = true
}
