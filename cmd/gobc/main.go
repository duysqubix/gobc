package main

import (
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/duysqubix/gobc/internal/opcodes"
)

func main() {
	mb := motherboard.NewMotherboard()
	// spew.Dump(mb)

	a := 0x1234

	opcodes.OPCODES[0x00](mb, (uint16)(a))
}
