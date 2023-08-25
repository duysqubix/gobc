package windows

import (
	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel/pixelgl"

	// "github.com/hajimehoshi/ebiten/v2"
	"github.com/spf13/afero"
)

const (
	CyclesFrameSBG = internal.DMG_CLOCK_SPEED / internal.FRAMES_PER_SECOND
	CyclesFrameCBG = internal.CGB_CLOCK_SPEED / internal.FRAMES_PER_SECOND
)

var (
	internalCycleCounter int
	internalCycleReturn  motherboard.OpCycles
	internalStatus       bool

	// profiling stuff
	totalProcessedCycles int64
)

type GoBoyColor struct {
	// BootRomFile string // Boot ROM filename
	Mb           *motherboard.Motherboard
	Stopped      bool
	Paused       bool
	DebugMode    bool
	Breakpoints  [2]uint16 // holds start and end address of breakpoint
	DebugWindows map[string]*pixelgl.Window
}

func NewGoBoyColor(romfile string, breakpoints []uint16, force_cbg bool) *GoBoyColor {
	// read cartridge first

	gobc := &GoBoyColor{
		Mb: motherboard.NewMotherboard(&motherboard.MotherboardParams{
			Filename:    pathlib.NewPathAfero(romfile, afero.NewOsFs()),
			Randomize:   true,
			Breakpoints: breakpoints,
			ForceCbg:    force_cbg,
		}),
		Stopped:      false,
		Paused:       false,
		DebugWindows: make(map[string]*pixelgl.Window),
	}
	return gobc
}

// will want to block on this for CYCLES/60 cycles to process before rendering graphics
func (g *GoBoyColor) updateInternalGameState() bool {
	if g.Stopped {
		return false
	}

	var still_good bool

	internalCycleCounter = 0
	for internalCycleCounter < CyclesFrameSBG {
		still_good = g.Tick()
		if !g.Paused {
			internalStatus, internalCycleReturn = g.Mb.Tick()
			internalCycleCounter += int(internalCycleReturn)

			if !internalStatus {
				g.Stopped = true
				break
			}
		}
	}
	totalProcessedCycles += int64(internalCycleCounter)
	// fmt.Println("Total Cycles Processed: ", totalProcessedCycles)
	if !still_good {
		g.Stop()
	}
	return still_good
}

// this will get called every frame
// every frame must be called 1 / GB_CLOCK_HZ times in order to run the emulator at the correct speed
func (g *GoBoyColor) Update() error {

	if !g.updateInternalGameState() {
		return nil
	}
	return nil
}

func (g *GoBoyColor) Draw() {

}

func (g *GoBoyColor) Tick() bool {
	if g.Stopped {
		return false
	}
	logger.Debug("----------------Tick-----------------")
	return true
}

func (g *GoBoyColor) Stop() {
	logger.Info("#########################")
	logger.Info("# Stopping Emulator.... #")
	logger.Info("#########################")
	g.Mb.Cpu.Stopped = true
	g.Stopped = true
}
