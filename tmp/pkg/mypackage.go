package mypackage

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

// #include <stdlib.h>
import "C"

const (
	SCREEN_ROWS = 144
	SCREEN_COLS = 160
)

func ConvertUint32ToByteSlice(data []uint32) []byte {
	buf := new(bytes.Buffer)
	for _, v := range data {
		binary.Write(buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

func MakeBufferU8(size int, default_value uint8) []uint8 {
	buf := make([]uint8, size)
	for i := range buf {
		buf[i] = default_value
	}
	return buf
}

func MakeBufferU32(size int, default_value uint32) []uint32 {
	buf := make([]uint32, size)
	for i := range buf {
		buf[i] = default_value
	}
	return buf
}

func MakeBufferU8_2D(w int, h int, default_value uint8) [][]uint8 {
	buf := make([][]uint8, w)
	fmt.Println(len(buf))
	for i := 0; i < w; i++ {
		buf[i] = MakeBufferU8(h, default_value)
	}
	return buf
}

func MakeFrameBuffer(w, h int) ([][]uint32, *unsafe.Pointer) {
	// buf0 := make([]uint32, w*h*4)
	buf0 := MakeBufferU32(w*h*4, 0x55)
	buf1 := make([][]uint32, w)

	for i := range buf1 {
		buf1[i] = []uint32(buf0[i*h : (i+1)*h])
	}

	// convert []uint32 to []byte
	byteBuf := new(bytes.Buffer)
	for _, v := range buf0 {
		binary.Write(byteBuf, binary.LittleEndian, v)
	}

	buf_p := C.CBytes(byteBuf.Bytes())
	CPtrs.Add("fbuf0_p", buf_p)
	return buf1, &buf_p
}

func DecodeFontBitmap(compdata []byte) ([][]uint32, *unsafe.Pointer) {
	// Your base64 encoded, zlib compressed string
	// encoded := "eJzLSM3JyQcABiwCFQ=="

	// Base64 decode
	decoded, err := base64.StdEncoding.DecodeString(string(compdata))
	if err != nil {
		panic(err)
	}

	// Zlib decompress
	b := bytes.NewReader(decoded)
	r, err := zlib.NewReader(b)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Read the decompressed data
	data, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	buf, buf_p := MakeFrameBuffer(8, 16*256)
	for y := 0; y < len(data); y++ {
		b := data[y]
		for x := 0; x < 8; x++ {
			if ((0x80 >> x) & b) != 0 {
				buf[x][y] = 0xffffffff
			} else {
				buf[x][y] = 0x00000000
			}
		}
	}
	return buf, buf_p
}

var CPtrs CPointers

type CPointers struct {
	Ptrs map[string]unsafe.Pointer
}

func (c *CPointers) Init() {
	if c.Ptrs == nil {
		c.Ptrs = make(map[string]unsafe.Pointer)
	}
}

func (c *CPointers) Add(name string, ptr unsafe.Pointer) {
	c.Ptrs[name] = ptr
}

func (c *CPointers) Get(name string) unsafe.Pointer {
	return c.Ptrs[name]
}

func (c *CPointers) FreeAll() {
	for _, ptr := range c.Ptrs {
		C.free(ptr)
	}
}

type DebugWindow struct {
	window      *sdl.Window     // SDL Window
	renderer    *sdl.Renderer   // SDL Renderer
	texture     *sdl.Texture    // SDL Texture
	windowId    uint32          // SDL Window ID
	scale       int8            // Scale of the window
	title       string          // Title of the window
	width       int32           // Width of the window
	height      int32           // Height of the window
	n_cols      int             // Number of columns
	n_rows      int             // Number of rows
	pos_x       int32           // X position of the window
	pos_y       int32           // Y position of the window
	text_buffer [][]uint8       // Text buffer
	fbuf0       [][]uint32      // Framebuffer 0
	fbuf0_p     *unsafe.Pointer // Framebuffer 0 pointer
}

func CreateDebugWindow(title string, width int32, height int32, scale int8, pos_x int32, pos_y int32) (*DebugWindow, error) {
	dw := &DebugWindow{
		title:  title,
		width:  width,
		height: height,
		scale:  scale,
		pos_x:  pos_x,
		pos_y:  pos_y,
		n_cols: 60,
		n_rows: 36,
	}

	_w, err := sdl.CreateWindow(dw.title, dw.pos_x, dw.pos_y, dw.width, dw.height, sdl.WINDOW_SHOWN)
	if err != nil {
		return nil, err
	}
	dw.window = _w
	dw.windowId, err = _w.GetID()

	_r, err := sdl.CreateRenderer(_w, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, err
	}
	_r.SetLogicalSize(dw.width, dw.height)
	dw.renderer = _r

	if err != nil {
		return nil, err
	}

	_t, err := _r.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STATIC, dw.width, dw.height)
	if err != nil {
		return nil, err
	}
	dw.texture = _t
	dw.text_buffer = MakeBufferU8_2D(dw.n_cols, dw.n_rows, 0x00)

	dw.writeBorder()
	////////////////////////////////////////////////////////////////////////////////
	// MemoryView Stuff Now    /////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////
	data, err := os.ReadFile("fontb64.bin")

	dw.fbuf0, dw.fbuf0_p = DecodeFontBitmap(data)

	dw.texture.Update(nil, *dw.fbuf0_p, 8*4)
	dw.texture.SetBlendMode(sdl.BLENDMODE_BLEND)
	dw.texture.SetColorMod(0x00, 0x00, 0x00)
	dw.renderer.SetDrawColor(0xff, 0xff, 0xff, 0xff)

	return dw, nil
}

func (dw *DebugWindow) writeBorder() {
	for x := 0; x < dw.n_cols; x++ {
		dw.text_buffer[0][x] = 0xCD
		dw.text_buffer[2][x] = 0xCD
		dw.text_buffer[dw.n_rows-1][x] = 0xCD
	}

	for y := 0; y < dw.n_rows; y++ {
		dw.text_buffer[y][0] = 0xBA
		dw.text_buffer[y][9] = 0xB3
		dw.text_buffer[y][dw.n_cols-1] = 0xBA
	}
	dw.text_buffer[0][0] = 0xC9
	dw.text_buffer[1][0] = 0xBA
	dw.text_buffer[0][dw.n_cols-1] = 0xBB
	dw.text_buffer[1][dw.n_cols-1] = 0xBA

	dw.text_buffer[2][0] = 0xCC
	dw.text_buffer[2][9] = 0xD1
	dw.text_buffer[2][dw.n_cols-1] = 0xB9

	dw.text_buffer[dw.n_rows-1][0] = 0xC8
	dw.text_buffer[dw.n_rows-1][9] = 0xCF
	dw.text_buffer[dw.n_rows-1][dw.n_cols-1] = 0xBC

}

// func (dw *DebugWindow) writeMemory() {
// 	for y := 0; y < 32; y++ {
// 		for x := 0; x < 16; x++ {
// 			random_memory_value := uint8(rand.Intn(0xff))
// 		}
// 	}
// }

func (dw *DebugWindow) PostTick() {
	dw.renderer.Clear()
	// dw.renderer.Copy(dw.texture, nil, nil)
	dw.renderer.Present()
}

func (dw *DebugWindow) CleanUp() {
	dw.window.Destroy()
}
