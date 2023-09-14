package windows

import (
	"fmt"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

const (
	cpuScreenWidth  = 800
	cpuScreenHeight = 450
	cpuScale        = 1
	cpuTrueWidth    = float64(cpuScreenWidth * cpuScale)
	cpuTrueHeight   = float64(cpuScreenHeight * cpuScale)
	cpuFontBuffer   = 4
)

var (
	cpuDefaultFont *basicfont.Face
	cpuMaxRows     int
	cpuBeginAddr   int
	cpuEndAddr     int
	cpuAddrOffset  int = 10
	cpuConsoleTxt  *text.Text
	cpuTableWriter *tablewriter.Table
	cpuRambank     int = 0
)

func init() {

	// set up font
	cpuDefaultFont = basicfont.Face7x13

	// cpuMaxRows = int(cpuTrueHeight) / (cpuDefaultFont.Height + 2)
	cpuMaxRows = 32

	cpuBeginAddr = 0
	cpuEndAddr = (cpuMaxRows-cpuAddrOffset-1)*0x10 + cpuBeginAddr
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
		pixel.V(10, mw.Window.Bounds().Max.Y-20),
		text.NewAtlas(cpuDefaultFont, text.ASCII),
		// text.NewAtlas(inconsolata.Regular8x16, text.ASCII),
	)
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

	cpuConsoleTxt.Color = colornames.White

	var data [][]string
	// print rows from memory
	for i := 0; i < 6; i++ {
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
		}
	}

	for _, d := range data {
		// fmt.Println(d)
		cpuTableWriter.Append(d)
	}
	cpuTableWriter.Render()
	// fmt.Fprintf(cpuConsoleTxt, "YOffset: %f\n", mw.YOffset)
	cpuConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(cpuConsoleTxt.Orig, 1.25))

	mw.Window.Update()
}
