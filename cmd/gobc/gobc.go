package main

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/pixel/pixelgl"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/image/colornames"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/windows"
)

var logger = internal.Logger
var frameTick *time.Ticker
var g *windows.GoBoyColor

func setFPS(fps int) {
	if fps <= 0 {
		frameTick = nil
	} else {
		frameTick = time.NewTicker(time.Second / time.Duration(fps))
	}
}

func Update(wins []windows.Window) {
	// update gameboy state
	for _, w := range wins {
		w.Update()
	}
}

func Draw(wins []windows.Window) {
	for _, w := range wins {
		w.Draw()
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
	var elasped float64 = 0
	var frame_cntr int64 = 0

	var wins []windows.Window = []windows.Window{
		windows.NewMainGameWindow(g),
	}

	if DEBUG_WINDOWS {
		wins = append(wins, windows.NewMemoryViewWindow(g))
		wins = append(wins, windows.NewVramViewWindow(g))
	}

	mainWin := wins[0].Win()

	// run layout once
	for _, w := range wins {
		w.SetUp()
	}

	for !mainWin.Closed() {
		mainWin.SetTitle("gobc v0.1 | FPS: " + fmt.Sprintf("%.2f", fps))
		start := time.Now()
		mainWin.Clear(colornames.White)

		Update(wins)
		Draw(wins)

		if frameTick != nil {
			<-frameTick.C
		}

		elasped += float64(time.Since(start).Milliseconds())
		frame_cntr++

		if frame_cntr == 50 {
			fps = 1000.0 / (elasped / 50.0)
			frame_cntr = 0
			elasped = 0
		}
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
