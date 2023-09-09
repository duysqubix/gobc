package windows

import (
	"image/color"

	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

const (
	vramScreenWidth  = 800
	vramScreenHeight = 500
	vramScale        = 1
	vramTrueWidth    = float64(vramScreenWidth * vramScale)
	vramTrueHeight   = float64(vramScreenHeight * vramScale)

	vramTileWidth         = 8
	vramTileHeight        = 8
	vramTilePictureWidth  = 32 * vramTileWidth
	vramTilePictureHeight = 12 * vramTileHeight
	vramTileScale         = 1
)

var (
	vramSprites   [1]*pixel.Sprite
	vramTileBatch *pixel.Batch
)

func init() {

}

type VramViewWindow struct {
	hw         *GoBoyColor
	YOffset    float64
	Window     *pixelgl.Window
	tileCanvas *pixel.PictureData
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

	tileCanvas := pixel.MakePictureData(pixel.R(0, 0, float64(vramTilePictureWidth), float64(vramTilePictureHeight)))
	return &VramViewWindow{
		Window:     win,
		YOffset:    0,
		hw:         gobc,
		tileCanvas: tileCanvas,
	}
}

func (mw *VramViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *VramViewWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, vramTrueWidth, vramTrueHeight))
}

func (mw *VramViewWindow) Update() error {
	// vramTileBatch.Clear()
	// parse in VRAM data
	tileData := mw.hw.Mb.Memory.TileData()
	// tileData = tileData[0x0250 : 0x0250+(16*4)]
	tileNum := 0
	// for i := 0; i < len(tileData); i += 16 {
	// for yCursor := 0; yCursor < vramTilePictureHeight; yCursor += vramTileHeight {
	for yCursor := vramTilePictureHeight - vramTileHeight; yCursor >= 0; yCursor -= vramTileHeight {
		for xCursor := 0; xCursor < vramTilePictureWidth; xCursor += vramTileWidth {

			tile := motherboard.Tile(tileData[tileNum : tileNum+16])
			palletteTile := tile.ParseTile()

			for yPixel := 0; yPixel < vramTileHeight; yPixel++ {
				for xPixel := 0; xPixel < vramTileWidth; xPixel++ {
					colIndex := palletteTile[yPixel*vramTileWidth+xPixel]
					col := motherboard.Palettes[0][colIndex]
					rgb := color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}
					idx := (yCursor+yPixel)*vramTilePictureWidth + (xCursor + xPixel)
					mw.tileCanvas.Pix[idx] = rgb
				}
			}
			tileNum += 16
		}
	}
	// }

	return nil
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.White)
	// startPos := mw.Window.Bounds().Center().Sub(
	// 	pixel.V(250.0, -180.0),
	// )
	// for i := 0; i < len(vramSprites); i++ {

	// vramSprites[i].Draw(mw.Window, pixel.IM.
	// 	Moved(startPos.Add(pixel.V(float64(i)*(9.0), 0.0))).
	// 	Scaled(mw.Window.Bounds().Center(), vramTileScale),
	// )

	// vramSprites[i].Draw(vramTileBatch, pixel.IM.Scaled(pixel.ZV, 20).Moved(pixel.V(200.0, 200.0)))
	// Moved(startPos.Add(pixel.V(float64(i)*(1.0), 0.0))).
	// fmt.Printf("%d: VRAM Sprite Coords: %v\n", i, vramSprites[i].Frame())
	// }
	// vramTileBatch.Draw(mw.Window)

	spr := pixel.NewSprite(mw.tileCanvas, mw.tileCanvas.Bounds())
	spr.Draw(mw.Window, pixel.IM.Scaled(pixel.ZV, 3).Moved(mw.Window.Bounds().Center()))
	mw.Window.Update()

}
