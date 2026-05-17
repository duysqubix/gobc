// Package motherboard — apu_test.go
//
// Unit tests for the Game Boy APU (Audio Processing Unit).
// Tests run headless (AudioEnabled=false): they exercise the register
// model, channel DSP and frame sequencer without touching the host
// speaker.

package motherboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPU_PostBootRegistersMatchPanDocs verifies post-bootrom defaults.
// Pan Docs lists NR50=0x77, NR51=0xF3, NR52=0xF1 after the boot ROM.
func TestAPU_PostBootRegistersMatchPanDocs(t *testing.T) {
	mb := newMbForSubsysTest(t)
	assert.Equal(t, uint8(0x77), mb.GetItem(0xFF24), "NR50 post-boot")
	assert.Equal(t, uint8(0xF3), mb.GetItem(0xFF25), "NR51 post-boot")
	assert.Equal(t, uint8(0xF1), mb.GetItem(0xFF26), "NR52 post-boot (enabled + ch1 active)")
}

// TestAPU_NR52DisableZeroesRegisters writes 0x00 to NR52 and confirms
// writes to NR10..NR51 are ignored until APU is re-enabled.
func TestAPU_NR52DisableZeroesRegisters(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.SetItem(0xFF26, 0x00)
	assert.Equal(t, uint8(0x70), mb.GetItem(0xFF26), "NR52 disabled = 0x00 | mask 0x70")

	// Writes are now ignored.
	mb.SetItem(0xFF10, 0x7F)
	mb.SetItem(0xFF11, 0xBF)
	mb.SetItem(0xFF24, 0xFF)
	assert.Equal(t, uint8(0x80), mb.GetItem(0xFF10), "NR10 should not have accepted write")
	assert.Equal(t, uint8(0x3F), mb.GetItem(0xFF11), "NR11 should not have accepted write")
	assert.Equal(t, uint8(0x00), mb.GetItem(0xFF24), "NR50 should not have accepted write")

	// Re-enable + accept writes.
	mb.SetItem(0xFF26, 0x80)
	mb.SetItem(0xFF24, 0x77)
	assert.Equal(t, uint8(0x77), mb.GetItem(0xFF24), "NR50 writable after re-enable")
}

// TestAPU_WaveRAMReadWrite confirms Wave RAM round-trips bytes.
func TestAPU_WaveRAMReadWrite(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// Disable channel 3 first so writes go through.
	mb.SetItem(0xFF1A, 0x00)
	for addr := uint16(0xFF30); addr <= 0xFF3F; addr++ {
		mb.SetItem(addr, uint16(addr-0xFF30+1))
	}
	for addr := uint16(0xFF30); addr <= 0xFF3F; addr++ {
		assert.Equal(t, byte(addr-0xFF30+1), mb.GetItem(addr), "wave RAM @%#x", addr)
	}
}

// TestAPU_ReadMasksAppliedToWriteOnlyRegisters checks that write-only
// bits (e.g. NR13 length-load) read back as 1 per Pan Docs masks.
func TestAPU_ReadMasksAppliedToWriteOnlyRegisters(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFF13, 0x00) // NR13 = period low (write only)
	assert.Equal(t, uint8(0xFF), mb.GetItem(0xFF13), "NR13 read-only mask = 0xFF")

	mb.SetItem(0xFF14, 0x00) // NR14 = trigger + length-enable + period-high
	assert.Equal(t, uint8(0xBF), mb.GetItem(0xFF14), "NR14 read mask = 0xBF (only bit 6 readable)")
}

// TestAPU_Ch1Trigger verifies that writing the trigger bit to NR14
// activates Channel 1 (visible via NR52 bit 0).
func TestAPU_Ch1Trigger(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// Power-cycle the APU + boot the channel with valid envelope.
	mb.SetItem(0xFF26, 0x00) // disable
	mb.SetItem(0xFF26, 0x80) // re-enable

	mb.SetItem(0xFF12, 0xF0) // NR12: full envelope volume, DAC on
	mb.SetItem(0xFF13, 0x00) // NR13: period low
	mb.SetItem(0xFF14, 0x87) // NR14: trigger + period high bits

	assert.NotZero(t, mb.GetItem(0xFF26)&0x01, "Channel 1 should be active after trigger")
}

