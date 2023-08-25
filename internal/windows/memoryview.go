package windows

import (
	"fmt"

	"github.com/duysqubix/gobc/internal"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/inconsolata"
)

const (
	memScreenWidth  = 670
	memScreenHeight = 1000
	memScale        = 1
	memTrueWidth    = float64(memScreenWidth * memScale)
	memTrueHeight   = float64(memScreenHeight * memScale)
	fontBuffer      = 4
	addr_offset     = 10
)

var (
	defaultFont *basicfont.Face
	logger      = internal.Logger
	mock_memory []uint8
	max_chars   int
	max_rows    int
	row_ptr     int = 1 // starting at the top of the screen
	begin_addr  int
	end_addr    int

	consoleTxt *text.Text
)

func init() {
	mock_memory = make([]uint8, 0xffff)

	for i := 0; i < 0xffff; i++ {
		mock_memory[i] = uint8(i)
	}

	// set up font
	defaultFont = inconsolata.Regular8x16

	// max_chars = memTrueWidth / defaultFont.Width
	// max_rows = memTrueHeight / (defaultFont.Height + 2)

	begin_addr = 0
	end_addr = (max_rows-addr_offset-1)*0x10 + begin_addr
}

type MemoryViewWindow struct {
	mw      *GoBoyColor
	YOffset float64
	Window  *pixelgl.Window
}

func NewMemoryViewWindow(gobc *GoBoyColor) *MemoryViewWindow {
	/// create memory window
	memWin, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "gobc v0.1 | Memory View",
		Bounds: pixel.R(0, 0, 670, 1000),
		VSync:  true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}
	return &MemoryViewWindow{
		Window:  memWin,
		YOffset: 0,
		mw:      gobc,
	}
}

func (mw *MemoryViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *MemoryViewWindow) SetUp() {
	mw.Window.SetTitle("gobc v0.1 | Memory View")
	mw.Window.SetBounds(pixel.R(0, 0, memTrueWidth, memTrueHeight))
	consoleTxt = text.New(
		pixel.V(0, mw.Window.Bounds().Max.Y/2),
		text.NewAtlas(basicfont.Face7x13, text.ASCII),
	)

}

func (mw *MemoryViewWindow) Draw() {
	mw.Window.Clear(colornames.Black)
	fmt.Fprintf(consoleTxt, "Hello world")

	consoleTxt.Draw(mw.Window, pixel.IM.Scaled(consoleTxt.Orig, 4))
	// PrintAt(screen, strings.Repeat("-", max_chars))
	// PrintAt(screen, fmt.Sprintf("Memory Address: 0x%04x - 0x%04x", begin_addr, end_addr))
	// PrintAt(screen, strings.Repeat("-", max_chars))

	// // print rows from memory
	// for i := 0; i < max_rows-addr_offset; i++ {

	// 	row_addr_start := (i * 0x10) + (int(g.YOffset) * 0x10)
	// 	row_str := fmt.Sprintf("0x%04x |", row_addr_start)
	// 	for j := 0; j < 16; j++ {
	// 		row_str += fmt.Sprintf(" %02x ", 0xDE) //g.memory[j+row_addr_start])
	// 	}
	// 	PrintAt(screen, row_str)
	// }
	// PrintAt(screen, strings.Repeat("-", max_chars))
	// row_ptr = 1
	consoleTxt.Clear()
	mw.Window.Update()
}

func (mw *MemoryViewWindow) Update() error {
	// if repeatingKeyPressed(ebiten.KeyPageDown) {
	// 	g.YOffset += float64(max_rows) - float64(addr_offset) - 1
	// }

	// if repeatingKeyPressed(ebiten.KeyPageUp) {
	// 	g.YOffset -= float64(max_rows) - float64(addr_offset) - 1
	// }

	// _, dy := ebiten.Wheel()
	// g.YOffset -= dy
	// if g.YOffset < 0 {
	// 	g.YOffset = 0.0
	// }

	// begin_addr = int(g.YOffset) * 0x10
	// end_addr = (max_rows-addr_offset-1)*0x10 + begin_addr
	return nil
}

// func repeatingKeyPressed(key ebiten.Key) bool {
// 	const (
// 		delay    = 30
// 		interval = 3
// 	)
// 	d := inpututil.KeyPressDuration(key)
// 	if d == 1 {
// 		return true
// 	}
// 	if d >= delay && (d-delay)%interval == 0 {
// 		return true
// 	}
// 	return false
// }

// func PrintAt(image *ebiten.Image, str string) {

// 	if row_ptr < 1 {
// 		logger.Errorf("row_ptr is negative: %d\n", row_ptr)
// 		for {

// 		}
// 	}
// 	// check if str len is longer than max cols
// 	row_str := "| " + str
// 	if len(row_str) > max_chars {
// 		row_str = row_str[:max_chars]
// 		row_str += "|"
// 	} else {
// 		buf_len := max_chars - len(row_str)
// 		row_str += strings.Repeat(" ", buf_len)
// 		row_str += "|"
// 	}

// 	y := row_ptr * 20
// 	text.Draw(image, row_str, defaultFont, 20, y, color.White)
// 	row_ptr++
// }

// func (g *MemoryViewWindow) Layout(outsideWidth, outsideHeight int) (int, int) {
// 	return trueWidth, trueHeight
// }
