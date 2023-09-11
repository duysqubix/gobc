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

	vramTileMapPictureWidth  = 32 * vramTileWidth
	vramTileMapPictureHeight = 32 * vramTileHeight
	vramTileMapScale         = 1
)

var ()

func init() {

}

type VramViewWindow struct {
	hw            *GoBoyColor
	YOffset       float64
	Window        *pixelgl.Window
	tileCanvas    *pixel.PictureData
	tileMapCanvas *pixel.PictureData
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
	tileMapCanvas := pixel.MakePictureData(pixel.R(0, 0, float64(vramTileMapPictureWidth), float64(vramTileMapPictureHeight)))
	return &VramViewWindow{
		Window:        win,
		YOffset:       0,
		hw:            gobc,
		tileCanvas:    tileCanvas,
		tileMapCanvas: tileMapCanvas,
	}
}

func (mw *VramViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *VramViewWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, vramTrueWidth, vramTrueHeight))
}

func (mw *VramViewWindow) Update() error {

	tileData := mw.hw.Mb.Memory.TileData()
	tileNum := 0

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

	tileMap := mw.hw.Mb.Memory.TileMap()
	tileNum = 0
	for yCursor := vramTileMapPictureHeight - vramTileHeight; yCursor >= 0; yCursor -= vramTileHeight {
		for xCursor := 0; xCursor < vramTileMapPictureWidth; xCursor += vramTileWidth {
			tile := motherboard.Tile(tileMap[tileNum : tileNum+16])
			palletteTile := tile.ParseTile()

			for yPixel := 0; yPixel < vramTileHeight; yPixel++ {
				for xPixel := 0; xPixel < vramTileWidth; xPixel++ {
					colIndex := palletteTile[yPixel*vramTileWidth+xPixel]
					col := motherboard.Palettes[0][colIndex]
					rgb := color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}
					idx := (yCursor+yPixel)*vramTileMapPictureWidth + (xCursor + xPixel)
					mw.tileMapCanvas.Pix[idx] = rgb
				}
			}
		}
	}

	return nil
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.White)

	// spr := pixel.NewSprite(mw.tileCanvas, mw.tileCanvas.Bounds())
	// spr.Draw(mw.Window, pixel.IM.Scaled(pixel.ZV, 3).Moved(mw.Window.Bounds().Center()))

	spr2 := pixel.NewSprite(mw.tileMapCanvas, mw.tileMapCanvas.Bounds())
	spr2.Draw(mw.Window, pixel.IM.Scaled(pixel.ZV, 3).Moved(pixel.V(0, 0)))
	mw.Window.Update()

}