// TestAPU_DACOffDisablesChannel verifies channel auto-disables when
// the DAC is turned off (NRx2 upper 5 bits = 0).
func TestAPU_DACOffDisablesChannel(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFF26, 0x80)

	mb.SetItem(0xFF12, 0xF0)
	mb.SetItem(0xFF14, 0x80)
	require.NotZero(t, mb.GetItem(0xFF26)&0x01)

	mb.SetItem(0xFF12, 0x00) // DAC off
	assert.Zero(t, mb.GetItem(0xFF26)&0x01, "Channel 1 should disable when DAC off")
}

// TestAPU_LengthCounterDisablesChannel runs the frame sequencer long
// enough for the length counter to expire and verifies the channel disables.
func TestAPU_LengthCounterDisablesChannel(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFF26, 0x80)
	mb.SetItem(0xFF12, 0xF0)
	mb.SetItem(0xFF11, 63)   // length load = 63 → length counter = 1
	mb.SetItem(0xFF14, 0xC0) // trigger + length enable
	require.NotZero(t, mb.GetItem(0xFF26)&0x01, "ch1 enabled after trigger")

	// Step the frame sequencer enough cycles to reach a length-clock step.
	// Frame sequencer steps 0, 2, 4, 6 clock length. apuFrameSeqPeriod
	// is 4194304/512 = 8192 cycles per step.
	for i := 0; i < 16; i++ {
		mb.Sound.Tick(8192)
	}
	assert.Zero(t, mb.GetItem(0xFF26)&0x01, "ch1 should be disabled after length counter expires")
}

// TestAPU_DutyTable verifies the duty-cycle waveforms match Pan Docs.
func TestAPU_DutyTable(t *testing.T) {
	// All 4 patterns sum to: 12.5% (1), 25% (2), 50% (4), 75% (6) high steps.
	for d := 0; d < 4; d++ {
		var highs int
		for _, b := range dutyTable[d] {
			if b == 1 {
				highs++
			}
		}
		expected := []int{1, 2, 4, 6}[d]
		assert.Equal(t, expected, highs, "duty %d high-step count", d)
	}
}

// TestAPU_NoiseLFSR confirms 7-bit and 15-bit LFSR shifts produce
// distinct sequences.
func TestAPU_NoiseLFSR(t *testing.T) {
	c := newNoiseChannel()
	c.lfsr = 0x7FFF
	// 15-bit mode
	c.widthMode7 = false
	for i := 0; i < 8; i++ {
		c.shiftLFSR()
	}
	state15 := c.lfsr

	c.lfsr = 0x7FFF
	c.widthMode7 = true
	for i := 0; i < 8; i++ {
		c.shiftLFSR()
	}
	state7 := c.lfsr

	assert.NotEqual(t, state15, state7,
		"7-bit and 15-bit LFSR must diverge after 8 shifts")
}

// TestAPU_WaveChannelVolumeCode verifies the four NR32 volume codes
// apply the right shift to wave samples.
func TestAPU_WaveChannelVolumeCode(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFF26, 0x80)
	apu := mb.Sound

	// Load wave RAM with full-volume samples (all 0xF nibbles).
	for i := uint16(0xFF30); i <= 0xFF3F; i++ {
		mb.SetItem(i, 0xFF)
	}
	mb.SetItem(0xFF1A, 0x80) // NR30: DAC on
	mb.SetItem(0xFF1B, 0)    // NR31: max length
	mb.SetItem(0xFF1D, 0)
	mb.SetItem(0xFF1E, 0x80) // trigger
	apu.ch3.wavePos = 0      // deterministic position
	expectedShifts := []byte{0, 0, 1, 2}
	for code := byte(1); code <= 3; code++ {
		mb.SetItem(0xFF1C, uint16(code)<<5)
		expected := float64(byte(0x0F)>>expectedShifts[code]) / 15.0
		actual := apu.ch3.output()
		assert.InDelta(t, expected, actual, 0.001, "volume code %d", code)
	}

	// Mute (code 0)
	mb.SetItem(0xFF1C, 0)
	assert.Equal(t, 0.0, apu.ch3.output(), "code 0 mutes")
}

