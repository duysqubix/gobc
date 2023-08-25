package windows

import (
	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/spf13/afero"
	"golang.org/x/image/colornames"
)

const (
	CyclesFrameSBG = internal.DMG_CLOCK_SPEED / internal.FRAMES_PER_SECOND
	CyclesFrameCBG = internal.CGB_CLOCK_SPEED / internal.FRAMES_PER_SECOND
)

var (
	gameScale        = 1
	gameScreenWidth  = internal.GB_SCREEN_WIDTH
	gameScreenHeight = internal.GB_SCREEN_HEIGHT

	gameTrueWidth  = float64(gameScreenWidth * gameScale)
	gameTrueHeight = float64(gameScreenHeight * gameScale)

	internalCycleCounter int
	internalCycleReturn  motherboard.OpCycles
	internalStatus       bool

	// profiling stuff
	totalProcessedCycles int64
)

type MainGameWindow struct {
	hw     *GoBoyColor
	Window *pixelgl.Window
}

// this will get called every frame
// every frame must be called 1 / GB_CLOCK_HZ times in order to run the emulator at the correct speed
func (mw *MainGameWindow) Update() error {

	if !mw.hw.updateInternalGameState() {
		return nil
	}
	return nil
}

func (mw *MainGameWindow) Draw() {
	mw.Window.Clear(colornames.White)
	mw.Window.Update()
}

func (mw *MainGameWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *MainGameWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, gameTrueWidth, gameTrueHeight))
}

type GoBoyColor struct {
	// BootRomFile string // Boot ROM filename
	Mb          *motherboard.Motherboard
	Stopped     bool
	Paused      bool
	DebugMode   bool
	Breakpoints [2]uint16 // holds start and end address of breakpoint
	ForceCgb    bool
}

func NewGoBoyColor(romfile string, breakpoints []uint16, force_cgb bool) *GoBoyColor {
	// read cartridge first

	gobc := &GoBoyColor{
		Mb: motherboard.NewMotherboard(&motherboard.MotherboardParams{
			Filename:    pathlib.NewPathAfero(romfile, afero.NewOsFs()),
			Randomize:   true,
			Breakpoints: breakpoints,
			ForceCgb:    force_cgb,
		}),
		Stopped: false,
		Paused:  false,
	}
	return gobc
}

func NewMainGameWindow(gobc *GoBoyColor) *MainGameWindow {

	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "gobc v0.1 | Main Game Window",
		Bounds: pixel.R(0, 0, gameTrueWidth, gameTrueHeight),
		VSync:  true,
	})

	if err != nil {
		// logger.Panicf("Failed to create window: %s", err)
		panic(err)
	}

	return &MainGameWindow{
		Window: win,
		hw:     gobc,
	}
}

// will want to block on this for CYCLES/60 cycles to process before rendering graphics
func (g *GoBoyColor) updateInternalGameState() bool {
	if g.Stopped {
		return false
	}

	var still_good bool

	internalCycleCounter = 0
	for internalCycleCounter < CyclesFrameSBG {
		still_good = g.Tick()
		if !g.Paused {
			internalStatus, internalCycleReturn = g.Mb.Tick()
			internalCycleCounter += int(internalCycleReturn)

			if !internalStatus {
				g.Stopped = true
				break
			}
		}
	}
	totalProcessedCycles += int64(internalCycleCounter)
	// fmt.Println("Total Cycles Processed: ", totalProcessedCycles)
	if !still_good {
		g.Stop()
	}
	return still_good
}

func (g *GoBoyColor) Tick() bool {
	if g.Stopped {
		return false
	}
	logger.Debug("----------------Tick-----------------")
	return true
}

func (g *GoBoyColor) Stop() {
	logger.Info("#########################")
	logger.Info("# Stopping Emulator.... #")
	logger.Info("#########################")
	g.Mb.Cpu.Stopped = true
	g.Stopped = true
}
