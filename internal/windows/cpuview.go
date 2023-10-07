package windows

import (
	"fmt"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
	pixel "github.com/gopxl/pixel/v2"
	"github.com/gopxl/pixel/v2/pixelgl"
	"github.com/gopxl/pixel/v2/text"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

const (
	cpuScreenWidth  = 420
	cpuScreenHeight = 600
	cpuScale        = 1
	cpuTrueWidth    = float64(cpuScreenWidth * cpuScale)
	cpuTrueHeight   = float64(cpuScreenHeight * cpuScale)
	cpuFontBuffer   = 4
)

var (
	cpuDefaultFont *basicfont.Face
	cpuConsoleTxt  *text.Text
	cpuTableWriter *tablewriter.Table
)

func init() {

	// set up font
	cpuDefaultFont = basicfont.Face7x13

}

type CpuViewWindow struct {
	hw      *GoBoyColor
	YOffset float64
	Window  *pixelgl.Window
}

func NewCpuViewWindow(gobc *GoBoyColor) *CpuViewWindow {
	/// create memory window
	memWin, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:       fmt.Sprintf("gobc v%s | CPU View", internal.VERSION),
		Bounds:      pixel.R(0, 0, cpuTrueWidth, cpuTrueHeight),
		VSync:       true,
		AlwaysOnTop: true,
		Resizable:   true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}
	return &CpuViewWindow{
		Window:  memWin,
		YOffset: 0,
		hw:      gobc,
	}
}

func (mw *CpuViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *CpuViewWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, cpuTrueWidth, cpuTrueHeight))
	cpuConsoleTxt = text.New(
		pixel.V(10, 5),
		text.NewAtlas(cpuDefaultFont, text.ASCII),
		// text.NewAtlas(inconsolata.Regular8x16, text.ASCII),
	)
	cpuConsoleTxt.Color = colornames.Red

	cpuTableWriter = tablewriter.NewWriter(cpuConsoleTxt)
	cpuTableWriter.SetAutoWrapText(false)
	cpuTableWriter.SetAlignment(tablewriter.ALIGN_LEFT)
	cpuTableWriter.SetBorder(true)
	cpuTableWriter.SetHeader([]string{"HR", "HV", "LV", "LR"})
}

func (mw *CpuViewWindow) Finalize() {
	mw.Window.Update()
}

func (mw *CpuViewWindow) Update() error {

	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (mw *CpuViewWindow) Draw() {
	cpuConsoleTxt.Clear()
	mw.Window.Clear(colornames.Black)
	// cpuTableWriter.ClearRows()

	template := `
-------------------------
|     Z = %1d  | N = %1d    |
|     H = %1d  | C = %1d    |
-------------------------
| A = $%02x    | F = $%02x  |
| B = $%02x    | C = $%02x  |
| D = $%02x    | E = $%02x  |
| H = $%02x    | L = $%02x  |
-------------------------
|      SP = $%04x       |
|      PC = $%04x       |
-------------------------
| IME=%1d    | HALT=%1d     |
-------------------------
| DOUBLE SPEED = %3s    |
-------------------------
| BOOTROM = %3s         |
-------------------------

`
	fmt.Fprintf(cpuConsoleTxt, template,
		internal.BitValue(mw.hw.Mb.Cpu.Registers.F, motherboard.FLAGZ),
		internal.BitValue(mw.hw.Mb.Cpu.Registers.F, motherboard.FLAGN),
		internal.BitValue(mw.hw.Mb.Cpu.Registers.F, motherboard.FLAGH),
		internal.BitValue(mw.hw.Mb.Cpu.Registers.F, motherboard.FLAGC),
		mw.hw.Mb.Cpu.Registers.A,
		mw.hw.Mb.Cpu.Registers.F,
		mw.hw.Mb.Cpu.Registers.B,
		mw.hw.Mb.Cpu.Registers.C,
		mw.hw.Mb.Cpu.Registers.D,
		mw.hw.Mb.Cpu.Registers.E,
		mw.hw.Mb.Cpu.Registers.H,
		mw.hw.Mb.Cpu.Registers.L,
		mw.hw.Mb.Cpu.Registers.SP,
		mw.hw.Mb.Cpu.Registers.PC,
		boolToInt(mw.hw.Mb.Cpu.Interrupts.InterruptsOn),
		boolToInt(mw.hw.Mb.Cpu.Halted),
		"N/A",
		"N/A",
	)

	cntr := mw.hw.Mb.Cpu.PcHist.Len() - 1
	for i := mw.hw.Mb.Cpu.PcHist.Back(); i != nil; i = i.Prev() {
		tup := i.Value.(motherboard.Tuple)
		var opCodeName string = ""
		if tup.IsOpCode {
			opCodeName = internal.OPCODE_NAMES[tup.OpCode]
		}

		fmt.Fprintf(cpuConsoleTxt, "PC-%d: %04x (%02x) [%s]\n", cntr+1, tup.Addr, mw.hw.Mb.GetItem(tup.Addr), opCodeName)
		cntr--
	}

	// look into the future by 5 steps
	for i := uint16(1); i < 3; i++ {
		fpc := mw.hw.Mb.Cpu.Registers.PC + i
		fmt.Fprintf(cpuConsoleTxt, "PC+%d: %04x (%02x)\n", i, fpc, mw.hw.Mb.GetItem(fpc))
	}

	cpuConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(cpuConsoleTxt.Orig, 1.5))
}
