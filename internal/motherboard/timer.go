/*
* Implements the Timer functionality of the Gameboy
 */

package motherboard

type Timer struct {
	DivCounter  OpCycles // Divider counter
	TimaCounter OpCycles // Timer counter
	DIV         uint32   // Divider register (0xFF04)
	TIMA        uint32   // Timer counter (0xFF05)
	TMA         uint32   // Timer modulo (0xFF06)
	TAC         uint32   // Timer control (0xFF07)
}

func NewTimer() *Timer {
	return &Timer{
		DivCounter: 0,
		DIV:        0x18,
		TIMA:       0x00,
		TMA:        0x00,
		TAC:        0xF8,
	}
}

func (t *Timer) Reset() {
	t.DivCounter = 0
	t.TimaCounter = 0
	t.DIV = 0x00
	t.TIMA = 0x00
	t.TMA = 0x00
	t.TAC = 0x00
}

func (t *Timer) Enabled() bool {
	return (t.TAC>>2)&1 == 1
}

func (t *Timer) getClockFreqCount() OpCycles {
	switch t.TAC & 0x03 {
	case 0x00:
		return OpCycles(1024)
	case 0x01:
		return OpCycles(16)
	case 0x02:
		return OpCycles(64)
	default:
		return OpCycles(256)
	}

	// idx := t.TAC & 0b11
	// return t.Dividers[idx]
}

func (t *Timer) updateDividerRegister(cycles OpCycles) {
	t.DivCounter += cycles

	if t.DivCounter >= 255 {
		t.DivCounter -= 255
		t.DIV++
	}
}

func (t *Timer) Tick(cycles OpCycles, c *CPU) {

	t.updateDividerRegister(cycles)

	if t.Enabled() {

		t.TimaCounter += cycles
		freq := t.getClockFreqCount()
		for t.TimaCounter >= freq {
			t.TimaCounter -= freq
			// tima := t.TIMA
			if t.TIMA == 0xFF {
				t.TIMA = t.TMA
				c.SetInterruptFlag(INTR_TIMER)
			} else {
				t.TIMA++
			}
		}
	}

}
