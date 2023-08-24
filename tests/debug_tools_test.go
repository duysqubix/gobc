package tests

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	example "github.com/duysqubix/gobc/internal/windows"
)

func TestConvertUint32ToByteSlice(t *testing.T) {
	data := []uint32{0x55, 0x55, 0x55, 0x55}
	expected := []byte{0x55, 0x00, 0x00, 0x00, 0x55, 0x00, 0x00, 0x00, 0x55, 0x00, 0x00, 0x00, 0x55, 0x00, 0x00, 0x00}
	result := example.ConvertUint32ToByteSlice(data)
	if !bytes.Equal(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestMMakeFrameBuffer(t *testing.T) {
	example.CPtrs.Init()
	defer example.CPtrs.FreeAll()
	w, h := 2, 2
	buf, _ := example.MakeFrameBuffer(w, h)
	expected := [][]uint32{
		{0x55, 0x55},
		{0x55, 0x55},
	}
	if !reflect.DeepEqual(buf, expected) {
		t.Errorf("Expected %v, but got %v", expected, buf)
	}
}

func TestDecodeFontBitmap(t *testing.T) {
	example.CPtrs.Init()
	defer example.CPtrs.FreeAll()

	// Mock the os.ReadFile function
	data := []byte("eJzLSM3JyQcABiwCFQ==")
	buffer, _ := example.DecodeFontBitmap(data)

	if len(buffer) != 8 {
		t.Errorf("Width is incorrect, got: %d, want: %d.", len(buffer), 8)
	}
	for i := range buffer {
		if len((buffer)[i]) != 16*256 {
			t.Errorf("Height is incorrect, got: %d, want: %d.", len((buffer)[i]), 16*256)
		}
	}
}

func TestMakeBufferU8(t *testing.T) {
	w, h := 2, 2
	default_value := uint8(1)
	expected := []uint8{1, 1, 1, 1}
	result := example.MakeBufferU8(w*h, default_value)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestMakeBufferU32(t *testing.T) {
	w, h := 2, 2
	default_value := uint32(1)
	expected := []uint32{1, 1, 1, 1}
	result := example.MakeBufferU32(w*h, default_value)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestMakeBufferU8_2D(t *testing.T) {
	w, h := 5, 5
	default_value := uint8(10)
	buffer := example.MakeBufferU8_2D(w, h, default_value)

	if len(buffer) != w {
		t.Errorf("Width is incorrect, got: %d, want: %d.", len(buffer), w)
	}

	for i := 0; i < w; i++ {
		if len(buffer[i]) != h {
			t.Errorf("Height is incorrect at index %d, got: %d, want: %d.", i, len(buffer[i]), h)
		}

		for j := 0; j < h; j++ {
			if buffer[i][j] != default_value {
				t.Errorf("Default value is incorrect at index %d,%d, got: %d, want: %d.", i, j, buffer[i][j], default_value)
			}
		}
	}
}

// Mock functions
var (
	readFile = os.ReadFile
)
