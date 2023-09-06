package windows

import (
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
)

var (
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

	// var pic [vramTilePictureHeight][vramTilePictureWidth]uint8 = [vramTilePictureHeight][vramTilePictureWidth]uint8{}
	//iterate over tiles
	for i := 0; i < len(tileData); i += 16 {
	}

	for y := 0; y < vramTilePictureHeight; y++ {
		for x := 0; x < vramTilePictureWidth; x++ {

		}
	}

	// colCntr := 0
	// for i := 0; i < len(tileData); i += 16 {
	// 	tile := motherboard.Tile(tileData[i : i+16])
	// 	// fmt.Printf("Len: %v\n", tile)
	// 	// os.Exit(0)
	// 	palletteTile := tile.ParseTile()

	// 	for j := 0; j < len(palletteTile); j++ {
	// 		colIndex := palletteTile[j]
	// 		col := motherboard.Palettes[0][colIndex]

	// 		// idx := (i>>1)*8 + j
	// 		idx := (colCntr>>4)*mw.picture.Stride + j
	// 		// fmt.Printf("%d, ", idx)
	// 		mw.picture.Pix[idx] = color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}

	// 	}
	// 	colCntr++
	// 	break
	// }

	return nil
}

func (mw *VramViewWindow) Draw() {
	mw.Window.Clear(colornames.White)
	vramConsoleTxt.Color = colornames.Black

	spr := pixel.NewSprite(pixel.Picture(mw.picture), mw.picture.Rect)
	spr.Draw(mw.Window, pixel.IM.Moved(mw.Window.Bounds().Center()).Scaled(mw.Window.Bounds().Center(), 3))

	vramConsoleTxt.Clear()
	mw.Window.Update()
}
