package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/duysqubix/gobc/internal/motherboard"
)

type Register struct {
	Name   int    `json:"name"`
	A      int    `json:"a"`
	F      int    `json:"f"`
	B      int    `json:"b"`
	C      int    `json:"c"`
	D      int    `json:"d"`
	E      int    `json:"e"`
	HL     int    `json:"hl"`
	SP     int    `json:"sp"`
	PC     int    `json:"pc"`
	ARGS   string `json:"args"`
	CYCLES uint8  `json:"cycles"`
	IME    bool   `json:"ime"`
	IF     uint8  `json:"if"`
	IE     uint8  `json:"ie"`
}

func main() {
	opcode := os.Args[1]
	value := os.Args[2]
	opcode_i, err := strconv.ParseUint(opcode, 16, 16)
	// fmt.Println("opcode: ", opcode, "value: ", value)

	if err != nil {
		log.Fatal(err)
	}

	value_n, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		log.Fatal(err)
	}
	do_opcodes(uint16(opcode_i), uint16(value_n))
}

type MockMB struct {
	m      *motherboard.Motherboard
	args   string
	cycles motherboard.OpCycles
}

func do_opcodes(opCodeNum uint16, value uint16) {
	params := motherboard.MotherboardParams{
		Filename: nil,
	}
	mb := MockMB{
		m:      motherboard.NewMotherboard(&params),
		args:   fmt.Sprint(value),
		cycles: 0,
	}

	// mb := motherboard.NewMotherboard()
	// spew.Dump(mb)
	c := mb.m.Cpu
	c.RandomizeRegisters(int64(time.Now().UnixNano()))
	// c.RandomizeRegisters(1600)

	reg := Register{
		Name:   int(opCodeNum),
		A:      int(c.Registers.A),
		F:      int(c.Registers.F),
		B:      int(c.Registers.B),
		C:      int(c.Registers.C),
		D:      int(c.Registers.D),
		E:      int(c.Registers.E),
		HL:     int(c.Registers.H)<<8 | int(c.Registers.L),
		SP:     int(c.Registers.SP),
		PC:     int(c.Registers.PC),
		ARGS:   fmt.Sprint(mb.args),
		CYCLES: uint8(mb.cycles),
		IME:    c.Interrupts.Master_Enable,
		IF:     c.Interrupts.IF,
		IE:     c.Interrupts.IE,
	}
	jsonData, err := json.MarshalIndent(reg, "", "    ")

	err = os.WriteFile("registers-start.json", jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}
	c.Dump("Initial State")
	op := motherboard.OPCODES[motherboard.OpCode(opCodeNum)]
	mb.cycles = op(mb.m, value) // INC BC

	// name :=
	c.Dump(fmt.Sprintf("Post Instruction [%X]", uint16(opCodeNum)))

	reg = Register{
		Name:   int(opCodeNum),
		A:      int(c.Registers.A),
		F:      int(c.Registers.F),
		B:      int(c.Registers.B),
		C:      int(c.Registers.C),
		D:      int(c.Registers.D),
		E:      int(c.Registers.E),
		HL:     int(c.Registers.H)<<8 | int(c.Registers.L),
		SP:     int(c.Registers.SP),
		PC:     int(c.Registers.PC),
		ARGS:   fmt.Sprint(mb.args),
		CYCLES: uint8(mb.cycles),
		IME:    c.Interrupts.Master_Enable,
		IF:     c.Interrupts.IF,
		IE:     c.Interrupts.IE,
	}
	jsonData, err = json.MarshalIndent(reg, "", "    ")

	err = os.WriteFile("registers-test.json", jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}

}
