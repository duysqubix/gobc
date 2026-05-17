// Package motherboard — apu_square.go
//
// Square-wave channels (channels 1 and 2). Channel 1 adds a frequency
// sweep unit (NR10); channel 2 is identical otherwise.
//
// References:
//   - Pan Docs §Audio: NR10..NR14, NR21..NR24
//   - Humpheh/goboy pkg/apu/channel.go (envelope / sweep logic)

package motherboard

import (
	"bytes"
	"encoding/binary"
	"errors"
)

var errAPUUnknownVersion = errors.New("apu: unknown save-state version")

// 4 duty patterns × 8 phase steps. 1 = high, 0 = low (Pan Docs).
var dutyTable = [4][8]byte{
	{0, 0, 0, 0, 0, 0, 0, 1}, // 12.5%
	{1, 0, 0, 0, 0, 0, 0, 1}, // 25%
	{1, 0, 0, 0, 0, 1, 1, 1}, // 50%
	{0, 1, 1, 1, 1, 1, 1, 0}, // 75%
}

// squareChannel emulates a Game Boy square-wave channel.
//
// Channel 1 has hasSweep=true (uses NR10). Channel 2 has hasSweep=false
// and NR10-equivalent reads as 0xFF (handled by APU read mask, not here).
type squareChannel struct {
	enabled  bool
	dacOn    bool
	hasSweep bool

	// Raw register bytes (kept for read-back).
	nrx0, nrx1, nrx2, nrx3, nrx4 byte

	// Decoded state.
	duty           byte   // 0..3 (NRx1 bits 6-7)
	lengthLoad     byte   // NRx1 bits 0-5 (initial length value)
	lengthCounter  uint16 // 0..64; channel disabled when reaches 0 and length enabled
	envelopeVolume byte   // current envelope volume (0..15)
	envelopeInit   byte   // NRx2 bits 4-7
	envelopeUp     bool   // NRx2 bit 3
	envelopePeriod byte   // NRx2 bits 0-2
	envelopeTimer  byte   // counts down in 64 Hz steps
	frequency      uint16 // 11-bit period value (NRx3 + NRx4 bits 0-2)
	lengthEnabled  bool   // NRx4 bit 6

	// Sweep (channel 1 only).
	sweepShift   byte
	sweepDown    bool // NR10 bit 3
	sweepPeriod  byte
	sweepTimer   byte
	sweepEnabled bool
	sweepFreq    uint16
	sweepDidNeg  bool // tracks whether a negative-sweep calc ran since last trigger

	// Period timer / phase.
	periodTimer int   // counts down in CPU cycles
	dutyPos     uint8 // 0..7
}

func newSquareChannel(hasSweep bool) *squareChannel {
	return &squareChannel{hasSweep: hasSweep}
}

// step advances the channel by `cycles` CPU clocks, advancing the duty
// position whenever the period timer expires.
func (c *squareChannel) step(cycles int) {
	if !c.enabled || !c.dacOn {
		return
	}
	period := (2048 - int(c.frequency)) * 4 // CPU cycles per duty step
	c.periodTimer -= cycles
	for c.periodTimer <= 0 {
		c.periodTimer += period
		c.dutyPos = (c.dutyPos + 1) & 7
	}
}

// output returns the current DAC sample in [-1, +1] for this channel.
func (c *squareChannel) output() float64 {
	if !c.enabled || !c.dacOn {
		return 0
	}
	if dutyTable[c.duty][c.dutyPos] == 0 {
		return 0
	}
	// 0..15 envelope volume → 0..15 DAC input → -1..+1 output.
	return float64(c.envelopeVolume) / 15.0
}

// clockLength is called by the frame sequencer at 256 Hz (steps 0,2,4,6).
func (c *squareChannel) clockLength() {
	if !c.lengthEnabled || c.lengthCounter == 0 {
		return
	}
	c.lengthCounter--
	if c.lengthCounter == 0 {
		c.enabled = false
	}
}

// clockEnvelope is called by the frame sequencer at 64 Hz (step 7).
func (c *squareChannel) clockEnvelope() {
	if c.envelopePeriod == 0 {
		return
	}
	if c.envelopeTimer > 0 {
		c.envelopeTimer--
	}
	if c.envelopeTimer == 0 {
		c.envelopeTimer = c.envelopePeriod
		if c.envelopeUp && c.envelopeVolume < 15 {
			c.envelopeVolume++
		} else if !c.envelopeUp && c.envelopeVolume > 0 {
			c.envelopeVolume--
		}
	}
}

// clockSweep is called by the frame sequencer at 128 Hz (steps 2,6).
// Only meaningful for channel 1.
func (c *squareChannel) clockSweep() {
	if !c.hasSweep {
		return
	}
	if c.sweepTimer > 0 {
		c.sweepTimer--
	}
	if c.sweepTimer == 0 {
		if c.sweepPeriod > 0 {
			c.sweepTimer = c.sweepPeriod
		} else {
			c.sweepTimer = 8
		}
		if c.sweepEnabled && c.sweepPeriod > 0 {
			newFreq := c.calcSweep()
			if newFreq <= 2047 && c.sweepShift > 0 {
				c.sweepFreq = newFreq
				c.frequency = newFreq
				// Pan Docs: a second calculation runs to check for overflow,
				// but the result is discarded.
				c.calcSweep()
			}
		}
	}
}

