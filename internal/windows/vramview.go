package windows

import (
	"fmt"
	"image/color"
	"math"

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
	picture *pixel.PictureData
}

func NewVramViewWindow(gobc *GoBoyColor) *VramViewWindow {
	/// create memory window
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "gobc v0.1 | VRAM View",
		Bounds: pixel.R(0, 0, vramTrueWidth, vramTrueHeight),
		VSync:  true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}
	return &VramViewWindow{
		Window:  win,
		YOffset: 0,
		hw:      gobc,
		picture: &pixel.PictureData{
			Pix:    make([]color.RGBA, int(vramTrueWidth)*int(vramTrueHeight)),
			Stride: int(vramTrueWidth),
			Rect:   pixel.R(0, 0, vramTrueWidth, vramTrueHeight),
		},
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
	tileData = mw.hw.Mb.Memory.Vram[0][0x400:0x520]
	for i := 0; i < len(tileData); i += 16 {
		tile := motherboard.Tile(tileData[i : i+16])
		palletteTile := tile.ParseTile()
		vramParsedTiles[i/16] = palletteTile

		for j := 0; j < 8; j++ {
			for k := 0; k < 8; k++ {
				// colIndex := palletteTile[j][k]
				// col := motherboard.Palettes[0][colIndex]
				// // rgb := color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}
				// height := int(vramTrueHeight) - 1 - k
				// width := int(vramTrueWidth) + j
				// fmt.Println(height*width, height, width)
				fmt.Println(len(mw.picture.Pix), mw.picture.Rect.Center())

				mw.picture.Pix[300*250] = color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0x00}
			}
		}
		break

	}

	return nil
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.Black)
	vramConsoleTxt.Color = colornames.Black
	fmt.Fprintf(vramConsoleTxt, fmt.Sprintf("%+v\n", vramParsedTiles))
	spr := pixel.NewSprite(pixel.Picture(mw.picture), pixel.R(0, 0, vramTrueWidth, vramTrueHeight))
	spr.Draw(mw.Window, pixel.IM.Scaled(mw.picture.Rect.Center(), 1))
	// vramConsoleTxt.Draw(mw.Window, pixel.IM)

	updateCamera(mw.Window)
	vramConsoleTxt.Clear()
	mw.Window.Update()
}

func updateCamera(win *pixelgl.Window) {
	xScale := win.Bounds().W() / vramTrueWidth
	yScale := win.Bounds().H() / vramTrueHeight
	scale := math.Min(yScale, xScale)

	shift := win.Bounds().Size().Scaled(0.5).Sub(pixel.ZV)
	cam := pixel.IM.Scaled(pixel.ZV, scale).Moved(shift)
	win.SetMatrix(cam)
}
