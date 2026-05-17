// Package motherboard — apu_noise.go
//
// Noise channel (channel 4). LFSR-based pseudo-random output with
// envelope and length counter.
//
// References:
//   - Pan Docs §Audio Channel 4 (Noise)

package motherboard

import (
	"bytes"
	"encoding/binary"
)

// Divisor table for NR43 lower 3 bits (Pan Docs).
var noiseDivisorTable = [8]int{8, 16, 32, 48, 64, 80, 96, 112}

type noiseChannel struct {
	enabled bool
	dacOn   bool

	nr41, nr42, nr43, nr44 byte

	lengthLoad    byte
	lengthCounter uint16
	lengthEnabled bool

	envelopeVolume byte
	envelopeInit   byte
	envelopeUp     bool
	envelopePeriod byte
	envelopeTimer  byte

	clockShift  byte // NR43 bits 4-7
	widthMode7  bool // NR43 bit 3 (true = 7-bit LFSR)
	divisorCode byte // NR43 bits 0-2

	periodTimer int
	lfsr        uint16 // 15-bit LFSR (only bit 0 is the output)
}

func newNoiseChannel() *noiseChannel {
	return &noiseChannel{lfsr: 0x7FFF}
}

// step advances the LFSR per the divisor × 2^clockShift formula.
func (c *noiseChannel) step(cycles int) {
	if !c.enabled || !c.dacOn {
		return
	}
	period := noiseDivisorTable[c.divisorCode] << c.clockShift
	c.periodTimer -= cycles
	for c.periodTimer <= 0 {
		c.periodTimer += period
		c.shiftLFSR()
	}
}

// shiftLFSR runs one LFSR step. New bit = bit0 XOR bit1 of current LFSR,
// shifted into bit 14 (and bit 6 in 7-bit mode).
func (c *noiseChannel) shiftLFSR() {
	xor := (c.lfsr & 1) ^ ((c.lfsr >> 1) & 1)
	c.lfsr >>= 1
	c.lfsr |= xor << 14
	if c.widthMode7 {
		c.lfsr &^= 1 << 6
		c.lfsr |= xor << 6
	}
}

// output: bit 0 of LFSR inverted, scaled by envelope volume.
func (c *noiseChannel) output() float64 {
	if !c.enabled || !c.dacOn {
		return 0
	}
	if c.lfsr&1 != 0 {
		return 0 // LFSR bit 0 = 1 → low output
	}
	return float64(c.envelopeVolume) / 15.0
}

func (c *noiseChannel) clockLength() {
	if !c.lengthEnabled || c.lengthCounter == 0 {
		return
	}
	c.lengthCounter--
	if c.lengthCounter == 0 {
		c.enabled = false
	}
}

func (c *noiseChannel) clockEnvelope() {
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

func (c *noiseChannel) trigger() {
	c.enabled = c.dacOn
	if c.lengthCounter == 0 {
		c.lengthCounter = 64
	}
	period := noiseDivisorTable[c.divisorCode] << c.clockShift
	c.periodTimer = period
	c.envelopeVolume = c.envelopeInit
	if c.envelopePeriod == 0 {
		c.envelopeTimer = 8
	} else {
		c.envelopeTimer = c.envelopePeriod
	}
	c.lfsr = 0x7FFF
}

func (c *noiseChannel) writeReg(n int, v byte, frameSeqStep uint8) {
	switch n {
	case 1:
		c.nr41 = v
		c.lengthLoad = v & 0x3F
		c.lengthCounter = uint16(64 - c.lengthLoad)
	case 2:
		c.nr42 = v
		c.envelopeInit = (v >> 4) & 0x0F
		c.envelopeUp = v&0x08 != 0
		c.envelopePeriod = v & 0x07
		c.dacOn = (v & 0xF8) != 0
		if !c.dacOn {
			c.enabled = false
		}
	case 3:
		c.nr43 = v
		c.clockShift = (v >> 4) & 0x0F
		c.widthMode7 = v&0x08 != 0
		c.divisorCode = v & 0x07
	case 4:
		c.writeNR44(v, frameSeqStep)
	}
}

// writeNR44 implements the NR44 (trigger / length-enable) write with
// the DMG length-clock-on-trigger quirk. Same semantics as the square
// channel (see writeNRx4). SameBoy Core/apu.c lines 2141-2157.
func (c *noiseChannel) writeNR44(v byte, frameSeqStep uint8) {
	c.nr44 = v

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

func (c *noiseChannel) readReg(n int) byte {
	switch n {
	case 1:
		return c.nr41
	case 2:
		return c.nr42
	case 3:
		return c.nr43
	case 4:
		return c.nr44
	}
	return 0
}

func (c *noiseChannel) reset() {
	*c = noiseChannel{lfsr: 0x7FFF}
}

func (c *noiseChannel) powerOff() {
	c.nr41, c.nr42, c.nr43, c.nr44 = 0, 0, 0, 0
	c.envelopeInit, c.envelopePeriod, c.envelopeTimer, c.envelopeVolume = 0, 0, 0, 0
	c.envelopeUp = false
	c.lengthEnabled = false
	c.dacOn = false
	c.enabled = false
	c.clockShift, c.divisorCode = 0, 0
	c.widthMode7 = false
	c.periodTimer = 0
	c.lfsr = 0x7FFF
	c.lengthCounter = 0
	c.lengthLoad = 0
}

func (c *noiseChannel) serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, c.enabled)
	binary.Write(buf, binary.LittleEndian, c.dacOn)
	binary.Write(buf, binary.LittleEndian, c.nr41)
	binary.Write(buf, binary.LittleEndian, c.nr42)
	binary.Write(buf, binary.LittleEndian, c.nr43)
	binary.Write(buf, binary.LittleEndian, c.nr44)
	binary.Write(buf, binary.LittleEndian, c.lengthLoad)
	binary.Write(buf, binary.LittleEndian, c.lengthCounter)
	binary.Write(buf, binary.LittleEndian, c.lengthEnabled)
	binary.Write(buf, binary.LittleEndian, c.envelopeVolume)
	binary.Write(buf, binary.LittleEndian, c.envelopeInit)
	binary.Write(buf, binary.LittleEndian, c.envelopeUp)
	binary.Write(buf, binary.LittleEndian, c.envelopePeriod)
	binary.Write(buf, binary.LittleEndian, c.envelopeTimer)
	binary.Write(buf, binary.LittleEndian, c.clockShift)
	binary.Write(buf, binary.LittleEndian, c.widthMode7)
	binary.Write(buf, binary.LittleEndian, c.divisorCode)
	binary.Write(buf, binary.LittleEndian, int32(c.periodTimer))
	binary.Write(buf, binary.LittleEndian, c.lfsr)
	return buf
}

func (c *noiseChannel) deserialize(data *bytes.Buffer) error {
	fields := []any{
		&c.enabled, &c.dacOn,
		&c.nr41, &c.nr42, &c.nr43, &c.nr44,
		&c.lengthLoad, &c.lengthCounter, &c.lengthEnabled,
		&c.envelopeVolume, &c.envelopeInit, &c.envelopeUp, &c.envelopePeriod, &c.envelopeTimer,
		&c.clockShift, &c.widthMode7, &c.divisorCode,
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
	return binary.Read(data, binary.LittleEndian, &c.lfsr)
}
