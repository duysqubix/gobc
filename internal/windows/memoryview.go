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
	memScreenWidth  = 800
	memScreenHeight = 1000
	memScale        = 1
	memTrueWidth    = float64(memScreenWidth * memScale)
	memTrueHeight   = float64(memScreenHeight * memScale)
	fontBuffer      = 4
)

var (
	defaultFont *basicfont.Face
	logger      = internal.Logger
	mock_memory []uint8
	max_rows    int
	// row_ptr     int = 1 // starting at the top of the screen
	begin_addr     int
	end_addr       int
	addr_offset    int = 10
	consoleTxt     *text.Text
	memTableWriter *tablewriter.Table
)

func init() {
	mock_memory = make([]uint8, 0xffff)

	for i := 0; i < 0xffff; i++ {
		mock_memory[i] = uint8(i)
	}

	// set up font
	defaultFont = basicfont.Face7x13

	max_rows = int(memTrueHeight) / (defaultFont.Height + 2)

	begin_addr = 0
	end_addr = (max_rows-addr_offset-1)*0x10 + begin_addr
}

type MemoryViewWindow struct {
	hw      *GoBoyColor
	YOffset float64
	Window  *pixelgl.Window
}

func NewMemoryViewWindow(gobc *GoBoyColor) *MemoryViewWindow {
	/// create memory window
	memWin, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "gobc v0.1 | Memory View",
		Bounds: pixel.R(0, 0, 670, 1000),
		VSync:  true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}
	return &MemoryViewWindow{
		Window:  memWin,
		YOffset: 0,
		hw:      gobc,
	}
}

func (mw *MemoryViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *MemoryViewWindow) SetUp() {
	mw.Window.SetTitle("gobc v0.1 | Memory View")
	mw.Window.SetBounds(pixel.R(0, 0, memTrueWidth, memTrueHeight))
	consoleTxt = text.New(
		pixel.V(10, mw.Window.Bounds().Max.Y-20),
		text.NewAtlas(defaultFont, text.ASCII),
		// text.NewAtlas(inconsolata.Regular8x16, text.ASCII),
	)
	memTableWriter = tablewriter.NewWriter(consoleTxt)
	memTableWriter.SetAutoWrapText(false)
	memTableWriter.SetAlignment(tablewriter.ALIGN_LEFT)
	memTableWriter.SetBorder(true)
	memTableWriter.SetHeader([]string{"Addr", "Data", "Section"})
}

func (mw *MemoryViewWindow) Draw() {
	mw.Window.Clear(colornames.Black)
	memTableWriter.ClearRows()

	consoleTxt.Color = colornames.White

	var data [][]string
	// print rows from memory
	for i := 0; i < (max_rows - addr_offset); i++ {
		row_str := ""
		row_addr_start := (i * 0x10) + (int(mw.YOffset) * 0x10)
		row_addr := fmt.Sprintf("0x%04x", row_addr_start)
		for j := 0; j < 16; j++ {
			addr := uint16(j + row_addr_start)
			row_str += fmt.Sprintf("%02x ", mw.hw.Mb.GetItem(addr))
		}
		data = append(data, []string{row_addr, row_str, motherboard.MemoryMapName(uint16(row_addr_start))})
	}

	for _, d := range data {
		memTableWriter.Append(d)
	}
	memTableWriter.Render()
	consoleTxt.Draw(mw.Window, pixel.IM.Scaled(consoleTxt.Orig, 1.25))
	consoleTxt.Clear()
	mw.Window.Update()
}

func (mw *MemoryViewWindow) Update() error {
	if mw.Window.JustPressed(pixelgl.KeyRight) || mw.Window.Repeated(pixelgl.KeyRight) {
		mw.YOffset += float64(max_rows) - float64(addr_offset) - 1
	}

	if mw.Window.JustPressed(pixelgl.KeyUp) || mw.Window.Repeated(pixelgl.KeyUp) {
		mw.YOffset -= 1
	}

	if mw.Window.JustPressed(pixelgl.KeyLeft) || mw.Window.Repeated(pixelgl.KeyLeft) {
		mw.YOffset -= float64(max_rows) - float64(addr_offset) - 1
	}

	if mw.Window.JustPressed(pixelgl.KeyDown) || mw.Window.Repeated(pixelgl.KeyDown) {
		mw.YOffset += 1
	}

	dy := mw.Window.MouseScroll().Y
	mw.YOffset -= dy
	if mw.YOffset < 0 {
		mw.YOffset = 0.0
	}

	maxYOffset := float64(0xffff-(max_rows-addr_offset-1)*0x10) / float64(0x10)

	if mw.YOffset > maxYOffset {
		mw.YOffset = maxYOffset
	}
	begin_addr = int(mw.YOffset) * 0x10
	end_addr = (max_rows-addr_offset-1)*0x10 + begin_addr

	if end_addr > 0xffff {
		end_addr = 0xffff
	}

	return nil
}
