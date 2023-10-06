package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	// putils "github.com/dusk125/pixelutils"
	"github.com/duysqubix/pixel2/pixelgl"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
	"github.com/duysqubix/gobc/internal/windows"
)

var logger = internal.Logger
var frameTick *time.Ticker
var g *windows.GoBoyColor

var frameRateMicro int64 = 16670

// var frameRateMicro int64 = 16000

func init() {
	runtime.LockOSThread()
}

func setFPS(fps int) {
	if fps <= 0 {
		frameTick = nil
	} else {
		ms := 1.0 / float64(fps) * 1000.0
		logger.Infof("Setting FPS to %d (%.2f ms)", fps, ms)
		dur := time.Duration(ms) * time.Millisecond
		frameTick = time.NewTicker(dur)
	}
}

var DEBUG_WINDOWS bool = false
var SHOW_GUI bool = true

func gameLoopGUI() {
	setFPS(internal.FRAMES_PER_SECOND)

	if g == nil {
		logger.Fatal("GoBoyColor core is not initialized")
	}

	var fps float64
	var elasped int64 = 0

	var wins []windows.Window = []windows.Window{
		windows.NewMainGameWindow(g),
	}

	if DEBUG_WINDOWS {
		wins = append(wins,
			windows.NewVramViewWindow(g),
			windows.NewMemoryViewWindow(g),
			windows.NewCartViewWindow(g),
			windows.NewCpuViewWindow(g),
			windows.NewIoViewWindow(g),
		)
	}

	mainWin := wins[0].Win()

	// run layout once
	for _, w := range wins {
		w.SetUp()
	}

	for !mainWin.Closed() {
		mainWin.SetTitle(fmt.Sprintf("gobc v%s | %s | FPS: %.2f", internal.VERSION, g.Mb.Cartridge.Filename, fps))
		elasped = 0
		start := time.Now()

		for _, w := range wins {
			w.Update()
			w.Draw()
		}

		// update internal GLFW events
		for _, w := range wins {
			w.Finalize()
		}

		elasped += time.Since(start).Microseconds()

		if elasped < frameRateMicro {
			time.Sleep(time.Duration(frameRateMicro-elasped) * time.Microsecond)
		}

		fps = 1000000.0 / float64(time.Since(start).Microseconds())
	}
}

func gameLoop() {
	setFPS(internal.FRAMES_PER_SECOND)

	if g == nil {
		logger.Fatal("GoBoyColor core is not initialized")
	}

	cyclesFrame := windows.CyclesFrameDMG

	if g.Mb.Cgb {
		logger.Infof("Game is CGB, setting cycles per frame to %d", windows.CyclesFrameCBG)
		cyclesFrame = windows.CyclesFrameCBG
	}

	for {

		if !g.UpdateInternalGameState(cyclesFrame) {
			break
		}

		if frameTick != nil {
			<-frameTick.C
		}

	}
}

func mainAction(ctx *cli.Context) error {
	var force_cgb bool = false
	var panicOnStuck bool = false
	var randomize bool = false
	var force_dmg bool = false

	if ctx.Bool("force-cgb") {
		force_cgb = true
	}

	if ctx.Bool("force-dmg") {
		force_dmg = true
	}

	if !ctx.Args().Present() {
		cli.ShowAppHelpAndExit(ctx, 1)
	}

	if ctx.Bool("panic-on-stuck") {
		panicOnStuck = true
	}

	if ctx.Bool("randomize") {
		randomize = true
	}

	var breakpoints []uint16
	if ctx.String("breakpoints") != "" {
		breakpoints = windows.ParseBreakpoints(ctx.String("breakpoints"))

	}

	romfile := ctx.Args().First()
	g = windows.NewGoBoyColor(romfile, breakpoints, force_cgb, force_dmg, panicOnStuck, randomize)

	if ctx.Bool("debug") {
		// logger.SetLevel(log.DebugLevel)
		DEBUG_WINDOWS = true
	}

	if ctx.Bool("no-gui") {
		SHOW_GUI = false
	}

	if SHOW_GUI {
		pixelgl.Run(gameLoopGUI)
	} else {
		gameLoop()
	}

	// save SRAM state
	cartridge.SaveSRAM(romfile, &g.Mb.Cartridge.RamBanks, g.Mb.Cartridge.RamBankCount)

	// default save state
	// internal.StateToFile(g.Mb.Cartridge.Filename, g.Mb)
	return cli.Exit("", 0)

}

func main() {
	app := &cli.App{
		Name:      "gobc",
		Version:   "0.0.1",
		Compiled:  time.Now(),
		Usage:     "A Gameboy emulator written in Go",
		UsageText: "gobc ROM_File [options] ",
		Authors: []*cli.Author{
			{
				Name:  "duys",
				Email: "duys@qubixds.com",
			},
		},
		Action: func(cCtx *cli.Context) error {
			return mainAction(cCtx)
		},

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Value: false,
				Usage: "Enable debug mode",
			},
			&cli.StringFlag{
				Name:  "breakpoints",
				Usage: "Define breakpoints",
			},
			&cli.BoolFlag{
				Name:  "force-cgb",
				Usage: "Force CGB mode",
			},

			&cli.BoolFlag{
				Name:  "force-dmg",
				Usage: "Force DMG mode",
			},
			&cli.BoolFlag{
				Name:  "no-gui",
				Usage: "Run without GUI",
			},
			&cli.BoolFlag{
				Name:  "panic-on-stuck",
				Usage: "Panic when CPU is stuck",
			},
			&cli.BoolFlag{
				Name:  "randomize",
				Usage: "Randomize RAM on startup",
			},
		},
	}

	var cmdsMessage string = `
	Commands: 
		Arrow Up:                    Up
		Arrow Down:                  Down
		Arrow Left:                  Left
		Arrow Right:                 Right
		S:                           A
		A:                           B
		Enter:                       Start
		Backspace:                   Select

	FKeys:
		F1:                          Toggle Grid
		F2:                          Toggle DebugMode
		F3:                          Cycle Color Palette (DMG Only)
	
	Debug Commands:
		Space:                       Game Pause
		F:                           1x Step Frame
		B:                           10x Step Frame
		N:                           nX Step Frame (Based on Value set by B,M)
		M:                           -10x Step Frame


	`
	fmt.Println(cmdsMessage)
	if err := app.Run(os.Args); err != nil {

		log.Fatal(err)
	}
}
