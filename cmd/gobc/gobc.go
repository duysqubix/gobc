package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	// putils "github.com/dusk125/pixelutils"
	"github.com/faiface/pixel/pixelgl"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/windows"
)

var logger = internal.Logger
var frameTick *time.Ticker
var g *windows.GoBoyColor

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

// var ticker = putils.NewTicker(internal.FRAMES_PER_SECOND)

func gameLoopGUI() {
	setFPS(internal.FRAMES_PER_SECOND)

	if g == nil {
		logger.Fatal("GoBoyColor core is not initialized")
	}

	var fps float64
	var elasped float64 = 0

	var wins []windows.Window = []windows.Window{
		windows.NewMainGameWindow(g),
	}

	if DEBUG_WINDOWS {
		wins = append(wins,
			windows.NewVramViewWindow(g),
			windows.NewMemoryViewWindow(g),
			windows.NewCartViewWindow(g),
		)
	}

	mainWin := wins[0].Win()

	// run layout once
	for _, w := range wins {
		w.SetUp()
	}

	for !mainWin.Closed() {
		mainWin.SetTitle("gobc v0.1 | FPS: " + fmt.Sprintf("%.2f", fps))
		elasped = 0
		start := time.Now()

		for _, w := range wins {
			w.Update()
			w.Draw()
		}

		if frameTick != nil {
			<-frameTick.C
		}

		elasped += float64(time.Since(start).Milliseconds())

		fps = 1000.0 / elasped

	}
}

func gameLoop() {
	setFPS(internal.FRAMES_PER_SECOND)

	if g == nil {
		logger.Fatal("GoBoyColor core is not initialized")
	}

	for {

		if !g.UpdateInternalGameState() {
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

	if ctx.Bool("force-cgb") {
		logger.Panic("Force CGB is not implemented yet")
		force_cgb = true
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
		// logger.Errorf("Breakpoints: %02x", breakpoints)

	}

	romfile := ctx.Args().First()
	g = windows.NewGoBoyColor(romfile, breakpoints, force_cgb, panicOnStuck, randomize)

	if ctx.Bool("debug") {
		logger.SetLevel(log.DebugLevel)
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

	if err := app.Run(os.Args); err != nil {

		log.Fatal(err)
	}
}
