package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chigopher/pathlib"
	pixelgl "github.com/gopxl/pixel/v2/backends/opengl"
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

var SHOW_GUI bool = true

func openDebugWindows() []windows.Window {
	return []windows.Window{
		windows.NewVramViewWindow(g),
		windows.NewMemoryViewWindow(g),
		windows.NewCartViewWindow(g),
		windows.NewCpuViewWindow(g),
		windows.NewIoViewWindow(g),
	}
}

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

	debugWinsCreated := false
	if windows.IsDebugInfo() {
		extra := openDebugWindows()
		wins = append(wins, extra...)
		debugWinsCreated = true
	}

	mainWin := wins[0].Win()

	for _, w := range wins {
		w.SetUp()
	}

	for !mainWin.Closed() {
		if windows.IsDebugInfo() && !debugWinsCreated {
			extra := openDebugWindows()
			for _, w := range extra {
				w.SetUp()
			}
			wins = append(wins, extra...)
			debugWinsCreated = true
		}

		mainWin.SetTitle(fmt.Sprintf("gobc v%s | %s | FPS: %.2f", internal.VERSION, g.Mb.Cartridge.Filename, fps))
		elasped = 0
		start := time.Now()

		for _, w := range wins {
			w.Update()
			w.Draw()
		}

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

func runAction(ctx *cli.Context) error {
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
		cli.ShowAppHelpAndExit(ctx, 0)
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
		windows.SetDebugInfo(true)
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
	cartridge.SaveSRAM(g.Mb.Cartridge.GetFilename(), &g.Mb.Cartridge.RamBanks, g.Mb.Cartridge.RamBankCount)

	// default save state
	return cli.Exit("", 0)

}

func cartdumpAction(ctx *cli.Context) error {
	if !ctx.Args().Present() {
		return cli.Exit("error: ROM file required. Usage: gobc cartdump ROM_File [options]", 1)
	}

	filename := ctx.Args().First()
	obj := pathlib.NewPath(filename)

	isFile, err := obj.IsFile()
	if err != nil {
		return cli.Exit(fmt.Sprintf("error reading %q: %v", filename, err), 1)
	}
	if !isFile {
		return cli.Exit(fmt.Sprintf("error: %q is not a regular file", filename), 1)
	}

	supportedRoms := []string{".gbc", ".gb"}
	ext := filepath.Ext(filename)
	if !internal.IsInStrArray(ext, supportedRoms) {
		return cli.Exit(fmt.Sprintf("error: unsupported ROM extension %q (want .gb or .gbc)", ext), 1)
	}

	fmt.Println("Reading ROM file:", filename)
	cart := cartridge.NewCartridge(obj)

	if ctx.Bool("raw") {
		cart.RawHeaderDump()
		return nil
	}

	output := ctx.String("output")
	if output == "" {
		output = "cartdump.txt"
	}

	file, err := os.Create(output)
	if err != nil {
		return cli.Exit(fmt.Sprintf("error: failed to create output file %q: %v", output, err), 1)
	}
	defer file.Close()

	cart.Dump(file)
	if ctx.Bool("instruction-set") {
		cart.DumpInstructionSet(file, ctx.Bool("include-nop"))
	}

	fmt.Printf("Dumped cartridge metadata to %s\n", output)
	return nil
}

// Must stay in sync with internal/windows/*.go input handlers.
const keybindingsHelp = `KEYBINDINGS:

   Main Game Window (always available):
     Up / Down / Left / Right      D-Pad
     S                             A button
     A                             B button
     Enter                         Start
     Right Shift                   Select
     R                             Reset emulator
     F1                            Toggle Grid overlay
     F2                            Toggle Debug Information (opens debug windows)
     F3                            Cycle Color Palette (DMG only)
     F4                            Save Cartridge SRAM to disk
     F5                            Save State
     F6                            Load State

   Main Game Window (debug mode only, --debug):
     Space                         Pause / Unpause emulation
     F                             Step 1 frame
     N                             Step N cycles (controlled by B / M)
     B                             Decrease cycles-per-step by 10
     M                             Increase cycles-per-step by 10

   VRAM Viewer Window:
     T                             Toggle Tile Addressing Mode
     B                             Toggle TileMap Addressing Mode
     G                             Toggle Grid overlay
     H                             Toggle Help overlay
     V                             Toggle VRAM bank 0 / 1 (CGB only)

   Memory Viewer Window:
     Up / Down                     Scroll by line
     Left / Right                  Page Up / Page Down
     Mouse Wheel                   Scroll

   Cartridge Viewer Window:
     Up / Down                     Scroll by line
     Left / Right                  Page Up / Page Down
     [ / ]                         Previous / Next RAM bank
     Mouse Wheel                   Scroll

ENVIRONMENT VARIABLES:
   LOG_LEVEL                       Set log verbosity: debug | info | warn | error

EXAMPLES:
   gobc roms/cpu_instrs.gb                            # shorthand: run a ROM
   gobc run roms/cpu_instrs.gb --debug                # run with debug windows
   gobc run roms/cpu_instrs.gb --breakpoints 0x100,0x200
   gobc run roms/pokemon.gb --force-cgb               # force CGB mode on a DMG ROM
   gobc run roms/blargg.gb --no-gui                   # headless (for test ROMs in CI)
   LOG_LEVEL=debug gobc run roms/zelda.gb             # raise log verbosity

   gobc cartdump roms/pokemon.gb                      # write cartdump.txt
   gobc cartdump roms/pokemon.gb --raw                # print raw header to stdout
   gobc cartdump roms/pokemon.gb --instruction-set --include-nop -o pokemon.txt
`

func main() {
	runFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug mode (opens VRAM / Memory / Cart / CPU / IO viewer windows)",
		},
		&cli.StringFlag{
			Name:  "breakpoints",
			Usage: "Comma-separated list of PC breakpoint addresses (e.g. 0x100,0x200)",
		},
		&cli.BoolFlag{
			Name:  "force-cgb",
			Usage: "Force CGB mode on a DMG ROM",
		},
		&cli.BoolFlag{
			Name:  "force-dmg",
			Usage: "Force DMG mode on a CGB ROM",
		},
		&cli.BoolFlag{
			Name:  "no-gui",
			Usage: "Run without GUI (headless, useful for test ROMs / CI)",
		},
		&cli.BoolFlag{
			Name:  "panic-on-stuck",
			Usage: "Panic when the CPU is detected as stuck",
		},
		&cli.BoolFlag{
			Name:  "randomize",
			Usage: "Randomize RAM contents on startup",
		},
	}

	app := &cli.App{
		Name:        "gobc",
		Version:     internal.VERSION,
		Compiled:    time.Now(),
		Usage:       "A GameBoy / GameBoy Color emulator written in Go",
		UsageText:   "gobc [global options] ROM_File           # shorthand for `gobc run ROM_File`\n   gobc [global options] command [command options] [arguments...]",
		Description: keybindingsHelp,
		Authors: []*cli.Author{
			{
				Name:  "duys",
				Email: "duys@qubixds.com",
			},
		},
		Action: runAction,
		Flags:  runFlags,
		Commands: []*cli.Command{
			{
				Name:      "run",
				Usage:     "Run a ROM file (default action if no subcommand is given)",
				UsageText: "gobc run ROM_File [options]",
				Description: "Boots the emulator with the given .gb / .gbc ROM. Without --no-gui this opens\n" +
					"the main game window; with --debug it also opens VRAM, Memory, Cart, CPU and IO\n" +
					"debugger windows. See `gobc --help` for the full keybinding reference.",
				Flags:  runFlags,
				Action: runAction,
			},
			{
				Name:      "cartdump",
				Usage:     "Dump cartridge header metadata (and optional disassembly) from a ROM",
				UsageText: "gobc cartdump ROM_File [--raw | --instruction-set [--include-nop]] [-o FILE]",
				Description: "Inspects a GameBoy ROM (.gb / .gbc) and writes its parsed cartridge header to a\n" +
					"text file. With --raw the raw header bytes are printed to stdout instead. With\n" +
					"--instruction-set the full disassembled instruction listing is appended to the\n" +
					"output file (use --include-nop to also emit NOP opcodes).",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "raw",
						Usage: "Print the raw header bytes to stdout instead of writing a file",
					},
					&cli.BoolFlag{
						Name:  "instruction-set",
						Usage: "Append the full instruction-set disassembly to the output file",
					},
					&cli.BoolFlag{
						Name:  "include-nop",
						Usage: "Include NOP instructions in the disassembly (requires --instruction-set)",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "cartdump.txt",
						Usage:   "Output file for the cartridge dump",
					},
				},
				Action: cartdumpAction,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
