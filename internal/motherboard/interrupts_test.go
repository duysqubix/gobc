package motherboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInterrupts_BitConstants pins the five interrupt source numbers to the
// bit positions declared by Pan Docs (IE/IF). If these shift, every consumer
// of SetInterruptFlag silently targets the wrong bit, so we lock them down.
func TestInterrupts_BitConstants(t *testing.T) {
	assert.Equal(t, uint8(0), INTR_VBLANK, "VBlank -> IF bit 0")
	assert.Equal(t, uint8(1), INTR_LCDSTAT, "LCD STAT -> IF bit 1")
	assert.Equal(t, uint8(2), INTR_TIMER, "Timer overflow -> IF bit 2")
	assert.Equal(t, uint8(3), INTR_SERIAL, "Serial transfer -> IF bit 3")
	assert.Equal(t, uint8(4), INTR_HIGHTOLOW, "Joypad hi-to-lo -> IF bit 4")
}

func TestInterrupts_AddressConstants(t *testing.T) {
	assert.Equal(t, uint16(0x0040), INTR_VBLANK_ADDR)
	assert.Equal(t, uint16(0x0048), INTR_LCDSTAT_ADDR)
	assert.Equal(t, uint16(0x0050), INTR_TIMER_ADDR)
	assert.Equal(t, uint16(0x0058), INTR_SERIAL_ADDR)
	assert.Equal(t, uint16(0x0060), INTR_HIGHTOLOW_ADDR)
}

func TestInterrupts_FlagAndEnableMasks(t *testing.T) {
	cases := []struct {
		source  uint8
		bitMask uint8
	}{
		{INTR_VBLANK, 0x01},
		{INTR_LCDSTAT, 0x02},
		{INTR_TIMER, 0x04},
		{INTR_SERIAL, 0x08},
		{INTR_HIGHTOLOW, 0x10},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.bitMask, uint8(1)<<tc.source,
			"source %d must map to bit mask %#02x", tc.source, tc.bitMask)
	}
}

func TestInterrupts_SetInterruptFlagSetsTargetBitAndForcesUpperBits(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.IF = 0x00

	mb.Cpu.SetInterruptFlag(INTR_VBLANK)

	// SetInterruptFlag does `IF | 0xE0 | (1<<f)` so the source bit is set
	// AND the three unused IF bits (5..7) are forced high.
	assert.Equal(t, uint8(0x01), mb.Cpu.Interrupts.IF&0x1F,
		"low 5 bits: only VBlank bit set")
	assert.Equal(t, uint8(0xE0), mb.Cpu.Interrupts.IF&0xE0,
		"upper 3 bits always forced high")
}

func TestInterrupts_SetInterruptFlagIsIdempotent(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.IF = 0x00

	mb.Cpu.SetInterruptFlag(INTR_TIMER)
	first := mb.Cpu.Interrupts.IF

	mb.Cpu.SetInterruptFlag(INTR_TIMER)
	assert.Equal(t, first, mb.Cpu.Interrupts.IF,
		"setting the same flag twice must not change IF")
}

func TestInterrupts_SetInterruptFlagsCoexist(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.IF = 0x00

	mb.Cpu.SetInterruptFlag(INTR_VBLANK)
	mb.Cpu.SetInterruptFlag(INTR_TIMER)
	mb.Cpu.SetInterruptFlag(INTR_HIGHTOLOW)

	assert.Equal(t, uint8(0x15), mb.Cpu.Interrupts.IF&0x1F,
		"VBlank(0)+Timer(2)+Joypad(4) = 0b00010101")
}

func TestInterrupts_CheckValidInterruptsMasksIEAgainstIF(t *testing.T) {
	cases := []struct {
		ie, ifv, want uint8
	}{
		{0x00, 0xFF, 0x00},
		{0xFF, 0x00, 0x00},
		{0x05, 0x05, 0x05},
		{0x1F, 0x04, 0x04},
		{0x04, 0x1F, 0x04},
		{0xFF, 0xFF, 0xFF},
	}
	for _, tc := range cases {
		i := &Interrupts{IE: tc.ie, IF: tc.ifv}
		assert.Equal(t, tc.want, i.CheckValidInterrupts(),
			"IE=%02X IF=%02X", tc.ie, tc.ifv)
	}
}

func TestInterrupts_IFRegisterBusReadForcesUpperBitsHigh(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.IF = 0x05

	got := mb.GetItem(0xFF0F)
	assert.Equal(t, uint8(0xE5), got,
		"IF bus read returns underlying IF OR 0xE0")
}

