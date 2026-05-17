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
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/duysqubix/gobc/internal/windows"
)

var logger = internal.Logger
var frameTick *time.Ticker
var g *windows.GoBoyColor

// 16742 μs per frame = 1 / 59.7275 Hz, the real Game Boy DMG V-Sync rate per
// Pan Docs (https://gbdev.io/pandocs/Rendering.html). The previous value of
// 16670 (1/60) made gobc try to run the emulator 0.46% faster than real
// hardware, over-producing samples by the same ratio.
var frameRateMicro int64 = 16742

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

	// PyBoy-style frame pacing: TARGET-time accumulator. Each iteration
	// advances `target` by an adaptive frame-time (0 when audio queue is
	// low → run free; ≈ frameRateMicro when audio is full → 60 FPS cap).
	// Then sleep until `target`. If `target` falls into the past (catch-
	// up from a low-buffer burst) we reset it to wall-clock to avoid
	// racing forward when the buffer refills.
	target := time.Now()

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
		start := time.Now()

		for _, w := range wins {
			w.Update()
			w.Draw()
		}

		for _, w := range wins {
			w.Finalize()
		}

		// PyBoy-style adaptive frame limiter.
		//
		//   audio queue > target depth   → advance target by full frame (60 Hz cap)
		//   audio queue ≤ target depth   → advance target by 0 (no cap, run free)
		//   audio disabled               → advance target by full frame (real-time)
		//
		// The accumulator-vs-now subtraction below produces a sleep equal
		// to (target - work_finish_time), so total wall time per frame is
		// max(work_time, frame_time) — never 2×.
		const audioPrebufferFrames = 5.0
		var frameInc time.Duration
		if g.Mb.Sound != nil && g.Mb.Sound.AudioEnabled() {
			framesBuffered := g.Mb.Sound.AudioQueueFramesBuffered()
			if framesBuffered > audioPrebufferFrames {
				overflow := framesBuffered - audioPrebufferFrames
				if overflow > 1.0 {
					overflow = 1.0
				}
				frameInc = time.Duration(overflow*float64(frameRateMicro)) * time.Microsecond
			}
			// else: leave frameInc = 0 → run free, refill queue
		} else {
			frameInc = time.Duration(frameRateMicro) * time.Microsecond
		}
		target = target.Add(frameInc)
		now := time.Now()
		if target.Before(now) {
			target = now
		} else if delay := target.Sub(now); delay > 0 {
			time.Sleep(delay)
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
	audioEnabled := !ctx.Bool("no-audio") && !ctx.Bool("no-gui")
	audioSmooth := ctx.Bool("audio-smooth")
	if rate := ctx.Int("audio-rate"); rate > 0 {
		motherboard.SetAudioSampleRateOverride(rate)
	}
	g = windows.NewGoBoyColor(romfile, breakpoints, force_cgb, force_dmg, panicOnStuck, randomize, audioEnabled, audioSmooth)

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
			Name:  "no-audio",
			Usage: "Disable audio output",
		},
		&cli.IntFlag{
			Name:  "audio-rate",
			Usage: "Audio sample rate in Hz (default 32000). Lower this if you hear audio chop on a slow CPU — match it to your actual sustained FPS × 533 (e.g. 57 FPS → 30400).",
			Value: 0,
		},
		&cli.BoolFlag{
			Name:  "audio-smooth",
			Usage: "Eliminate audio chop on slow CPUs by measuring host throughput at startup and matching the speaker rate to the producer rate. Costs a ~2% pitch drop on a 98%-speed host (about one third of a semitone — usually below the detectable threshold).",
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