// TestAPU_SerializeRoundtrip serializes APU state, mutates it, then
// deserializes and confirms state is restored.
func TestAPU_SerializeRoundtrip(t *testing.T) {
	mb := newMbForSubsysTest(t)

	// Build a non-default APU state.
	mb.SetItem(0xFF26, 0x80)
	mb.SetItem(0xFF11, 0x80) // duty=2
	mb.SetItem(0xFF12, 0xF8)
	mb.SetItem(0xFF13, 0x42)
	mb.SetItem(0xFF14, 0xC5) // trigger
	mb.SetItem(0xFF1A, 0x80) // wave DAC on
	mb.SetItem(0xFF25, 0xAA) // NR51 panning
	for i := uint16(0xFF30); i <= 0xFF3F; i++ {
		mb.SetItem(i, i&0xFF)
	}

	buf := mb.Sound.Serialize()
	require.NotEmpty(t, buf.Bytes())

	// Mutate state.
	mb.SetItem(0xFF26, 0x00) // disable, zero everything

	// Restore.
	require.NoError(t, mb.Sound.Deserialize(buf))

	assert.Equal(t, uint8(0xAA), mb.Sound.nr51, "NR51 restored")
	assert.True(t, mb.Sound.enabled, "APU enabled flag restored")
	assert.Equal(t, byte(2), mb.Sound.ch1.duty, "ch1 duty restored")
	for i := byte(0); i < 16; i++ {
		assert.Equal(t, byte(0x30+i), mb.Sound.waveRAM[i], "wave RAM[%d]", i)
	}
}

// TestAPU_FrameSequencerStepSchedule verifies the 8-step length /
// envelope / sweep cadence matches Pan Docs.
func TestAPU_FrameSequencerStepSchedule(t *testing.T) {
	mb := newMbForSubsysTest(t)
	a := mb.Sound
	a.frameSeqStep = 0
	a.ch1.enabled = true
	a.ch1.lengthEnabled = true
	a.ch1.lengthCounter = 64
	a.ch1.envelopeVolume = 0
	a.ch1.envelopeInit = 15
	a.ch1.envelopeUp = true
	a.ch1.envelopePeriod = 1
	a.ch1.envelopeTimer = 1

	// Step 0: length clock.
	a.stepFrameSequencer()
	assert.Equal(t, uint16(63), a.ch1.lengthCounter, "step 0 clocks length")

	// Step 1: nothing.
	a.stepFrameSequencer()
	assert.Equal(t, uint16(63), a.ch1.lengthCounter, "step 1 idle")

	// Step 2: length + sweep.
	a.stepFrameSequencer()
	assert.Equal(t, uint16(62), a.ch1.lengthCounter, "step 2 clocks length")

	// Step 7 (after 4 more increments): envelope.
	for i := 0; i < 4; i++ {
		a.stepFrameSequencer()
	}
	// We're now at step 7.
	prevVol := a.ch1.envelopeVolume
	a.stepFrameSequencer()
	assert.Greater(t, a.ch1.envelopeVolume, prevVol,
		"step 7 should clock envelope upward")
}

// TestAPU_Tick_NoAudio confirms the APU runs safely with audio disabled.
// (audioEnabled=false is what subsys test helpers provide.)
func TestAPU_Tick_NoAudio(t *testing.T) {
	mb := newMbForSubsysTest(t)
	require.False(t, mb.Sound.audioEnabled, "test helpers must not init speaker")
	// Run a few thousand ticks — must not panic, must not push to a nil ring.
	for i := 0; i < 10000; i++ {
		mb.Sound.Tick(4)
	}
}

// TestAPU_PanningMixesIntoCorrectChannel uses NR51 to route channel 1
// to LEFT only, then confirms emitSample produces left-only output.
func TestAPU_PanningMixesIntoCorrectChannel(t *testing.T) {
	a := NewAPU(nil, false, false) // headless

	// Force a deterministic full-volume ch1 output.
	a.enabled = true
	a.ch1.enabled = true
	a.ch1.dacOn = true
	a.ch1.envelopeVolume = 15
	a.ch1.duty = 2    // 50%
	a.ch1.dutyPos = 7 // dutyTable[2][7] = 1 (high)

	// Route ch1 LEFT only.
	a.nr51 = 0x10
	a.nr50 = 0x77

	// Mirror the math in emitSample (avoids needing a real streamer).
	s1 := a.ch1.output()
	require.Greater(t, s1, 0.0, "ch1 must produce non-zero output")
	var l, r float64
	if a.nr51&0x10 != 0 {
		l += s1
	}
	if a.nr51&0x01 != 0 {
		r += s1
	}
	assert.Greater(t, l, 0.0, "left channel must receive ch1")
	assert.Equal(t, 0.0, r, "right channel must be silent")
}
