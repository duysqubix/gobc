package motherboard

import (
	"bytes"
	"testing"

	"github.com/duysqubix/gobc/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minTimerCPU returns the smallest CPU+Motherboard graph that satisfies
// Timer.Tick's contract: only c.Mb.doubleSpeed and c.Interrupts.IF are
// touched. No Memory, Cartridge, or LCD allocations are required.
func minTimerCPU(doubleSpeed bool) (*CPU, *Motherboard) {
	mb := &Motherboard{doubleSpeed: doubleSpeed}
	cpu := &CPU{
		Registers:  &Registers{},
		Interrupts: &Interrupts{},
		Mb:         mb,
	}
	mb.Cpu = cpu
	return cpu, mb
}

func TestTimer_NewTimerInitialState(t *testing.T) {
	timer := NewTimer()

	assert.Equal(t, OpCycles(0), timer.DivCounter)
	assert.Equal(t, uint32(0x18), timer.DIV, "DIV initial value per cold-boot state")
	assert.Equal(t, uint32(0x00), timer.TIMA)
	assert.Equal(t, uint32(0x00), timer.TMA)
	assert.Equal(t, uint32(0xF8), timer.TAC, "TAC top bits (0xF8) are always read as 1")
}

func TestTimer_ResetClearsAllFields(t *testing.T) {
	timer := NewTimer()
	timer.DivCounter = 123
	timer.TimaCounter = 456
	timer.DIV = 0xAB
	timer.TIMA = 0xCD
	timer.TMA = 0xEF
	timer.TAC = 0xFF

	timer.Reset()

	assert.Equal(t, OpCycles(0), timer.DivCounter)
	assert.Equal(t, OpCycles(0), timer.TimaCounter)
	assert.Equal(t, uint32(0), timer.DIV)
	assert.Equal(t, uint32(0), timer.TIMA)
	assert.Equal(t, uint32(0), timer.TMA)
	assert.Equal(t, uint32(0), timer.TAC)
}

func TestTimer_EnabledFlag(t *testing.T) {
	cases := []struct {
		name string
		tac  uint32
		want bool
	}{
		{"bit 2 clear -> disabled", 0xF8, false},
		{"bit 2 set + freq 00 -> enabled", 0xF8 | 0x04, true},
		{"bit 2 set + freq 11 -> enabled", 0xF8 | 0x07, true},
		{"only freq bits set -> disabled", 0xF8 | 0x03, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			timer := NewTimer()
			timer.TAC = tc.tac
			assert.Equal(t, tc.want, timer.Enabled())
		})
	}
}

func TestTimer_GetClockFreqCount(t *testing.T) {
	// TAC[0:1] selects the TIMA period.
	cases := []struct {
		tacBits uint32
		want    OpCycles
	}{
		{0b00, TAC_SPEED_1024},
		{0b01, TAC_SPEED_16},
		{0b10, TAC_SPEED_64},
		{0b11, TAC_SPEED_256},
	}
	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			timer := NewTimer()
			timer.TAC = 0xF8 | tc.tacBits
			assert.Equal(t, tc.want, timer.getClockFreqCount())
		})
	}
}

func TestTimer_DivIncrementsEvery256Cycles(t *testing.T) {
	// DMG_CLOCK_SPEED / GB_TIMER_FREQ == 4194304 / 16384 == 256 cycles per
	// DIV tick.
	timer := NewTimer()
	timer.Reset()

	timer.updateDividerRegister(255, false)
	assert.Equal(t, uint32(0), timer.DIV, "DIV must not increment below threshold")
	assert.Equal(t, OpCycles(255), timer.DivCounter)

	timer.updateDividerRegister(1, false)
	assert.Equal(t, uint32(1), timer.DIV, "DIV increments when counter reaches 256")
	assert.Equal(t, OpCycles(0), timer.DivCounter)

	// Drive DIV through several more increments to confirm the modulo is stable.
	for i := 2; i <= 10; i++ {
		timer.updateDividerRegister(256, false)
		assert.Equal(t, uint32(i), timer.DIV)
	}
}

func TestTimer_DivWrapsToZeroAt256(t *testing.T) {
	timer := NewTimer()
	timer.Reset()
	timer.DIV = 0xFF

	timer.updateDividerRegister(256, false)
	assert.Equal(t, uint32(0), timer.DIV, "DIV wraps from 0xFF back to 0x00")
}

