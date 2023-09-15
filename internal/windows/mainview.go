package windows

import (
	"fmt"
	"image/color"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/spf13/afero"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

const (
	CyclesFrameDMG = internal.DMG_CLOCK_SPEED / internal.FRAMES_PER_SECOND
	CyclesFrameCBG = internal.CGB_CLOCK_SPEED / internal.FRAMES_PER_SECOND
)

var (
	internalCycleCounter int
	internalCycleReturn  motherboard.OpCycles
	internalStatus       bool
	internalGamePaused   bool = false
	internalConsoleTxt   *text.Text
)

type MainGameWindow struct {
	hw             *GoBoyColor
	Window         *pixelgl.Window
	gameScale      int
	gameTrueWidth  float64
	gameTrueHeight float64
	gameMapCanvas  *pixel.PictureData
	cyclesFrame    int
}

func (mw *MainGameWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, mw.gameTrueWidth, mw.gameTrueHeight))
	internalConsoleTxt = text.New(pixel.V(mw.Window.Bounds().Center().X/2, mw.Window.Bounds().Max.Y-20), text.NewAtlas(basicfont.Face7x13, text.ASCII))
	internalConsoleTxt.Color = colornames.Red
}

// this will get called every frame
// every frame must be called 1 / GB_CLOCK_HZ times in order to run the emulator at the correct speed
func (mw *MainGameWindow) Update() error {

	if mw.Window.JustPressed(pixelgl.KeyR) || mw.Window.Repeated(pixelgl.KeyR) {
		mw.hw.Reset()
	}

	if mw.Window.JustPressed(pixelgl.KeySpace) || mw.Window.Repeated(pixelgl.KeySpace) {
		internalGamePaused = !internalGamePaused
	}

	if (mw.Window.JustPressed(pixelgl.KeyN) || mw.Window.Repeated(pixelgl.KeyN)) && internalGamePaused {
		mw.hw.UpdateInternalGameState(1) // update every tick
	}

	if !internalGamePaused {
		if !mw.hw.UpdateInternalGameState(mw.cyclesFrame) {
			return nil
		}

		// tileMap := mw.hw.Mb.Memory.TileMap()
		// updatePicture(256, 256, 8, 8, &tileMap, mw.gameMapCanvas)
	}

	for y := 0; y < internal.GB_SCREEN_HEIGHT; y++ {
		for x := 0; x < internal.GB_SCREEN_WIDTH; x++ {
			col := mw.hw.Mb.Lcd.PreparedData[x][y]
			rgb := color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}
			mw.gameMapCanvas.Pix[(internal.GB_SCREEN_HEIGHT-1-y)*internal.GB_SCREEN_WIDTH+x] = rgb
		}
	}

	return nil
}

func (mw *MainGameWindow) Draw() {
	mw.Window.Clear(colornames.Black)

	if internalGamePaused {
		internalConsoleTxt.Clear()
		fmt.Fprint(internalConsoleTxt, "Game Paused")
		internalConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(internalConsoleTxt.Orig, 2))
	}

	// drawSprite(mw.Window, mw.gameMapCanvas, 1.5, 0, 0)
	r, g, b := motherboard.GetPaletteColour(3)
	bg := color.RGBA{R: r, G: g, B: b, A: 0xFF}
	mw.Window.Clear(bg)

	spr := pixel.NewSprite(pixel.Picture(mw.gameMapCanvas), pixel.R(0, 0, internal.GB_SCREEN_WIDTH, internal.GB_SCREEN_HEIGHT))
	spr.Draw(mw.Window, pixel.IM.Moved(mw.Window.Bounds().Center()).Scaled(mw.Window.Bounds().Center(), float64(mw.gameScale)))
	mw.Window.Update()
}

func (mw *MainGameWindow) Win() *pixelgl.Window {
	return mw.Window
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

func (g *GoBoyColor) Reset() {
	g.Mb.Reset()
	g.Stopped = false
	g.Paused = false
}

func NewMainGameWindow(gobc *GoBoyColor) *MainGameWindow {
	gameScale := 3
	gameScreenWidth := internal.GB_SCREEN_WIDTH
	gameScreenHeight := internal.GB_SCREEN_HEIGHT
	cyclesFrame := CyclesFrameDMG

	if gobc.Mb.Cgb {
		logger.Infof("Game is CGB, setting cycles per frame to %d", CyclesFrameCBG)
		cyclesFrame = CyclesFrameCBG
	}

	mgw := &MainGameWindow{
		hw:             gobc,
		gameScale:      gameScale,
		gameTrueWidth:  float64(gameScreenWidth * gameScale),
		gameTrueHeight: float64(gameScreenHeight * gameScale),
		gameMapCanvas:  pixel.MakePictureData(pixel.R(0, 0, float64(256), float64(256))),
		cyclesFrame:    cyclesFrame,
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
func (g *GoBoyColor) UpdateInternalGameState(every int) bool {
	if g.Stopped {
		return false
	}

	var still_good bool = true

	internalCycleCounter = 0
	for internalCycleCounter < every {
		// logger.Debug("----------------Tick-----------------")
		// if !g.Mb.BootRomEnabled() {
		// 	internalGamePaused = true
		// }
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
