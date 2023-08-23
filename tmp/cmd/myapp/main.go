package main

import (
	example "github.com/duysqubix/mypackage1/pkg"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	example.CPtrs.Init()
	sdl.Init(sdl.INIT_EVERYTHING)
	defer sdl.Quit()

	dw, err := example.CreateDebugWindow("MemoryView", 8*60, 16*36, 1, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED)
	if err != nil {
		panic(err)
	}
	defer dw.CleanUp()
	defer example.CPtrs.FreeAll()

	for {

	}

}
