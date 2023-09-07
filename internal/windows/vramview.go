package windows

import (
	"image/color"

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

	vramTileWidth         = 8
	vramTileHeight        = 8
	vramTilePictureWidth  = 24 * vramTileWidth
	vramTilePictureHeight = 16 * vramTileHeight
	vramTileScale         = 2
)

var (
	vramDefaultFont *basicfont.Face
	vramConsoleTxt  *text.Text
	sprites         [16]*pixel.Sprite
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

	picture := pixel.MakePictureData(pixel.R(0, 0, 24*8, 16*8))

	return &VramViewWindow{
		Window:  win,
		YOffset: 0,
		hw:      gobc,
		picture: picture,
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
	)

}

func (mw *VramViewWindow) Update() error {
	// parse in VRAM data
	tileData := mw.hw.Mb.Memory.TileData()
	tileData = tileData[0x0260 : 0x0260+(16*16)]

	for i := 0; i < len(tileData); i += 16 {
		tile := motherboard.Tile(tileData[i : i+16])
		palletteTile := tile.ParseTile()
		pd := pixel.MakePictureData(pixel.R(0, 0, 8, 8))
		for j := 0; j < len(palletteTile); j++ {
			colIndex := palletteTile[j]
			col := motherboard.Palettes[0][colIndex]

			pd.Pix[len(palletteTile)-j-1] = color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}

		}
		sprites[i/16] = pixel.NewSprite(pixel.Picture(pd), pd.Rect)
	}

	return nil
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.White)
	vramConsoleTxt.Color = colornames.Black

	// draw tiles
	for i := 0; i < len(sprites); i++ {
		// spew.Dump(t)
		startPos := mw.Window.Bounds().Center().Sub(
			pixel.V(25.0, -18.0),
		)

		sprites[i].Draw(mw.Window, pixel.IM.
			Moved(startPos.Add(pixel.V(float64(i)*(9.0), 0.0))).
			Scaled(mw.Window.Bounds().Center(), vramTileScale),
		)
	}

	// sprites[0].Draw(mw.Window, pixel.IM.
	// 	Moved(mw.Window.Bounds().Center().Add(pixel.V(0.0, 0.0))).
	// 	Scaled(mw.Window.Bounds().Center(), 10),
	// )

	// sprites[1].Draw(mw.Window, pixel.IM.
	// 	Moved(mw.Window.Bounds().Center().Add(pixel.V(9.0, 0.0))).
	// 	Scaled(mw.Window.Bounds().Center(), 10),
	// )
	mw.Window.Update()

	vramConsoleTxt.Clear()
}
