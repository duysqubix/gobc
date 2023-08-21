package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/duysqubix/gobc/internal"
	"github.com/urfave/cli/v2"
)

var logger = internal.Logger

func init() {
}

func MainAction(ctx *cli.Context) error {
	if !ctx.Args().Present() {
		cli.ShowAppHelpAndExit(ctx, 1)
	}

	if ctx.Bool("debug") {
		logger.SetLevel(log.DebugLevel)
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
		Action: MainAction,

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "debug",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
