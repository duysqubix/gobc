package windows

import (
	"fmt"

	"github.com/duysqubix/gobc/internal"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"

	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
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

var (
	vramTileAddressingMode uint8 = 0x01 // default 0x8000 mode
	vramBgAddressingMode   uint8 = 0x01 // default 0x9800 mode
	vramConsoleTxt         *text.Text
	vramShowHelp           bool = false
	vramShowGrid           bool = true
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
	vramConsoleTxt = text.New(
		mw.Window.Bounds().Center(),
		text.NewAtlas(basicfont.Face7x13, text.ASCII),
	)
	vramConsoleTxt.Color = colornames.Red
}

func (mw *VramViewWindow) Update() error {

	if mw.Window.JustPressed(pixelgl.KeyT) || mw.Window.Repeated(pixelgl.KeyT) {
		internal.ToggleBit(&vramTileAddressingMode, 0)
	}

	if mw.Window.JustPressed(pixelgl.KeyB) || mw.Window.Repeated(pixelgl.KeyB) {
		internal.ToggleBit(&vramBgAddressingMode, 0)
	}

	if mw.Window.JustPressed(pixelgl.KeyG) || mw.Window.Repeated(pixelgl.KeyG) {
		vramShowGrid = !vramShowGrid
	}

	if mw.Window.JustPressed(pixelgl.KeyH) || mw.Window.Repeated(pixelgl.KeyH) {
		vramShowHelp = !vramShowHelp
	}

	tileData := mw.hw.Mb.Memory.TileData()
	tileMap := mw.hw.Mb.Memory.TileMap(vramTileAddressingMode, vramBgAddressingMode)

	updatePicture(vramTilePictureHeight, vramTilePictureWidth, vramTileHeight, vramTileWidth, &tileData, mw.tileCanvas)
	updatePicture(vramTileMapPictureHeight, vramTileMapPictureWidth, vramTileHeight, vramTileWidth, &tileMap, mw.tileMapCanvas)

	return nil
}

func drawVramArea(win *pixelgl.Window, canvas *pixel.PictureData, scale float64, YOffset float64, XOffset float64) {
	spr2 := pixel.NewSprite(canvas, canvas.Bounds())

	spw := (spr2.Frame().W() / 2 * scale) + (XOffset * scale)
	sph := (spr2.Frame().H() / 2 * scale) + (YOffset * scale)

	spr2.Draw(win, pixel.IM.Scaled(pixel.ZV, scale).Moved(pixel.V(spw, sph)))

	if vramShowGrid {
		// create grid for selected canvas by tile height and width
		imd := imdraw.New(nil)
		imd.Color = pixel.RGB(1, 0, 0)
		// Draw vertical lines
		for x := 0.0; x <= spr2.Frame().W()*scale; x += 8.0 * scale {
			imd.Push(pixel.V(x+spw-spr2.Frame().W()/2*scale, sph-spr2.Frame().H()/2*scale))
			imd.Push(pixel.V(x+spw-spr2.Frame().W()/2*scale, sph+spr2.Frame().H()/2*scale))
			imd.Line(1)
		}

		// Draw horizontal lines
		for y := 0.0; y <= spr2.Frame().H()*scale; y += 8.0 * scale {
			imd.Push(pixel.V(spw-spr2.Frame().W()/2*scale, y+sph-spr2.Frame().H()/2*scale))
			imd.Push(pixel.V(spw+spr2.Frame().W()/2*scale, y+sph-spr2.Frame().H()/2*scale))
			imd.Line(1)
		}

		imd.Draw(win)
	}
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.Black)

	drawVramArea(mw.Window, mw.tileMapCanvas, vramTileMapScale, 0, 0)
	drawVramArea(mw.Window, mw.tileCanvas, vramTileScale, (mw.tileCanvas.Rect.H()*vramTileMapScale)+100, 0)

	if vramShowHelp {
		// TODO: not displaying...
		vramConsoleTxt.Clear()
		fmt.Fprintf(vramConsoleTxt, "VRAM View Help\n")
		fmt.Fprintf(vramConsoleTxt, "T: Toggle Tile Addressing Mode\n")
		fmt.Fprintf(vramConsoleTxt, "G: Toggle Grid\n")
		fmt.Fprintf(vramConsoleTxt, "H: Show this menu\n")
		vramConsoleTxt.Draw(mw.Window, pixel.IM.Moved(vramConsoleTxt.Orig).Scaled(vramConsoleTxt.Orig, 1.25))

	}
	mw.Window.Update()

}
