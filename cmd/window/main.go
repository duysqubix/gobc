package main

import (
	"fmt"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

func update() {

}

func draw() {

}

func run() {

	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "Pixel Rocks!",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	},
	)

	if err != nil {
		panic(err)
	}

	basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	basicTxt := text.New(pixel.V(100, 500), basicAtlas)

	for !win.Closed() {
		win.Clear(colornames.Black)
		dot := basicTxt.Dot
		fmt.Fprintln(basicTxt, "Hello, text!")
		fmt.Fprintln(basicTxt, "I support multiple lines!")
		fmt.Fprintf(basicTxt, "And I'm an %s, yay!", "io.Writer")

		basicTxt.Draw(win, pixel.IM.Scaled(basicTxt.Orig, 4))
		basicTxt.Clear()
		basicTxt.Dot = dot
		win.Update()
	}
}

func GameLoop() {
	// windowHeight := float64(gameHeight * scale)
	// windowWidth := float64(gameWidth * scale)
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:  "gobc",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	})

	if err != nil {
		// logger.Panicf("Failed to create window: %s", err)
		panic(err)
	}

	for !win.Closed() {

		win.Clear(colornames.White)

		// Update(gobc)
		// Draw(gobc)
		win.Update()
	}
}

func main() {
	pixelgl.Run(GameLoop)
}
