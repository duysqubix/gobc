/*
* Implements the Timer functionality of the Gameboy
 */

package motherboard

type Timer struct {
	DIV  uint8  // Divider register (0xFF04)
	TIMA uint16 // Timer counter (0xFF05)
	TMA  uint8  // Timer modulo (0xFF06)
	TAC  uint8  // Timer control (0xFF07)
}

func NewTimer() *Timer {
	return &Timer{
		DIV:  0x18,
		TIMA: 0x00,
		TMA:  0x00,
		TAC:  0xF8,
	}
}

func (t *Timer) Enabled() bool {
	return t.TAC&TAC_ENABLE != 0
}

func (t *Timer) Speed(baseFreq int64) int64 {
	switch t.TAC & 0x03 {
	case TAC_SPEED_1024:
		return baseFreq / 1024
	case TAC_SPEED_16:
		return baseFreq / 16
	case TAC_SPEED_64:
		return baseFreq / 64
	case TAC_SPEED_256:
		return baseFreq / 256
	}
	logger.Panicf("Invalid timer speed: %02x", t.TAC&0x03)
	return 0
}
