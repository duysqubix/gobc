package internal

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const DMG_CLOCK_SPEED = 4194304 // 4.194304 MHz or 4,194,304 cycles per second
const CGB_CLOCK_SPEED = 8388608 // 8.388608 MHz or 8,388,608 cycles per second
const DEFAULT_LOG_LEVEL = log.ErrorLevel

var Logger = log.New()

func init() {
	// check os.env for LOG_LEVEL
	log_env_set := os.Getenv("LOG_LEVEL")

	strings.ToLower(log_env_set)

	switch log_env_set {
	case "debug":
		Logger.SetLevel(log.DebugLevel)
		Logger.Debug("Log Level set to DEBUG")

	case "info":
		Logger.SetLevel(log.InfoLevel)
		Logger.Info("Log Level set to INFO")

	case "warn":
		Logger.SetLevel(log.WarnLevel)
		Logger.Warn("Log Level set to WARN")
	default:
		Logger.SetLevel(DEFAULT_LOG_LEVEL)
	}
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