func TestTimer_DivDoubleSpeedDoublesThreshold(t *testing.T) {
	timer := NewTimer()
	timer.Reset()

	timer.updateDividerRegister(256, true)
	assert.Equal(t, uint32(0), timer.DIV, "in double-speed DIV needs 512 cycles")
	assert.Equal(t, OpCycles(256), timer.DivCounter)

	timer.updateDividerRegister(256, true)
	assert.Equal(t, uint32(1), timer.DIV, "DIV increments after 512 cycles in double speed")
}

func TestTimer_DivWriteResetsToZero(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.Timer.DIV = 0xAA
	mb.Timer.DivCounter = 100
	mb.Timer.TimaCounter = 50

	mb.SetItem(0xFF04, 0x42) // any value resets DIV

	assert.Equal(t, uint32(0), mb.Timer.DIV)
	assert.Equal(t, OpCycles(0), mb.Timer.DivCounter)
	assert.Equal(t, OpCycles(0), mb.Timer.TimaCounter)
	assert.Equal(t, uint8(0), mb.GetItem(0xFF04))
}

func TestTimer_TIMADoesNotIncrementWhenDisabled(t *testing.T) {
	timer := NewTimer()
	timer.Reset()
	cpu, _ := minTimerCPU(false)

	timer.TAC = 0x00 // bit 2 clear -> disabled
	timer.TIMA = 0
	for i := 0; i < 8; i++ {
		timer.Tick(1024, cpu)
	}
	assert.Equal(t, uint32(0), timer.TIMA)
	assert.Equal(t, uint8(0), cpu.Interrupts.IF&(1<<INTR_TIMER))
}

func TestTimer_TIMAEnableIncrementsAtSelectedFrequency(t *testing.T) {
	cases := []struct {
		name string
		bits uint32
		freq OpCycles
	}{
		{"freq=00 -> 1024 cycles", 0b00, 1024},
		{"freq=01 -> 16 cycles", 0b01, 16},
		{"freq=10 -> 64 cycles", 0b10, 64},
		{"freq=11 -> 256 cycles", 0b11, 256},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			timer := NewTimer()
			timer.Reset()
			cpu, _ := minTimerCPU(false)
			timer.TAC = uint32(TAC_ENABLE) | tc.bits
			timer.TIMA = 0
			timer.TMA = 0

			timer.Tick(tc.freq-1, cpu)
			assert.Equal(t, uint32(0), timer.TIMA, "must not tick before freq cycles elapse")

			timer.Tick(1, cpu)
			assert.Equal(t, uint32(1), timer.TIMA, "must tick exactly at freq cycles")
			assert.Equal(t, uint8(0), cpu.Interrupts.IF&(1<<INTR_TIMER),
				"no IF bit set on plain increment")

			timer.Tick(tc.freq, cpu)
			assert.Equal(t, uint32(2), timer.TIMA, "increments per freq elapsed")
		})
	}
}

func TestTimer_TIMAOverflowResetsToTMAAndRaisesIF(t *testing.T) {
	timer := NewTimer()
	timer.Reset()
	cpu, _ := minTimerCPU(false)

	timer.TAC = uint32(TAC_ENABLE) // 1024 cycle period
	timer.TIMA = 0xFF
	timer.TMA = 0x42

	timer.Tick(1024, cpu)

	assert.Equal(t, uint32(0x42), timer.TIMA, "TIMA reloads from TMA after overflow")
	assert.NotZero(t, cpu.Interrupts.IF&(1<<INTR_TIMER),
		"IF Timer bit (bit 2) must be set on TIMA overflow")
	// SetInterruptFlag also forces the high 3 unused IF bits to 1.
	assert.Equal(t, uint8(0xE0), cpu.Interrupts.IF&0xE0,
		"unused IF bits 5..7 are always read as 1")
}

func TestTimer_TIMAOverflowFromIntermediateValue(t *testing.T) {
	timer := NewTimer()
	timer.Reset()
	cpu, _ := minTimerCPU(false)

	// freq=01 -> 16 cycle period. Start TIMA two ticks below overflow and
	// drive 32 cycles: first tick takes 0xFE -> 0xFF, second tick triggers
	// overflow and reloads from TMA.
	timer.TAC = uint32(TAC_ENABLE) | 0b01
	timer.TIMA = 0xFE
	timer.TMA = 0xA0

	timer.Tick(32, cpu)
	assert.Equal(t, uint32(0xA0), timer.TIMA)
	assert.NotZero(t, cpu.Interrupts.IF&(1<<INTR_TIMER))
}

