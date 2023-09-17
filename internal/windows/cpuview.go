package windows

import (
	"fmt"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

const (
	cpuScreenWidth  = 420
	cpuScreenHeight = 500
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
		Title:       "gobc v0.1 | Cpu View",
		Bounds:      pixel.R(0, 0, 670, 1000),
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
		pixel.V(10, mw.Window.Bounds().Max.Y-40),
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

func (mw *CpuViewWindow) Update() error {

	return nil
}

func (mw *CpuViewWindow) Draw() {
	cpuConsoleTxt.Clear()
	mw.Window.Clear(colornames.Black)
	cpuTableWriter.ClearRows()

	var data [][]string
	// print rows from memory
	for i := 0; i < 7; i++ {
		switch i {
		case 0: // AF flags
			data = append(data, []string{"A", fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.A), fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.F), "F"})
		case 1: // BC
			data = append(data, []string{"B", fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.B), fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.C), "C"})
		case 2: // DE
			data = append(data, []string{"D", fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.D), fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.E), "E"})
		case 3: // HL
			data = append(data, []string{"H", fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.H), fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.L), "L"})
		case 4: // SP
			data = append(data, []string{"SPH", fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.SP>>8), fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.SP&0xff), "SPL"})
		case 5: // PC
			data = append(data, []string{"PCH", fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.PC>>8), fmt.Sprintf("%#x", mw.hw.Mb.Cpu.Registers.PC&0xff), "PCL"})
		case 6: // IE & IF
			data = append(data, []string{"IE", fmt.Sprintf("0b%08b", mw.hw.Mb.Cpu.Interrupts.IE), fmt.Sprintf("0b%08b", mw.hw.Mb.Cpu.Interrupts.IF), "IF"})
		}
	}

	for _, d := range data {
		// fmt.Println(d)
		cpuTableWriter.Append(d)
	}
	cpuTableWriter.Render()

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
	for i := uint16(0); i < 3; i++ {
		fpc := mw.hw.Mb.Cpu.Registers.PC + i
		fmt.Fprintf(cpuConsoleTxt, "PC+%d: %04x (%02x)\n", i, fpc, mw.hw.Mb.GetItem(fpc))
	}

	cpuConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(cpuConsoleTxt.Orig, 1.5))

	mw.Window.Update()
}
