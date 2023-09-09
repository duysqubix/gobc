package main

import (
	"fmt"
	"time"

	"image/color"
	_ "image/png"

	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

var H = 8 * 2
var W = 8 * 2

var tileData = [16 * 4]uint8{
	0x76, 0x76, 0x0c, 0x0c, 0x18, 0x18, 0x0c, 0x0c, 0x06, 0x06, 0x66, 0x66, 0x3c, 0x3c, 0, 0,
	0xc, 0xc, 0x1c, 0x1c, 0x3c, 0x3c, 0x6c, 0x6c, 0x7e, 0x7e, 0xc, 0xc, 0xc, 0xc, 0, 0,
	0x7e, 0x7e, 0x60, 0x60, 0x7c, 0x7c, 0x6, 0x6, 0x6, 0x6, 0x66, 0x66, 0x3c, 0x3c, 0, 0,
	0x3c, 0x3c, 0x60, 0x60, 0x60, 0x60, 0x7c, 0x7c, 0x66, 0x66, 0x66, 0x66, 0x3c, 0x3c, 0, 0,
}

func run() {
	cfg := pixelgl.WindowConfig{
		Title:  "Pixel Rocks!",
		Bounds: pixel.R(0, 0, 1024, 768),
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	var (
		frames = 0
		second = time.Tick(time.Second)
	)

	canvas := pixel.MakePictureData(pixel.R(0, 0, float64(W), float64(H)))

	for !win.Closed() {

		win.Clear(colornames.Red)

		tileNum := 0
		tileWidth := 8
		for yCursor := W - tileWidth; yCursor >= 0; yCursor -= tileWidth {
			for xCursor := 0; xCursor < H; xCursor += tileWidth {

				tile := motherboard.Tile(tileData[tileNum : tileNum+16])
				ptile := tile.ParseTile()

				for yPixel := 0; yPixel < tileWidth; yPixel++ {
					for xPixel := 0; xPixel < tileWidth; xPixel++ {

						xPos := xCursor*tileWidth + xPixel
						yPos := yCursor*tileWidth + yPixel

						colorPalettePixel := ptile[yPixel*tileWidth+xPixel]
						cols := motherboard.Palettes[0][colorPalettePixel]
						rgb := color.RGBA{R: cols[0], G: cols[1], B: cols[2], A: 0xFF}

						idx := (yCursor+yPixel)*W + (xCursor + xPixel)

						fmt.Printf("xCursor: %d, yCursor: %d, xPixel: %d, yPixel: %d, xPos: %d, yPos: %d, idx: %d\n", xCursor, yCursor, xPixel, yPixel, xPos, yPos, idx)
						canvas.Pix[idx] = rgb
					}
				}
				tileNum += 16
			}

		}
		spr := pixel.NewSprite(canvas, canvas.Bounds())
		spr.Draw(win, pixel.IM.Scaled(pixel.ZV, 5).Moved(win.Bounds().Center()))
		win.Update()

		frames++
		select {
		case <-second:
			win.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, frames))
			frames = 0
		default:
		}
	}
}

func main() {
	pixelgl.Run(run)
}
