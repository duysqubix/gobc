package windows

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/faiface/pixel/pixelgl"
	"github.com/urfave/cli/v2"
)

type Window interface {
	Draw()                // draw the window
	Update() error        // returns error if window should be closed
	SetUp()               // Setup the window
	Win() *pixelgl.Window // returns the pixelgl window
}

func parseRangeBreakpoints(breakpoints string) []uint16 {
	// logger.Debug(breakpoints)
	var parsed []uint16
	start, err := strconv.ParseUint(strings.Split(breakpoints, ":")[0], 16, 16)
	if err != nil {
		errmsg := fmt.Sprintf("Invalid breakpoint format: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}
	end, err := strconv.ParseUint(strings.Split(breakpoints, ":")[1], 16, 16)
	if err != nil {
		errmsg := fmt.Sprintf("Invalid breakpoint format: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}

	for i := start; i <= end; i++ {
		parsed = append(parsed, uint16(i))
	}
	return parsed
}

func parseSingleBreakpoint(breakpoints string) uint16 {

	// logger.Debug(breakpoints)

	// single breakpoint
	addr, err := strconv.ParseUint(breakpoints, 16, 16)
	if addr > 0xffff {
		errmsg := fmt.Sprintf("Addr out of range: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}
	if err != nil {
		errmsg := fmt.Sprintf("Invalid breakpoint format: %s", breakpoints)
		cli.Exit(errmsg, 1)
	}

	return uint16(addr)
}

func ParseBreakpoints(breakpoints string) []uint16 {
	var a []uint16

	split := strings.Split(breakpoints, ",")
	// logger.Debug(split)

	if len(split) == 1 {
		if split[0] == "" {
			return a
		}
		// check if single element is a range
		is_range := strings.Split(split[0], ":")
		if len(is_range) == 2 {
			a = append(a, parseRangeBreakpoints(split[0])...)
		} else {
			// not a range so parse as single breakpoint
			a = append(a, parseSingleBreakpoint(split[0]))
		}
	}
	if len(split) > 1 {
		for _, b := range split {
			if b == "" {
				continue
			}
			// check if single element is a range
			is_range := strings.Split(b, ":")
			if len(is_range) == 2 {
				a = append(a, parseRangeBreakpoints(b)...)
			} else {
				// not a range so parse as single breakpoint
				a = append(a, parseSingleBreakpoint(b))
			}
		}
	}

	// now sort and remove duplicates
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })

	// Remove duplicates
	return removeDuplicates(a)

}

func removeDuplicates(a []uint16) []uint16 {
	j := 0
	for i := 1; i < len(a); i++ {
		if a[j] != a[i] {
			j++
			a[j] = a[i]
		}
	}
	result := a[:j+1]
	return result
}
