package windows

import (
	"fmt"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
	"github.com/duysqubix/pixel2"
	"github.com/duysqubix/pixel2/pixelgl"
	"github.com/duysqubix/pixel2/text"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

const (
	cartScreenWidth  = 670
	cartScreenHeight = 1000
	cartScale        = 1
	cartTrueWidth    = float64(cartScreenWidth * cartScale)
	cartTrueHeight   = float64(cartScreenHeight * cartScale)
	cartFontBuffer   = 4
)

var (
	cartDefaultFont *basicfont.Face
	cartMaxRows     int
	cartBeginAddr   int
	cartEndAddr     int
	cartAddrOffset  int = 10
	cartConsoleTxt  *text.Text
	cartTableWriter *tablewriter.Table
	cartRambank     int = 0
)

func init() {

	// set up font
	cartDefaultFont = basicfont.Face7x13

	// cartMaxRows = int(cartTrueHeight) / (cartDefaultFont.Height + 2)
	cartMaxRows = 32

	cartBeginAddr = 0
	cartEndAddr = (cartMaxRows-cartAddrOffset-1)*0x10 + cartBeginAddr
}

type CartViewWindow struct {
	hw      *GoBoyColor
	YOffset float64
	Window  *pixelgl.Window
}

func NewCartViewWindow(gobc *GoBoyColor) *CartViewWindow {
	/// create memory window
	memWin, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:       fmt.Sprintf("gobc v%s | Cart Memory View", internal.VERSION),
		Bounds:      pixel.R(0, 0, cartTrueWidth, cartScreenHeight),
		VSync:       true,
		AlwaysOnTop: true,
		Resizable:   true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}
	return &CartViewWindow{
		Window:  memWin,
		YOffset: 0,
		hw:      gobc,
	}
}

func (mw *CartViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *CartViewWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, cartTrueWidth, cartTrueHeight))
	cartConsoleTxt = text.New(
		pixel.V(10, 5),
		text.NewAtlas(cartDefaultFont, text.ASCII),
	)

	cartConsoleTxt.Color = colornames.Yellowgreen

	cartTableWriter = tablewriter.NewWriter(cartConsoleTxt)
	cartTableWriter.SetAutoWrapText(false)
	cartTableWriter.SetAlignment(tablewriter.ALIGN_LEFT)
	cartTableWriter.SetBorder(true)
	cartTableWriter.SetHeader([]string{"RAM Bank", "Addr", "Cart Data"})
}

func (mw *CartViewWindow) Update() error {
	maxYOffset := float64(int(cartridge.RAM_BANK_SIZE-1)-(cartMaxRows-cartAddrOffset-1)*0x10) / float64(0x10)

	if mw.Window.JustPressed(pixelgl.KeyRight) || mw.Window.Repeated(pixelgl.KeyRight) {
		mw.YOffset += float64(cartMaxRows) - float64(cartAddrOffset) - 1
	}

	if mw.Window.JustPressed(pixelgl.KeyUp) || mw.Window.Repeated(pixelgl.KeyUp) {
		mw.YOffset -= 1
	}

	if mw.Window.JustPressed(pixelgl.KeyLeft) || mw.Window.Repeated(pixelgl.KeyLeft) {
		mw.YOffset -= float64(cartMaxRows) - float64(cartAddrOffset) - 1
	}

	if mw.Window.JustPressed(pixelgl.KeyDown) || mw.Window.Repeated(pixelgl.KeyDown) {
		mw.YOffset += 1
	}

	if mw.Window.JustPressed(pixelgl.KeyRightBracket) || mw.Window.Repeated(pixelgl.KeyRightBracket) {
		cartRambank++
		if cartRambank >= 16 {
			cartRambank = 0
		}
	}

	if mw.Window.JustPressed(pixelgl.KeyLeftBracket) || mw.Window.Repeated(pixelgl.KeyLeftBracket) {
		cartRambank--
		if cartRambank < 0 {
			cartRambank = 15
		}

	}

	dy := mw.Window.MouseScroll().Y
	mw.YOffset -= dy
	if mw.YOffset < 0 {
		mw.YOffset = 0.0
	}

	if mw.YOffset > maxYOffset {
		mw.YOffset = maxYOffset
	}
	cartBeginAddr = int(mw.YOffset) * 0x10
	cartEndAddr = (cartMaxRows-cartAddrOffset-1)*0x10 + cartBeginAddr

	if cartEndAddr > int(cartridge.RAM_BANK_SIZE) {
		cartEndAddr = int(cartridge.RAM_BANK_SIZE)
	}

	return nil
}

func (mw *CartViewWindow) Draw() {
	cartConsoleTxt.Clear()
	mw.Window.Clear(colornames.Black)
	cartTableWriter.ClearRows()

	var data [][]string
	// print rows from memory
	for i := 0; i < (cartMaxRows - cartAddrOffset); i++ {
		row_str := ""
		row_addr_start := (i * 0x10) + (int(mw.YOffset) * 0x10)
		row_addr := fmt.Sprintf("0x%04x", row_addr_start)
		for j := 0; j < 16; j++ {
			addr := uint16(j + row_addr_start)
			row_str += fmt.Sprintf("%02x ", mw.hw.Mb.Cartridge.RamBanks[cartRambank][addr])
		}
		data = append(data, []string{fmt.Sprintf("Bank %d", cartRambank), row_addr, row_str})
	}

	for _, d := range data {
		// fmt.Println(d)
		cartTableWriter.Append(d)
	}
	cartTableWriter.Render()
	// fmt.Fprintf(cartConsoleTxt, "YOffset: %f\n", mw.YOffset)
	cartConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(cartConsoleTxt.Orig, 1.25))

	mw.Window.Update()
}
