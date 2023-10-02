package windows

import (
	"fmt"

	"github.com/duysqubix/gobc/internal"
	pixel "github.com/duysqubix/pixel2"
	"github.com/duysqubix/pixel2/imdraw"
	"github.com/duysqubix/pixel2/pixelgl"
	"github.com/duysqubix/pixel2/text"

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

	vramTilePreviewPictureWidth  = vramTileWidth
	vramTilePreviewPictureHeight = vramTileHeight
	vramTilePreviewScale         = 20

	gridSize    = 16
	gridSizeMax = 32
)

var (
	vramTileAddressingMode uint8 = 0x01 // default 0x8000 mode
	vramBgAddressingMode   uint8 = 0x01 // default 0x9800 mode
	vramConsoleTxt         *text.Text
	vramShowHelp           bool  = false
	vramShowGrid           bool  = true
	vramTileDataBank       uint8 = 0x00
)

func init() {

}

type VramViewWindow struct {
	hw                *GoBoyColor
	YOffset           float64
	Window            *pixelgl.Window
	tileCanvas        *pixel.PictureData
	tileMapCanvas     *pixel.PictureData
	tilePreviewCanvas *pixel.PictureData
}

func NewVramViewWindow(gobc *GoBoyColor) *VramViewWindow {
	/// create memory window
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:       fmt.Sprintf("gobc v%s | VRAM View", internal.VERSION),
		Bounds:      pixel.R(0, 0, vramTrueWidth, vramTrueHeight),
		VSync:       true,
		AlwaysOnTop: true,
		Resizable:   true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}

	tileCanvas := pixel.MakePictureData(pixel.R(0, 0, float64(vramTilePictureWidth), float64(vramTilePictureHeight)))
	tileMapCanvas := pixel.MakePictureData(pixel.R(0, 0, float64(vramTileMapPictureWidth), float64(vramTileMapPictureHeight)))
	tilePreviewCanvas := pixel.MakePictureData(pixel.R(0, 0, float64(vramTilePreviewPictureWidth), float64(vramTilePreviewPictureHeight)))

	return &VramViewWindow{
		Window:            win,
		YOffset:           0,
		hw:                gobc,
		tileCanvas:        tileCanvas,
		tileMapCanvas:     tileMapCanvas,
		tilePreviewCanvas: tilePreviewCanvas,
	}
}

func (mw *VramViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *VramViewWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, vramTrueWidth, vramTrueHeight))
	vramConsoleTxt = text.New(
		pixel.V(520, 520),
		text.NewAtlas(basicfont.Face7x13, text.ASCII),
	)
	vramConsoleTxt.Color = colornames.Green
}

func (mw *VramViewWindow) Finalize() {
	mw.Window.Update()
}

func calculateVramIndex(x, y int) int {
	// cover bounds of tile map
	if x < 0 || x > vramTileMapPictureWidth*vramTileMapScale || y < 0 || y > vramTileMapPictureHeight*vramTileMapScale {
		return -1
	}
	row := y / gridSize
	col := x / gridSize

	index := (gridSizeMax-row-1)*gridSizeMax + col
	return index
}

var currentIndex int = 0
var currentTileOffset int16 = 0
var currentTileLocation uint16 = 0
var bgAddressingMode uint16
var tileAddressingMode uint16

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

	if mw.Window.JustPressed(pixelgl.KeyV) || mw.Window.Repeated(pixelgl.KeyV) {
		vramTileDataBank = (vramTileDataBank + 1) % 2
	}

	var unsigned bool = false
	if vramBgAddressingMode == 0x01 {
		bgAddressingMode = 0x9800
	} else {
		bgAddressingMode = 0x9C00
	}

	if vramTileAddressingMode == 0x01 {
		tileAddressingMode = 0x8000
		unsigned = true
	} else {
		tileAddressingMode = 0x8800
		unsigned = false
	}

	mw.Window.SetTitle(fmt.Sprintf("gobc v0.1 | VRAM View | BG: %#x | Tile: %#x | Unsigned: %v", bgAddressingMode, tileAddressingMode, unsigned))

	v := mw.Window.MousePosition()
	currentIndex = calculateVramIndex(int(v.X), int(v.Y))
	if currentIndex > 0 {
		currentTileLocation, currentTileOffset = mw.hw.Mb.Lcd.FindTileLocation(uint16(currentIndex)+bgAddressingMode, tileAddressingMode, unsigned)
	}

	tileData := mw.hw.Mb.Memory.TileData(vramTileDataBank)
	tileMap := mw.hw.Mb.Memory.TileMap(vramTileAddressingMode, vramBgAddressingMode)

	updatePicture(vramTilePictureHeight, vramTilePictureWidth, vramTileHeight, vramTileWidth, &tileData, mw.tileCanvas)
	updatePicture(vramTileMapPictureHeight, vramTileMapPictureWidth, vramTileHeight, vramTileWidth, &tileMap, mw.tileMapCanvas)

	// update tile preview with select Tile
	if currentTileLocation > 0 {
		tilePreview := tileData[currentTileLocation-0x8000 : currentTileLocation+16-0x8000]
		updatePicture(vramTilePreviewPictureHeight, vramTilePreviewPictureWidth, vramTileHeight, vramTileWidth, &tilePreview, mw.tilePreviewCanvas)
	}
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
	vramConsoleTxt.Clear()

	fmt.Fprintf(vramConsoleTxt, "Index: %d ($%02x)\n", currentIndex, currentIndex)
	fmt.Fprintf(vramConsoleTxt, "TileIndex: %d ($%04x)\n", currentTileOffset, currentTileOffset)
	fmt.Fprintf(vramConsoleTxt, "\t@ VRAM 00:%04X\n", int32(currentIndex)+int32(bgAddressingMode))
	vramConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(vramConsoleTxt.Orig, 1.5))

	drawVramArea(mw.Window, mw.tileMapCanvas, vramTileMapScale, 0, 0)
	drawVramArea(mw.Window, mw.tileCanvas, vramTileScale, (mw.tileCanvas.Rect.H()*vramTileMapScale)+100, 0)

	spr2 := pixel.NewSprite(mw.tilePreviewCanvas, mw.tilePreviewCanvas.Bounds())

	v := pixel.V(
		(mw.tileMapCanvas.Rect.W()*vramTileMapScale)+((mw.tileMapCanvas.Rect.W()*vramTileMapScale)/3)-80,
		mw.tileMapCanvas.Rect.H()*vramTileMapScale+150,
	)
	spr2.Draw(mw.Window, pixel.IM.Scaled(pixel.ZV, vramTilePreviewScale).Moved(v))
}
