package main

import (
	"os"
	"time"

	"github.com/chigopher/pathlib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/urfave/cli/v2"
)

var logger = internal.Logger

func init() {
}

type Gobc struct {
	// BootRomFile string // Boot ROM filename
	Mb          *motherboard.Motherboard
	Stopped     bool
	Paused      bool
	Breakpoints [2]uint16 // holds start and end address of breakpoint
}

func NewGobc(romfile string) *Gobc {
	// read cartridge first

	gobc := &Gobc{
		Mb: motherboard.NewMotherboard(&motherboard.MotherboardParams{
			Filename:  pathlib.NewPathAfero(romfile, afero.NewOsFs()),
			Randomize: true,
			Cbg:       false,
		}),
		Stopped: false,
		Paused:  false,
	}
	return gobc
}

func (g *Gobc) Tick() bool {
	if g.Stopped {
		return false
	}
	time.Sleep(1 * time.Millisecond)
	logger.Debug("Tick")
	return true
}

func (g *Gobc) Stop() {
	logger.Info("#########################")
	logger.Info("# Stopping Emulator.... #")
	logger.Info("#########################")
	g.Mb.Cpu.Stopped = true
	g.Stopped = true
}

func MainAction(ctx *cli.Context) error {
	if !ctx.Args().Present() {
		cli.ShowAppHelpAndExit(ctx, 1)
	}

	if ctx.Bool("verbose") {
		logger.SetLevel(log.InfoLevel)
		logger.Debugf("Verbose enabled")
	}

	if ctx.Bool("debug") {
		logger.SetLevel(log.DebugLevel)
		logger.Debugf("Debugging enabled")
	}

	romfile := ctx.Args().First()
	gobc := NewGobc(romfile)

	for gobc.Tick() {
		if !gobc.Paused {
			if !gobc.Mb.Tick() {
				gobc.Stopped = true
			}
		}
	}

	gobc.Stop()

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
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
