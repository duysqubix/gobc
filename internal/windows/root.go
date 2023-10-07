package windows

import (
	"image/color"

	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/gopxl/pixel/v2"
)

func updatePicture(pictureHeight int, pictureWidth int, tileHeight int, tileWidth int, tileData *[]uint8, canvas *pixel.PictureData) {
	tileNum := 0
	for yCursor := pictureHeight - tileHeight; yCursor >= 0; yCursor -= tileHeight {
		for xCursor := 0; xCursor < pictureWidth; xCursor += tileWidth {

			tile := motherboard.Tile((*tileData)[tileNum : tileNum+16])
			palletteTile := tile.ParseTile()

			for yPixel := 0; yPixel < tileHeight; yPixel++ {
				for xPixel := 0; xPixel < tileWidth; xPixel++ {
					colIndex := palletteTile[yPixel*tileWidth+xPixel]
					col := motherboard.Palettes[0][colIndex]
					rgb := color.RGBA{R: col[0], G: col[1], B: col[2], A: 0xFF}
					idx := (yCursor+yPixel)*pictureWidth + (xCursor + xPixel)
					canvas.Pix[idx] = rgb
				}
			}
			tileNum += 16
		}
	}

}
