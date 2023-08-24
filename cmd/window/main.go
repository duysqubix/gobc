// Copyright 2017 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/duysqubix/gobc/internal"
)

const (
	screenWidth  = internal.SCREEN_WIDTH
	screenHeight = internal.SCREEN_HEIGHT
)

var data []uint8

func init() {
	data = make([]uint8, 0xffff)

	for i := 0; i < 0xffff; i++ {
		data[i] = uint8(rand.Intn(0xff))
	}
}

type Gobc struct {
	memory []uint8
	y      float64
}

func (g *Gobc) Update() error {
	_, dy := ebiten.Wheel()
	g.y -= dy
	if g.y < 0 {
		g.y = 0.0
	}

	return nil
}

func (g *Gobc) Draw(screen *ebiten.Image) {
	// op := &ebiten.DrawImageOptions{}
	// op.GeoM.Translate(float64(g.x), float64(g.y))
	// screen.DrawImage(pointerImage, op)

	// msg := fmt.Sprintf("TPS: %0.2f, x: %d, y: %d", ebiten.ActualTPS(), g.x, g.y)
	// ebitenutil.DebugPrint(screen, msg)
	// ebitenutil.Print(screen, msg)
	// ebitenutil.DebugPrint(screen, fmt.Sprint(g.y))

	ebitenutil.DebugPrintAt(screen, "Addr | Values", 0, 0)
	ebitenutil.DebugPrintAt(screen, "-----+-------", 0, 16)
	max_y := screen.Bounds().Max.Y
	max_x := screen.Bounds().Max.X

	max_rows := max_y / 16
	max_cols := max_x / 6
	max_cols += 1

	// print rows from memory
	for i := 0; i < max_rows; i++ {
		// print address
		row_addr_start := i + int(g.y)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%04x", row_addr_start), 0, 29+(i*16))

		for j := 0; j < 4; j++ {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%02x", g.memory[j+row_addr_start]), 45+(j*16), 29+(i*16))
		}

	}
	// ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Max Rows: %d, Max Cols: %d", max_rows, max_cols), 0, 32)

}

func (g *Gobc) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Gobc{memory: data, y: 0}

	ebiten.SetWindowSize(screenWidth*4, screenHeight*4)
	ebiten.SetWindowTitle("Gobc")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