func TestInterrupts_NoServiceWhenIMEDisabled(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.InterruptsOn = false
	mb.Cpu.Interrupts.InterruptsEnabling = false
	mb.Cpu.Halted = false
	mb.Cpu.Interrupts.IE = 0xFF
	mb.Cpu.Interrupts.IF = 0x01 // VBlank pending
	pcBefore := mb.Cpu.Registers.PC
	spBefore := mb.Cpu.Registers.SP

	cycles := mb.Cpu.handleInterrupts()

	assert.Equal(t, OpCycles(0), cycles, "no cycles consumed when IME=0")
	assert.Equal(t, pcBefore, mb.Cpu.Registers.PC, "PC unchanged")
	assert.Equal(t, spBefore, mb.Cpu.Registers.SP, "SP unchanged")
	assert.Equal(t, uint8(0x01), mb.Cpu.Interrupts.IF, "IF unchanged")
}

func TestInterrupts_NoServiceWhenSourceMaskedInIE(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.InterruptsOn = true
	mb.Cpu.Interrupts.IE = 0x00 // every source masked
	mb.Cpu.Interrupts.IF = 0x1F
	pcBefore := mb.Cpu.Registers.PC

	cycles := mb.Cpu.handleInterrupts()

	assert.Equal(t, OpCycles(0), cycles)
	assert.Equal(t, pcBefore, mb.Cpu.Registers.PC)
	assert.Equal(t, uint8(0x1F), mb.Cpu.Interrupts.IF, "IF unchanged when masked")
}

func TestInterrupts_HandleInterruptsConsumes20CyclesOnService(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.InterruptsOn = true
	mb.Cpu.Interrupts.IE = 0x01
	mb.Cpu.Interrupts.IF = 0x01

	cycles := mb.Cpu.handleInterrupts()
	assert.Equal(t, OpCycles(20), cycles, "servicing an interrupt costs 20 cycles")
}

// TestInterrupts_PriorityOrdering pins the documented Pan Docs ordering:
// when multiple IF bits are set AND enabled, the lowest bit (VBlank=0) is
// serviced first; then LCDSTAT, Timer, Serial, Joypad.
func TestInterrupts_PriorityOrdering(t *testing.T) {
	cases := []struct {
		name      string
		ie, ifv   uint8
		wantVec   uint16
		wantIFBit uint8
	}{
		{"vblank wins over lcdstat", 0x03, 0x03, INTR_VBLANK_ADDR, 0x01},
		{"lcdstat wins over timer", 0x06, 0x06, INTR_LCDSTAT_ADDR, 0x02},
		{"timer wins over serial", 0x0C, 0x0C, INTR_TIMER_ADDR, 0x04},
		{"serial wins over joypad", 0x18, 0x18, INTR_SERIAL_ADDR, 0x08},
		{"joypad serviced alone", 0x10, 0x10, INTR_HIGHTOLOW_ADDR, 0x10},
		{"all pending -> vblank wins", 0x1F, 0x1F, INTR_VBLANK_ADDR, 0x01},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mb := newMbForSubsysTest(t)
			mb.Cpu.Registers.PC = 0x1234
			mb.Cpu.Registers.SP = 0xFFFE
			mb.Cpu.Interrupts.InterruptsOn = true
			mb.Cpu.Interrupts.IE = tc.ie
			mb.Cpu.Interrupts.IF = tc.ifv

			mb.Cpu.handleInterrupts()

			assert.Equal(t, tc.wantVec, mb.Cpu.Registers.PC,
				"PC must vector to the highest-priority enabled interrupt")
			assert.Zero(t, mb.Cpu.Interrupts.IF&tc.wantIFBit,
				"serviced IF bit must be cleared")
			assert.False(t, mb.Cpu.Interrupts.InterruptsOn,
				"IME must be disabled after servicing")
		})
	}
}

func TestInterrupts_ServiceClearsIFAndPushesPCAndVectors(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Registers.PC = 0x1234
	mb.Cpu.Registers.SP = 0xFFFE
	mb.Cpu.Interrupts.InterruptsOn = true
	mb.Cpu.Interrupts.IE = 0x01
	mb.Cpu.Interrupts.IF = 0x01

	mb.Cpu.ServiceInterrupt(INTR_VBLANK)

	assert.Zero(t, mb.Cpu.Interrupts.IF&0x01, "VBlank IF bit cleared")
	assert.Equal(t, uint16(0x0040), mb.Cpu.Registers.PC, "PC vectors to 0x0040")
	assert.Equal(t, uint16(0xFFFC), mb.Cpu.Registers.SP, "SP -= 2")
	assert.False(t, mb.Cpu.Interrupts.InterruptsOn, "IME cleared by service")

	// Pushed PC: high byte at SP-1 (0xFFFD), low byte at SP-2 (0xFFFC),
	// where SP was 0xFFFE before push.
	assert.Equal(t, uint8(0x12), mb.GetItem(0xFFFD), "high byte of old PC pushed")
	assert.Equal(t, uint8(0x34), mb.GetItem(0xFFFC), "low byte of old PC pushed")
}

