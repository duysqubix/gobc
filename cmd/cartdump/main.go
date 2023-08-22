/*

This handles cartridge metadata and viewing

*/

package main

import (
	"fmt"
	"os"

	"path/filepath"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
)

var SUPPORTED_ROMS = []string{".gbc", ".gb"}
var logger = internal.Logger

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Please provide path to ROM cartridge")
		os.Exit(1)
	}

	filename := os.Args[1]

	if filename == "" {
		fmt.Println("Please provide a filename")
		os.Exit(1)
	}

	// Open the file
	obj := pathlib.NewPath(filename)

	// check if not file and panic
	is_file, err := obj.IsFile()
	if err != nil {
		panic(err)
	}

	if !is_file {
		panic("Not a file")
	}

	// check if file is supported
	ext := filepath.Ext(filename)
	if !internal.IsInStrArray(ext, SUPPORTED_ROMS) {
		internal.Logger.Panicf("Not a supported ROM: %s", ext)
	}

	fmt.Println("Reading ROM file: ", filename)
	// create cartridge
	cart := cartridge.NewCartridge(obj)

	// check if flag --raw is set
	if internal.IsInStrArray("--raw", os.Args) {
		cart.RawHeaderDump()
	} else {
		file, err := os.Create("cartdump.txt")
		if err != nil {
			panic(err)
		}

		defer file.Close()

		include_nop := false

		if internal.IsInStrArray("--include-nop", os.Args) {
			include_nop = true
		}
		if internal.IsInStrArray("--instruction-set", os.Args) {
			cart.Dump(file)
			cart.DumpInstructionSet(file, include_nop)
		} else {
			cart.Dump(file)
		}

		fmt.Println("Dumped cartridge metadata to cartdump.txt")
	}
}
