package internal

import (
	log "github.com/sirupsen/logrus"
)

const DMG_CLOCK_SPEED = 4194304 // 4.194304 MHz or 4,194,304 cycles per second
const CGB_CLOCK_SPEED = 8388608 // 8.388608 MHz or 8,388,608 cycles per second

var Logger = log.New()

func init() {
	Logger.SetLevel(log.DebugLevel) // change to be read from OS env
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

func ToggleBit(value *uint8, bit uint8) {
	*value ^= (1 << bit)
}

func IsInStrArray(value string, array []string) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}
