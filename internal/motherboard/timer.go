/*
* Implements the Timer functionality of the Gameboy
 */

package motherboard

type timerDivider [4]OpCycles
type timerRegister uint16

type Timer struct {
	DivCounter  OpCycles     // Divider counter
	TimaCounter OpCycles     // Timer counter
	DIV         uint16       // Divider register (0xFF04)
	TIMA        uint16       // Timer counter (0xFF05)
	TMA         uint16       // Timer modulo (0xFF06)
	TAC         uint16       // Timer control (0xFF07)
	Dividers    timerDivider // Dividers for each timer speed
}

func NewTimer() *Timer {
	return &Timer{
		DivCounter: 0,
		DIV:        0x18,
		TIMA:       0x00,
		TMA:        0x00,
		TAC:        0xF8,
		Dividers:   timerDivider{TAC_SPEED_1024, TAC_SPEED_16, TAC_SPEED_64, TAC_SPEED_256},
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
	return t.TAC&0b100 == 0b100
}

func (t *Timer) GetDivider() OpCycles {
	idx := t.TAC & 0b11
	return t.Dividers[idx]
}

func (t *Timer) Tick(cycles OpCycles) bool {

	t.DivCounter += cycles
	t.DIV += uint16(t.DivCounter >> 8)
	t.DivCounter &= 0xFF
	t.DIV &= 0xFF

	// check if timer is enabled
	if !t.Enabled() {
		return false
	}

	t.TimaCounter += cycles
	divider := t.GetDivider()

	if t.TimaCounter >= divider {
		t.TimaCounter -= divider
		t.TIMA++

		if t.TIMA > 0xFF {
			t.TIMA = t.TMA
			t.TIMA &= 0xFF
			return true
		}
	}
	return false
}

func (t *Timer) CyclesToInterrupt() OpCycles {
	if t.TAC&0b100 == 0 {
		return 1 >> 16
	}

	divider := t.GetDivider()
	cyclesLeft := OpCycles((0x100-t.TIMA)*uint16(divider)) - t.TimaCounter

	return cyclesLeft
}
