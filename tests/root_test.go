package tests

import (
	"testing"

	"github.com/duysqubix/gobc/internal"
)

func TestIsBitSet(t *testing.T) {
	var value uint8 = 0b10101010
	if !internal.IsBitSet(value, 1) {
		t.Errorf("Expected bit 1 to be set")
	}
	if internal.IsBitSet(value, 0) {
		t.Errorf("Expected bit 0 to be not set")
	}
}

func TestSetBit(t *testing.T) {
	var value uint8 = 0b10101010
	internal.SetBit(&value, 0)
	if !internal.IsBitSet(value, 0) {
		t.Errorf("Expected bit 0 to be set")
	}
}

func TestResetBit(t *testing.T) {
	var value uint8 = 0b10101010
	internal.ResetBit(&value, 1)
	if internal.IsBitSet(value, 1) {
		t.Errorf("Expected bit 1 to be not set")
	}
}
