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
	internalCycleCounter int
	internalCycleReturn  motherboard.OpCycles
	internalStatus       bool
)

type MainGameWindow struct {
	hw             *GoBoyColor
	Window         *pixelgl.Window
	gameScale      int
	gameTrueWidth  float64
	gameTrueHeight float64
	gameMapCanvas  *pixel.PictureData
}

// this will get called every frame
// every frame must be called 1 / GB_CLOCK_HZ times in order to run the emulator at the correct speed
func (mw *MainGameWindow) Update() error {

	if !mw.hw.UpdateInternalGameState() {
		return nil
	}

	tileMap := mw.hw.Mb.Memory.TileMap()

	updatePicture(256, 256, 8, 8, &tileMap, mw.gameMapCanvas)

	return nil
}

func (mw *MainGameWindow) Draw() {
	mw.Window.Clear(colornames.Black)

	drawSprite(mw.Window, mw.gameMapCanvas, 1.5, 0, 0)
	// spr2 := pixel.NewSprite(mw.gameMapCanvas, mw.gameMapCanvas.Bounds())
	// spr2.Draw(mw.Window, pixel.IM.Moved(mw.Window.Bounds().Center()).Scaled(mw.Window.Bounds().Center(), 1))
	mw.Window.Update()
}

func (mw *MainGameWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *MainGameWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, mw.gameTrueWidth, mw.gameTrueHeight))
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

func NewGoBoyColor(romfile string, breakpoints []uint16, forceCgb bool, panicOnStuck bool, randomize bool) *GoBoyColor {
	// read cartridge first

	gobc := &GoBoyColor{
		Mb: motherboard.NewMotherboard(&motherboard.MotherboardParams{
			Filename:     pathlib.NewPathAfero(romfile, afero.NewOsFs()),
			Randomize:    randomize,
			Breakpoints:  breakpoints,
			ForceCgb:     forceCgb,
			PanicOnStuck: panicOnStuck,
		}),
		Stopped: false,
		Paused:  false,
	}
	return gobc
}

func NewMainGameWindow(gobc *GoBoyColor) *MainGameWindow {
	gameScale := 3
	gameScreenWidth := internal.GB_SCREEN_WIDTH
	gameScreenHeight := internal.GB_SCREEN_HEIGHT

	mgw := &MainGameWindow{
		hw:             gobc,
		gameScale:      gameScale,
		gameTrueWidth:  float64(gameScreenWidth * gameScale),
		gameTrueHeight: float64(gameScreenHeight * gameScale),
		gameMapCanvas:  pixel.MakePictureData(pixel.R(0, 0, float64(256), float64(256))),
	}

	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:       "gobc v0.1 | Main Game Window",
		Bounds:      pixel.R(0, 0, mgw.gameTrueWidth, mgw.gameTrueHeight),
		VSync:       true,
		AlwaysOnTop: true,
	})

	if err != nil {
		// logger.Panicf("Failed to create window: %s", err)
		panic(err)
	}
	mgw.Window = win

	return mgw
}

// will want to block on this for CYCLES/60 cycles to process before rendering graphics
func (g *GoBoyColor) UpdateInternalGameState() bool {
	if g.Stopped {
		return false
	}

	var still_good bool = true

	internalCycleCounter = 0
	for internalCycleCounter < CyclesFrameSBG {
		// logger.Debug("----------------Tick-----------------")

		if g.Stopped {
			still_good = false
			break
		}

		if !g.Paused {
			internalStatus, internalCycleReturn = g.Mb.Tick()
			internalCycleCounter += int(internalCycleReturn)

			if !internalStatus {
				g.Stopped = true
				break
			}
		}
	}
	// totalProcessedCycles += int64(internalCycleCounter)
	// fmt.Println("Total Cycles Processed: ", totalProcessedCycles)
	if !still_good {
		g.Stop()
	}
	return still_good
}

func (g *GoBoyColor) Stop() {
	logger.Info("#########################")
	logger.Info("# Stopping Emulator.... #")
	logger.Info("#########################")
	g.Mb.Cpu.Stopped = true
	g.Stopped = true
}