func TestInterrupts_ServiceEachSourceVectorsCorrectly(t *testing.T) {
	cases := []struct {
		source uint8
		vector uint16
	}{
		{INTR_VBLANK, INTR_VBLANK_ADDR},
		{INTR_LCDSTAT, INTR_LCDSTAT_ADDR},
		{INTR_TIMER, INTR_TIMER_ADDR},
		{INTR_SERIAL, INTR_SERIAL_ADDR},
		{INTR_HIGHTOLOW, INTR_HIGHTOLOW_ADDR},
	}
	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			mb := newMbForSubsysTest(t)
			mb.Cpu.Registers.PC = 0x0200
			mb.Cpu.Registers.SP = 0xFFFE
			mb.Cpu.Interrupts.InterruptsOn = true
			mb.Cpu.Interrupts.IE = 1 << tc.source
			mb.Cpu.Interrupts.IF = 1 << tc.source

			mb.Cpu.ServiceInterrupt(tc.source)
			assert.Equal(t, tc.vector, mb.Cpu.Registers.PC,
				"interrupt %d must vector to %#04x", tc.source, tc.vector)
		})
	}
}

func TestInterrupts_ServiceNoOpWhenIMEDisabledAndNotHalted(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Registers.PC = 0x0100
	mb.Cpu.Registers.SP = 0xFFFE
	mb.Cpu.Interrupts.InterruptsOn = false
	mb.Cpu.Halted = false
	mb.Cpu.Interrupts.IE = 0xFF
	mb.Cpu.Interrupts.IF = 0x01

	mb.Cpu.ServiceInterrupt(INTR_VBLANK)

	assert.Equal(t, uint16(0x0100), mb.Cpu.Registers.PC, "PC unchanged")
	assert.Equal(t, uint16(0xFFFE), mb.Cpu.Registers.SP, "SP unchanged")
	assert.Equal(t, uint8(0x01), mb.Cpu.Interrupts.IF, "IF unchanged")
}

func TestInterrupts_ServiceWhenHaltedClearsHaltedAndAdvancesPC(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Halted = true
	mb.Cpu.Registers.PC = 0x1234
	mb.Cpu.Registers.SP = 0xFFFE
	mb.Cpu.Interrupts.InterruptsOn = false
	mb.Cpu.Interrupts.IF = 0x01

	mb.Cpu.ServiceInterrupt(INTR_VBLANK)

	assert.False(t, mb.Cpu.Halted, "Halted cleared")
	assert.Equal(t, uint16(0x1235), mb.Cpu.Registers.PC, "PC advances by 1")
	assert.Equal(t, uint16(0xFFFE), mb.Cpu.Registers.SP, "SP unchanged (no push)")
	assert.Equal(t, uint8(0x01), mb.Cpu.Interrupts.IF, "IF unchanged on halt path")
}

func TestInterrupts_HandleInterruptsConsumesEnablingFlagOnFirstTick(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Cpu.Interrupts.InterruptsEnabling = true
	mb.Cpu.Interrupts.InterruptsOn = false
	mb.Cpu.Interrupts.IE = 0xFF
	mb.Cpu.Interrupts.IF = 0x01

	cycles := mb.Cpu.handleInterrupts()

	assert.Equal(t, OpCycles(0), cycles, "enable-pending tick costs 0 cycles")
	assert.True(t, mb.Cpu.Interrupts.InterruptsOn, "IME now on")
	assert.False(t, mb.Cpu.Interrupts.InterruptsEnabling, "pending flag cleared")
	assert.Equal(t, uint8(0x01), mb.Cpu.Interrupts.IF,
		"IF unchanged on the enabling tick - no service yet")
}

func TestInterrupts_ReportOnReturnsExpectedShape(t *testing.T) {
	i := &Interrupts{IE: 0x05, IF: 0x01}

	row := i.ReportOn(INTR_VBLANK)
	assert.Equal(t, 4, len(row))
	assert.Equal(t, "VBLNK", row[0])
	assert.Equal(t, "ON", row[1], "VBlank both enabled and flagged")

	row = i.ReportOn(INTR_TIMER)
	assert.Equal(t, "TIMER", row[0])
	assert.Equal(t, "OFF", row[1], "Timer enabled but not flagged -> OFF")

	row = i.ReportOn(INTR_LCDSTAT)
	assert.Equal(t, "LCDST", row[0])
	assert.Equal(t, "OFF", row[1], "LCD STAT not enabled -> OFF")

	row = i.ReportOn(99) // out-of-range
	assert.Equal(t, []string{"", "", "", ""}, row, "unknown source returns empty row")
}

func TestInterrupts_IERegisterRoundtripsViaBus(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFFFF, 0x1F)
	assert.Equal(t, uint8(0x1F), mb.Cpu.Interrupts.IE)
	assert.Equal(t, uint8(0x1F), mb.GetItem(0xFFFF))
}

func TestInterrupts_IFRegisterWriteUpdatesUnderlyingIF(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFF0F, 0x07)
	assert.Equal(t, uint8(0x07), mb.Cpu.Interrupts.IF,
		"writes to 0xFF0F store the literal value (no upper-bit OR on write)")
	assert.Equal(t, uint8(0xE7), mb.GetItem(0xFF0F),
		"but bus reads still OR in 0xE0")
}
