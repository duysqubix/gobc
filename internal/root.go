package internal

import (
	"fmt"
)

func Panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

func IsBitSet(value uint8, bit uint8) bool {
	return (value & (1 << bit)) != 0
}

func SetBit(value *uint8, bit uint8) {
	*value |= (1 << bit)
}

func ResetBit(value *uint8, bit uint8) {
	*value &= ^(1 << bit)
}

func HalfCarryTest(value *uint8) bool {
	return (*value & 0x0f) == 0x00
}

func FullCarryTest(value *uint8) bool {
	return *value == 0xff
}

func ZeroTest(value *uint8) bool {
	return *value == 0
}