// calcSweep computes the next sweep frequency and disables the channel
// on overflow. Used both by clockSweep and at trigger time.
func (c *squareChannel) calcSweep() uint16 {
	delta := c.sweepFreq >> c.sweepShift
	var newFreq uint16
	if c.sweepDown {
		c.sweepDidNeg = true
		if delta > c.sweepFreq {
			newFreq = 0
		} else {
			newFreq = c.sweepFreq - delta
		}
	} else {
		newFreq = c.sweepFreq + delta
	}
	if newFreq > 2047 {
		c.enabled = false
	}
	return newFreq
}

// trigger handles NRx4 bit 7 = 1. Pan Docs:
//   - if length=0, reload to max (64)
//   - reload period timer
//   - reload envelope
//   - channel-1: reload sweep
//   - re-enable if DAC on
func (c *squareChannel) trigger() {
	c.enabled = c.dacOn
	if c.lengthCounter == 0 {
		c.lengthCounter = 64
	}
	c.periodTimer = (2048 - int(c.frequency)) * 4
	c.envelopeVolume = c.envelopeInit
	if c.envelopePeriod == 0 {
		c.envelopeTimer = 8
	} else {
		c.envelopeTimer = c.envelopePeriod
	}
	if c.hasSweep {
		c.sweepFreq = c.frequency
		if c.sweepPeriod > 0 {
			c.sweepTimer = c.sweepPeriod
		} else {
			c.sweepTimer = 8
		}
		c.sweepEnabled = c.sweepPeriod != 0 || c.sweepShift != 0
		c.sweepDidNeg = false
		if c.sweepShift != 0 {
			c.calcSweep()
		}
	}
}

// writeReg applies a write to NRxN (n=0..4). The APU dispatches by
// register index inside the channel's 5-byte block. frameSeqStep is the
// current frame-sequencer step (0..7) used by the NRx4 length-clock-on-
// trigger quirk; ignored for n != 4.
func (c *squareChannel) writeReg(n int, v byte, frameSeqStep uint8) {
	switch n {
	case 0:
		c.nrx0 = v
		if c.hasSweep {
			c.sweepPeriod = (v >> 4) & 0x07
			oldDown := c.sweepDown
			c.sweepDown = v&0x08 != 0
			c.sweepShift = v & 0x07
			// Negative→positive transition after a neg sweep ran disables ch1.
			if oldDown && !c.sweepDown && c.sweepDidNeg {
				c.enabled = false
			}
		}
	case 1:
		c.nrx1 = v
		c.duty = (v >> 6) & 0x03
		c.lengthLoad = v & 0x3F
		c.lengthCounter = uint16(64 - c.lengthLoad)
	case 2:
		c.nrx2 = v
		c.envelopeInit = (v >> 4) & 0x0F
		c.envelopeUp = v&0x08 != 0
		c.envelopePeriod = v & 0x07
		c.dacOn = (v & 0xF8) != 0
		if !c.dacOn {
			c.enabled = false
		}
	case 3:
		c.nrx3 = v
		c.frequency = (c.frequency & 0x0700) | uint16(v)
	case 4:
		c.writeNRx4(v, frameSeqStep)
	}
}

// writeNRx4 implements the NRx4 (trigger / length-enable) write with
// the DMG "obscure behavior" quirk (Blargg dmg_sound test 03):
//
//  1. On trigger with length==0 reload to max 64 AND clear length_enabled
//     (so step 3 can re-check the transition).
//  2. If length-enable transitions 0→1 AND frame seq is in "first half"
//     (next FS step won't clock length, i.e. frameSeqStep ∈ {1,3,5,7})
//     AND lengthCounter > 0 → decrement lengthCounter. If it reaches 0
//     AND we're not also triggering → disable channel. If it reaches 0
//     AND we ARE triggering → reload to 63 (max-1).
//  3. Commit new length_enabled = (v & 0x40) != 0.
//
// Matches SameBoy Core/apu.c lines 1885-1934.
func (c *squareChannel) writeNRx4(v byte, frameSeqStep uint8) {
	c.nrx4 = v
	c.frequency = (c.frequency & 0x00FF) | (uint16(v&0x07) << 8)

	trigger := v&0x80 != 0
	newLengthEnable := v&0x40 != 0

	if trigger {
		wasZero := c.lengthCounter == 0
		c.trigger()
		if wasZero {
			c.lengthEnabled = false
		}
	}

	if newLengthEnable && !c.lengthEnabled && (frameSeqStep&1) == 1 && c.lengthCounter > 0 {
		c.lengthCounter--
		if c.lengthCounter == 0 {
			if trigger {
				c.lengthCounter = 63
			} else {
				c.enabled = false
			}
		}
	}

	c.lengthEnabled = newLengthEnable
}

