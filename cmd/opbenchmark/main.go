/*
	Benchmarks for GoBC

*/

package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
)

const ITERATIONS = 1_000_000    // 1 million tests
const DMG_CLOCK_SPEED = 4194304 // 4.194304 MHz or 4,194,304 cycles per second
const CGB_CLOCK_SPEED = 8388608 // 8.388608 MHz or 8,388,608 cycles per second

type BenchMarkResult struct {
	OpCode    motherboard.OpCode
	Iters     int
	AvgCycles float64
	XSpeedDMG float64
	XSpeedCGB float64
	DMGPass   bool
	CGBPass   bool
}

func Benchmark(opcode motherboard.OpCode, f motherboard.OpLogic, mb *motherboard.Motherboard) BenchMarkResult {

	var avg_ccps float64
	var dmg_pass bool = false
	var cgb_pass bool = false
	var xDMG float64
	var xCGB float64

	var xSpeedDMG float64
	var xSpeedCGB float64
	var avg_cycles float64

	for i := 0; i < ITERATIONS; i++ {

		rand_value := uint16(rand.Intn(0xffff))

		start := time.Now()
		cycles := f(mb, rand_value)
		elapsed := time.Since(start)
		elapsed_ns := elapsed.Nanoseconds()

		// avg cycles per second
		avg_ccps = float64(cycles) / (float64(elapsed_ns) / float64(time.Second))

		xDMG = avg_ccps / float64(DMG_CLOCK_SPEED) // how many times faster than DMG ( > 1 is faster)
		xCGB = avg_ccps / float64(CGB_CLOCK_SPEED) // how many times faster than CGB ( > 1 is faster)

		xSpeedDMG += xDMG
		xSpeedCGB += xCGB
		avg_cycles += avg_ccps

	}

	xSpeedDMG = xSpeedDMG / float64(ITERATIONS)
	xSpeedCGB = xSpeedCGB / float64(ITERATIONS)
	avg_cycles = avg_cycles / float64(ITERATIONS)

	if avg_cycles >= float64(DMG_CLOCK_SPEED) {
		dmg_pass = true
	}

	if avg_cycles >= float64(CGB_CLOCK_SPEED) {
		cgb_pass = true
	}

	return BenchMarkResult{
		OpCode:    opcode,
		Iters:     ITERATIONS,
		AvgCycles: avg_cycles,
		DMGPass:   dmg_pass,
		CGBPass:   cgb_pass,
		XSpeedDMG: xSpeedDMG,
		XSpeedCGB: xSpeedCGB,
	}
}
func isInArray(value motherboard.OpCode, array []motherboard.OpCode) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

func printReport(data []BenchMarkResult, filename string) {
	headers := "OpCode,Iters,AvgCycles,DMGPass,CGBPass,XSpeedDMG,XSpeedCGB\n"

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file: ", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(headers)
	if err != nil {
		fmt.Println("Error writing to file: ", err)
		return
	}

	for _, v := range data {
		opcode := v.OpCode
		if opcode > 0xff {
			opcode = opcode - 0xff
		}
		_, err = file.WriteString(fmt.Sprintf("%#x,%d,%f,%t,%t,%f,%f\n", v.OpCode, v.Iters, v.AvgCycles, v.DMGPass, v.CGBPass, v.XSpeedDMG, v.XSpeedCGB))
		if err != nil {
			fmt.Println("Error writing to file: ", err)
			return
		}
	}
}

func main() {

	// an array of illegal opcodes for the Gameboy
	params := &motherboard.MotherboardParams{Decouple: true}
	mb := motherboard.NewMotherboard(params)

	var benchmarks []BenchMarkResult
	var start_opcode uint16 = 0x00
	var end_opcode uint16 = 0xff

	if len(os.Args) > 1 {
		opcode := os.Args[1]
		opcode_i, err := strconv.ParseUint(opcode, 16, 16)

		if err != nil {
			log.Fatal(err)
		}

		start_opcode = uint16(opcode_i)
		end_opcode = uint16(opcode_i)
	}

	for i := start_opcode; i <= end_opcode; i++ {
		i16 := motherboard.OpCode(i)
		if isInArray(i16, motherboard.ILLEGAL_OPCODES) {
			internal.Logger.Infof("Skipping illegal opcode: %#x\n", i)
			continue
		}

		var opfunc motherboard.OpLogic = motherboard.OPCODES[i16]
		benchmark := Benchmark(i16, opfunc, mb)
		benchmarks = append(benchmarks, benchmark)

		dmg := benchmark.DMGPass
		cgb := benchmark.CGBPass

		var cb string = ""
		if i > 0xff {
			cb = "CB "
			i16 = i16 - 0xff
		}
		fmt.Printf("Benchmarking opcode: %s%02X (DMG: %t, CBG: %t )\n", cb, i16, dmg, cgb)
		// break
	}

	printReport(benchmarks, "benchmark.csv")

}
