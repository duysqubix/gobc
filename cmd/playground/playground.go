package main

import (
	"fmt"
	"math/rand"

	"github.com/duysqubix/gobc/internal/motherboard"
)

func main() {
	td := make([]uint8, 16)
	for i := 0; i < 16; i++ {
		td[i] = uint8(rand.Intn(256))
	}

	t := motherboard.Tile(td)
	pt := t.ParseTile()
	fmt.Println(pt)

	memory := motherboard.NewInternalRAM(true, true)
	fmt.Printf("TileDataLength: $%X\n", len(memory.TileData()))
	fmt.Printf("TileMapLength: $%X\n", len(memory.TileMap()))

}