// readReg returns the register byte. The APU applies the read mask.
func (c *squareChannel) readReg(n int) byte {
	switch n {
	case 0:
		return c.nrx0
	case 1:
		return c.nrx1
	case 2:
		return c.nrx2
	case 3:
		return c.nrx3
	case 4:
		return c.nrx4
	}
	return 0
}

func (c *squareChannel) reset() {
	*c = squareChannel{hasSweep: c.hasSweep}
}

func (c *squareChannel) powerOff() {
	// DMG quirk: lengthCounter and lengthLoad are PRESERVED across APU
	// power-off (Blargg dmg_sound tests 08, 11). SameBoy implements this
	// by snapshotting pulse_length, wiping the APU, then restoring
	// (Core/apu.c lines 1719-1743). We achieve the same effect by simply
	// not zeroing those two fields.
	preservedLengthCounter := c.lengthCounter
	preservedLengthLoad := c.lengthLoad

	c.nrx0, c.nrx1, c.nrx2, c.nrx3, c.nrx4 = 0, 0, 0, 0, 0
	c.duty = 0
	c.envelopeInit, c.envelopePeriod, c.envelopeTimer, c.envelopeVolume = 0, 0, 0, 0
	c.envelopeUp = false
	c.frequency = 0
	c.lengthEnabled = false
	c.dacOn = false
	c.enabled = false
	c.sweepShift, c.sweepPeriod, c.sweepTimer = 0, 0, 0
	c.sweepDown, c.sweepEnabled, c.sweepDidNeg = false, false, false
	c.sweepFreq = 0
	c.periodTimer = 0
	c.dutyPos = 0

	c.lengthCounter = preservedLengthCounter
	c.lengthLoad = preservedLengthLoad
}

func (c *squareChannel) serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, c.enabled)
	binary.Write(buf, binary.LittleEndian, c.dacOn)
	binary.Write(buf, binary.LittleEndian, c.nrx0)
	binary.Write(buf, binary.LittleEndian, c.nrx1)
	binary.Write(buf, binary.LittleEndian, c.nrx2)
	binary.Write(buf, binary.LittleEndian, c.nrx3)
	binary.Write(buf, binary.LittleEndian, c.nrx4)
	binary.Write(buf, binary.LittleEndian, c.duty)
	binary.Write(buf, binary.LittleEndian, c.lengthLoad)
	binary.Write(buf, binary.LittleEndian, c.lengthCounter)
	binary.Write(buf, binary.LittleEndian, c.envelopeVolume)
	binary.Write(buf, binary.LittleEndian, c.envelopeInit)
	binary.Write(buf, binary.LittleEndian, c.envelopeUp)
	binary.Write(buf, binary.LittleEndian, c.envelopePeriod)
	binary.Write(buf, binary.LittleEndian, c.envelopeTimer)
	binary.Write(buf, binary.LittleEndian, c.frequency)
	binary.Write(buf, binary.LittleEndian, c.lengthEnabled)
	binary.Write(buf, binary.LittleEndian, c.sweepShift)
	binary.Write(buf, binary.LittleEndian, c.sweepDown)
	binary.Write(buf, binary.LittleEndian, c.sweepPeriod)
	binary.Write(buf, binary.LittleEndian, c.sweepTimer)
	binary.Write(buf, binary.LittleEndian, c.sweepEnabled)
	binary.Write(buf, binary.LittleEndian, c.sweepFreq)
	binary.Write(buf, binary.LittleEndian, c.sweepDidNeg)
	binary.Write(buf, binary.LittleEndian, int32(c.periodTimer))
	binary.Write(buf, binary.LittleEndian, c.dutyPos)
	return buf
}

func (c *squareChannel) deserialize(data *bytes.Buffer) error {
	fields := []any{
		&c.enabled, &c.dacOn,
		&c.nrx0, &c.nrx1, &c.nrx2, &c.nrx3, &c.nrx4,
		&c.duty, &c.lengthLoad, &c.lengthCounter,
		&c.envelopeVolume, &c.envelopeInit, &c.envelopeUp, &c.envelopePeriod, &c.envelopeTimer,
		&c.frequency, &c.lengthEnabled,
		&c.sweepShift, &c.sweepDown, &c.sweepPeriod, &c.sweepTimer,
		&c.sweepEnabled, &c.sweepFreq, &c.sweepDidNeg,
	}
	for _, f := range fields {
		if err := binary.Read(data, binary.LittleEndian, f); err != nil {
			return err
		}
	}
	var pt int32
	if err := binary.Read(data, binary.LittleEndian, &pt); err != nil {
		return err
	}
	c.periodTimer = int(pt)
	return binary.Read(data, binary.LittleEndian, &c.dutyPos)
}
