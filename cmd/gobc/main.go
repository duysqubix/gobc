package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"

	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/duysqubix/gobc/internal/opcodes"
)

type Register struct {
	Name int    `json:"name"`
	A    int    `json:"a"`
	F    int    `json:"f"`
	B    int    `json:"b"`
	C    int    `json:"c"`
	D    int    `json:"d"`
	E    int    `json:"e"`
	HL   int    `json:"hl"`
	SP   int    `json:"sp"`
	PC   int    `json:"pc"`
	ARGS string `json:"args"`
}

func main() {
	arg := os.Args[1]
	value := os.Args[2]
	num, err := strconv.ParseUint(arg, 16, 16)

	if err != nil {
		log.Fatal(err)
	}

	value_n, err := strconv.ParseUint(value, 16, 16)
	if err != nil {
		log.Fatal(err)
	}
	do_opcodes(uint16(num), uint16(value_n))
}

func do_opcodes(opCodeNum uint16, value uint16) {
	mb := motherboard.NewMotherboard()
	// spew.Dump(mb)
	c := mb.Cpu()
	c.RandomizeRegisters(int64(rand.Intn(0xfffffffffffff)))
	// c.RandomizeRegisters(1600)
	c.Registers.B = 255

	reg := Register{
		Name: int(opCodeNum),
		A:    int(c.Registers.A),
		F:    int(c.Registers.F),
		B:    int(c.Registers.B),
		C:    int(c.Registers.C),
		D:    int(c.Registers.D),
		E:    int(c.Registers.E),
		HL:   int(c.Registers.H)<<8 | int(c.Registers.L),
		SP:   int(c.Registers.SP),
		PC:   int(c.Registers.PC),
		ARGS: fmt.Sprint(value),
	}
	jsonData, err := json.MarshalIndent(reg, "", "    ")

	err = os.WriteFile("registers-start.json", jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}

	op := opcodes.OPCODES[uint16(opCodeNum)]
	op(mb, value) // INC BC

	name := fmt.Sprintf("%04x", uint16(opCodeNum))
	c.Dump(name)

	reg = Register{
		Name: int(opCodeNum),
		A:    int(c.Registers.A),
		F:    int(c.Registers.F),
		B:    int(c.Registers.B),
		C:    int(c.Registers.C),
		D:    int(c.Registers.D),
		E:    int(c.Registers.E),
		HL:   int(c.Registers.H)<<8 | int(c.Registers.L),
		SP:   int(c.Registers.SP),
		PC:   int(c.Registers.PC),
		ARGS: fmt.Sprint(value),
	}
	jsonData, err = json.MarshalIndent(reg, "", "    ")

	err = os.WriteFile("registers-test.json", jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}

}
