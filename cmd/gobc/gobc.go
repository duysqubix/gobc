package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
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

func NewGobc(romfile string, breakpoints []uint16, force_cbg bool) *Gobc {
	// read cartridge first

	gobc := &Gobc{
		Mb: motherboard.NewMotherboard(&motherboard.MotherboardParams{
			Filename:    pathlib.NewPathAfero(romfile, afero.NewOsFs()),
			Randomize:   true,
			Breakpoints: breakpoints,
			ForceCbg:    force_cbg,
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
	time.Sleep(1 * time.Nanosecond)
	logger.Debug("----------------Tick-----------------")
	return true
}

func (g *Gobc) Stop() {
	logger.Info("#########################")
	logger.Info("# Stopping Emulator.... #")
	logger.Info("#########################")
	g.Mb.Cpu.Stopped = true
	g.Stopped = true
}

func parseRangeBreakpoints(breakpoints string) []uint16 {
	// logger.Debug(breakpoints)
	var parsed []uint16
	start, err := strconv.ParseUint(strings.Split(breakpoints, ":")[0], 16, 16)
	if err != nil {
		errmsg := fmt.Sprintf("Invalid breakpoint format: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}
	end, err := strconv.ParseUint(strings.Split(breakpoints, ":")[1], 16, 16)
	if err != nil {
		errmsg := fmt.Sprintf("Invalid breakpoint format: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}

	for i := start; i <= end; i++ {
		parsed = append(parsed, uint16(i))
	}
	return parsed
}

func parseSingleBreakpoint(breakpoints string) uint16 {

	// logger.Debug(breakpoints)

	// single breakpoint
	addr, err := strconv.ParseUint(breakpoints, 16, 16)
	if addr > 0xffff {
		errmsg := fmt.Sprintf("Addr out of range: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}
	if err != nil {
		errmsg := fmt.Sprintf("Invalid breakpoint format: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}

	return uint16(addr)
}

func parseBreakpoints(breakpoints string) []uint16 {
	var a []uint16

	split := strings.Split(breakpoints, ",")
	// logger.Debug(split)

	if len(split) == 1 {
		if split[0] == "" {
			return a
		}
		// check if single element is a range
		is_range := strings.Split(split[0], ":")
		if len(is_range) == 2 {
			a = append(a, parseRangeBreakpoints(split[0])...)
		} else {
			// not a range so parse as single breakpoint
			a = append(a, parseSingleBreakpoint(split[0]))
		}
	}
	if len(split) > 1 {
		for _, b := range split {
			if b == "" {
				continue
			}
			// check if single element is a range
			is_range := strings.Split(b, ":")
			if len(is_range) == 2 {
				a = append(a, parseRangeBreakpoints(b)...)
			} else {
				// not a range so parse as single breakpoint
				a = append(a, parseSingleBreakpoint(b))
			}
		}
	}

	// now sort and remove duplicates
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })

	// Remove duplicates
	return removeDuplicates(a)

}

func removeDuplicates(a []uint16) []uint16 {
	j := 0
	for i := 1; i < len(a); i++ {
		if a[j] != a[i] {
			j++
			a[j] = a[i]
		}
	}
	result := a[:j+1]
	return result
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

	if ctx.Bool("debug") {
		logger.SetLevel(log.DebugLevel)
		logger.Debugf("Debugging enabled")
	}

	var breakpoints []uint16
	if ctx.String("breakpoints") != "" {
		breakpoints = parseBreakpoints(ctx.String("breakpoints"))
		// logger.Errorf("Breakpoints: %02x", breakpoints)

	}
	fmt.Println(force_cgb)
	romfile := ctx.Args().First()
	gobc := NewGobc(romfile, breakpoints, force_cgb)

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