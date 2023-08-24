package windows

import (
	"github.com/duysqubix/gobc/internal"
	"golang.org/x/image/font/basicfont"
)

const (
	screenWidth  = 670
	screenHeight = 1000
	scale        = 1
	trueWidth    = screenWidth * scale
	trueHeight   = screenHeight * scale
	fontBuffer   = 4
	addr_offset  = 10
)

var (
	defaultFont *basicfont.Face
	logger      = internal.Logger
	mock_memory []uint8
	max_chars   int
	max_rows    int
	row_ptr     int = 1 // starting at the top of the screen
	begin_addr  int
	end_addr    int
)
