package main

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/image/colornames"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/windows"
	// "github.com/duysqubix/gobc/internal/windows"
	// "github.com/urfave/cli/v2"
)

var logger = internal.Logger

var (
	gameWidth  = internal.GB_SCREEN_WIDTH
	gameHeight = internal.GB_SCREEN_HEIGHT
	scale      = 2
	frameTick  *time.Ticker
	g          *windows.GoBoyColor
)

func setFPS(fps int) {
	if fps <= 0 {
		frameTick = nil
	} else {
		frameTick = time.NewTicker(time.Second / time.Duration(fps))
	}
}

func Update(g windows.Window) {
	// update gameboy state
	g.Update()
}

func Draw(g windows.Window) {
	// draw gameboy state
	g.Draw()
}

func GameLoop() {
	setFPS(internal.FRAMES_PER_SECOND)

	windowHeight := float64(gameHeight * scale)
	windowWidth := float64(gameWidth * scale)
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "gobc v0.1",
		Bounds: pixel.R(0, 0, windowWidth, windowHeight),
		VSync:  true,
	})

	if err != nil {
		// logger.Panicf("Failed to create window: %s", err)
		panic(err)
	}

	var fps float64
	var elasped int64 = 0
	for !win.Closed() {
		win.SetTitle("gobc v0.1 | FPS: " + fmt.Sprintf("%.2f", fps))
		start := time.Now()
		win.Clear(colornames.White)

		Update(g)
		Draw(g)
		win.Update()

		if frameTick != nil {
			<-frameTick.C
		}
		elasped += time.Since(start).Milliseconds()
		fps = 1000 / float64(elasped)

		// fmt.Println(elasped)
	}
}

func MainAction(ctx *cli.Context) error {
	var force_cgb bool = false

	if ctx.Bool("force-cgb") {
		force_cgb = true
	}

	if !ctx.Args().Present() {
		cli.ShowAppHelpAndExit(ctx, 1)
	}

	if ctx.Bool("verbose") {
		logger.SetLevel(log.InfoLevel)
		logger.Debugf("Verbose enabled")
	}

	var breakpoints []uint16
	if ctx.String("breakpoints") != "" {
		breakpoints = windows.ParseBreakpoints(ctx.String("breakpoints"))
		// logger.Errorf("Breakpoints: %02x", breakpoints)

	}

	romfile := ctx.Args().First()
	g = windows.NewGoBoyColor(romfile, breakpoints, force_cgb)

	if ctx.Bool("debug") {
		// spin up Memory Window and show ROMs Memory Map
		// g.Debug_MemoryView = windows.NewMemoryViewWindow(gobc)
		// gobc.DebugMode = true
		logger.SetLevel(log.DebugLevel)
	}

	pixelgl.Run(GameLoop)

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
			return MainAction(cCtx)
		},

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Value: false,
			},
			&cli.BoolFlag{
				Name: "verbose",
			},
			&cli.StringFlag{
				Name: "breakpoints",
			},
			&cli.BoolFlag{
				Name: "force-cgb",
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
