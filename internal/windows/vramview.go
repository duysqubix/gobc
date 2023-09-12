package windows

import (
	"github.com/duysqubix/gobc/internal"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"

	"golang.org/x/image/colornames"
)

const (
	vramScreenWidth  = 800
	vramScreenHeight = 800
	vramScale        = 1
	vramTrueWidth    = float64(vramScreenWidth * vramScale)
	vramTrueHeight   = float64(vramScreenHeight * vramScale)

	vramTileWidth         = 8
	vramTileHeight        = 8
	vramTilePictureWidth  = 32 * vramTileWidth
	vramTilePictureHeight = 12 * vramTileHeight
	vramTileScale         = 2

	vramTileMapPictureWidth  = 32 * vramTileWidth
	vramTileMapPictureHeight = 32 * vramTileHeight
	vramTileMapScale         = 2
)

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
		Title:       "gobc v0.1 | VRAM View",
		Bounds:      pixel.R(0, 0, vramTrueWidth, vramTrueHeight),
		VSync:       false,
		AlwaysOnTop: true,
		Resizable:   true,
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
	tileMap := mw.hw.Mb.Memory.TileMap()

	updatePicture(vramTilePictureHeight, vramTilePictureWidth, vramTileHeight, vramTileWidth, &tileData, mw.tileCanvas)
	updatePicture(vramTileMapPictureHeight, vramTileMapPictureWidth, vramTileHeight, vramTileWidth, &tileMap, mw.tileMapCanvas)

	return nil
}

func drawSprite(win *pixelgl.Window, canvas *pixel.PictureData, scale float64, YOffset float64, XOffset float64) {
	spr2 := pixel.NewSprite(canvas, canvas.Bounds())

	spw := (spr2.Frame().W() / 2 * scale) + (XOffset * scale)
	sph := (spr2.Frame().H() / 2 * scale) + (YOffset * scale)

	spr2.Draw(win, pixel.IM.Scaled(pixel.ZV, scale).Moved(pixel.V(spw, sph)))
}

func (mw *VramViewWindow) drawBorder() *imdraw.IMDraw {
	imd := imdraw.New(nil)
	imd.Color = pixel.RGB(1, 0, 0)

	x1 := 0.0
	y1 := (mw.tileMapCanvas.Rect.H() * vramTileMapScale) - (vramTileMapScale * internal.GB_SCREEN_HEIGHT)

	x2 := float64(vramTileMapScale) * float64(internal.GB_SCREEN_WIDTH)
	y2 := (mw.tileMapCanvas.Rect.H() * vramTileMapScale)

	imd.Push(pixel.V(x1, y1), pixel.V(x2, y2))
	imd.Rectangle(1)
	return imd
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.Black)


	drawSprite(mw.Window, mw.tileMapCanvas, vramTileMapScale, 0, 0)
	drawSprite(mw.Window, mw.tileCanvas, vramTileScale, (mw.tileCanvas.Rect.H()*vramTileMapScale)+100, 0)
	mw.drawBorder().Draw(mw.Window)
	mw.Window.Update()

}
