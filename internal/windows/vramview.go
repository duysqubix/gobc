package windows

import (
	"fmt"

	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

const (
	vramScreenWidth  = 600
	vramScreenHeight = 500
	vramScale        = 1
	vramTrueWidth    = float64(vramScreenWidth * vramScale)
	vramTrueHeight   = float64(vramScreenHeight * vramScale)
	vramfontBuffer   = 4
)

var (
	vramParsedTiles [384]motherboard.PaletteTile // 384 tiles in VRAM
	vramDefaultFont *basicfont.Face
	vramConsoleTxt  *text.Text
)

func init() {
	vramDefaultFont = basicfont.Face7x13

}

type VramViewWindow struct {
	hw      *GoBoyColor
	YOffset float64
	Window  *pixelgl.Window
}

func NewVramViewWindow(gobc *GoBoyColor) *VramViewWindow {
	/// create memory window
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "gobc v0.1 | VRAM View",
		Bounds: pixel.R(0, 0, 670, 1000),
		VSync:  true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}
	return &VramViewWindow{
		Window:  win,
		YOffset: 0,
		hw:      gobc,
	}
}

func (mw *VramViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *VramViewWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, vramTrueWidth, vramTrueHeight))

	vramConsoleTxt = text.New(
		pixel.V(10, mw.Window.Bounds().Max.Y-20),
		text.NewAtlas(defaultFont, text.ASCII),
		// text.NewAtlas(inconsolata.Regular8x16, text.ASCII),
	)

}

func (mw *VramViewWindow) Update() error {
	// parse in VRAM data
	tileData := mw.hw.Mb.Memory.TileData()

	tileData = tileData[0x0400:0x040F]
	fmt.Println(tileData)

	for i := 0; i < len(tileData); i += 16 {
		tile := motherboard.Tile(tileData[i : i+16])
		vramParsedTiles[i/16] = tile.ParseTile()
	}

	return nil
}

func (mw *VramViewWindow) Draw() {
	// draw VRAM data
	mw.Window.Clear(colornames.White)
	vramConsoleTxt.Color = colornames.Black

	// spew.Fdump(vramConsoleTxt, vramParsedTiles[64])
	fmt.Fprintf(vramConsoleTxt, fmt.Sprintf("%+v\n", vramParsedTiles[0]))

	vramConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(vramConsoleTxt.Orig, 1))
	vramConsoleTxt.Clear()
	mw.Window.Update()
}
