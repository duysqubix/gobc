package windows

import (
	"image/color"

	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
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
	vramTileScale         = 1
)

var (
	vramSprites [16]*pixel.Sprite
)

func init() {

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
		VSync:  false,
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
}

func (mw *VramViewWindow) Update() error {
	// parse in VRAM data
	tileData := mw.hw.Mb.Memory.TileData()
	tileData = tileData[0x0210 : 0x0210+(16*16)]

	for i := 0; i < len(tileData); i += 16 {
		tile := motherboard.Tile(tileData[i : i+16])
		palletteTile := tile.ParseTile()
		pd := pixel.MakePictureData(pixel.R(0, 0, 8, 8))
		for j := 0; j < len(palletteTile); j++ {
			colIndex := palletteTile[j]
			col := motherboard.Palettes[0][colIndex]

			pd.Pix[len(palletteTile)-j-1] = color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}

		}
		pixelgl.NewGLPicture(pixel.MakePictureData(pixel.R(0, 0, 8, 8)))

		vramSprites[i/16] = pixel.NewSprite(pd, pixel.Picture(pd).Bounds())
		// sprites = append(sprites, pixel.NewSprite(pixel.Picture(pd), pd.Rect))

	}

	return nil
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.Black)

	startPos := mw.Window.Bounds().Center().Sub(
		pixel.V(250.0, -180.0),
	)
	for i := 0; i < len(vramSprites); i++ {

		vramSprites[i].Draw(mw.Window, pixel.IM.
			Moved(startPos.Add(pixel.V(float64(i)*(9.0), 0.0))).
			Scaled(mw.Window.Bounds().Center(), vramTileScale),
		)
		// sprites[i].Draw(tileBatch, pixel.IM.
		// 	Moved(startPos.Add(pixel.V(float64(i)*(1.0), 0.0))).
		// 	Scaled(mw.Window.Bounds().Center(), vramTileScale))
	}
	// tileBatch.Draw(mw.Window)
	mw.Window.Update()

}