func TestTimer_TACWritePreservesUpperBits(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.SetItem(0xFF07, 0x05)
	assert.Equal(t, uint32(0xFD), mb.Timer.TAC, "TAC writes OR in 0xF8 (bits 3..7)")
	assert.Equal(t, uint8(0xFD), mb.GetItem(0xFF07))

	mb.SetItem(0xFF07, 0x00)
	assert.Equal(t, uint32(0xF8), mb.Timer.TAC, "even a zero write keeps bits 3..7 set")
}

func TestTimer_TACWriteChangingFrequencyResetsTimaCounter(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.SetItem(0xFF07, uint16(TAC_ENABLE)|0b01)
	mb.Timer.TimaCounter = 10

	mb.SetItem(0xFF07, uint16(TAC_ENABLE)|0b10)
	assert.Equal(t, OpCycles(0), mb.Timer.TimaCounter,
		"changing freq bits clears TimaCounter")
}

func TestTimer_TACWriteSameFrequencyKeepsTimaCounter(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.SetItem(0xFF07, uint16(TAC_ENABLE)|0b01)
	mb.Timer.TimaCounter = 10

	mb.SetItem(0xFF07, uint16(TAC_ENABLE)|0b01) // re-enable, same freq bits
	assert.Equal(t, OpCycles(10), mb.Timer.TimaCounter,
		"identical freq bits leave TimaCounter untouched")
}

func TestTimer_TIMAAndTMAWritesPersist(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.SetItem(0xFF05, 0x77)
	mb.SetItem(0xFF06, 0x88)

	assert.Equal(t, uint32(0x77), mb.Timer.TIMA)
	assert.Equal(t, uint32(0x88), mb.Timer.TMA)
	assert.Equal(t, uint8(0x77), mb.GetItem(0xFF05))
	assert.Equal(t, uint8(0x88), mb.GetItem(0xFF06))
}

func TestTimer_TickViaMotherboardSetsTimerInterrupt(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Timer.TAC = uint32(TAC_ENABLE)
	mb.Timer.TIMA = 0xFF
	mb.Timer.TMA = 0x10

	mb.Timer.Tick(1024, mb.Cpu)

	assert.Equal(t, uint32(0x10), mb.Timer.TIMA)
	assert.NotZero(t, mb.Cpu.Interrupts.IF&(1<<INTR_TIMER))
	// The IF register read through the bus also includes the forced top
	// bits (per IF: bit 7..5 are always read as 1).
	assert.Equal(t, uint8(0xE4), mb.GetItem(0xFF0F)&0xE7,
		"IF bus read: bits 7..5 high, bit 2 (Timer) high, others clear")
}

func TestTimer_SerializeRoundTrip(t *testing.T) {
	original := NewTimer()
	original.DivCounter = 100
	original.TimaCounter = 200
	original.DIV = 0xAA
	original.TIMA = 0xBB
	original.TMA = 0xCC
	original.TAC = 0xDD

	buf := original.Serialize()
	require.NotNil(t, buf)

	restored := NewTimer()
	require.NoError(t, restored.Deserialize(bytes.NewBuffer(buf.Bytes())))

	assert.Equal(t, original.DivCounter, restored.DivCounter)
	assert.Equal(t, original.TimaCounter, restored.TimaCounter)
	assert.Equal(t, original.DIV, restored.DIV)
	assert.Equal(t, original.TIMA, restored.TIMA)
	assert.Equal(t, original.TMA, restored.TMA)
	assert.Equal(t, original.TAC, restored.TAC)
}

func TestTimer_DeserializeReturnsErrorOnShortBuffer(t *testing.T) {
	timer := NewTimer()
	err := timer.Deserialize(bytes.NewBuffer([]byte{0x01}))
	require.Error(t, err)
}

// Sanity check that the clock-speed constants the timer relies on are the
// canonical Game Boy values. If anyone changes these in internal/root.go the
// timer behaviour above silently shifts, so we pin them here.
func TestTimer_ClockConstants(t *testing.T) {
	assert.Equal(t, 4194304, internal.DMG_CLOCK_SPEED)
	assert.Equal(t, 16384, internal.GB_TIMER_FREQ)
	assert.Equal(t, 4194304/16384, 256, "256 CPU cycles per DIV increment in single-speed")
}
