// Package motherboard — apu_wave.go
//
// Wave channel (channel 3): 32 user-defined 4-bit samples in
// 0xFF30-0xFF3F. Volume code right-shifts the sample.
//
// References:
//   - Pan Docs §Audio Channel 3 (Wave Output)

package motherboard

import (
	"bytes"
	"encoding/binary"
)

type waveChannel struct {
	apu *APU

	enabled bool
	dacOn   bool // NR30 bit 7

	nr30, nr31, nr32, nr33, nr34 byte

	lengthLoad    byte
	lengthCounter uint16
	lengthEnabled bool

	volumeCode byte // NR32 bits 5,6: 0=mute, 1=100%, 2=50%, 3=25%
	frequency  uint16

	periodTimer int
	wavePos     int // 0..31 (4-bit sample index into Wave RAM)

	// DMG wave-RAM-bus quirk (Blargg dmg_sound test 09 / 12).
	// True only when the most recent APU.Tick chunk ended exactly on
	// a sample-fetch cycle. CPU reads of $FF30-$FF3F while ch3 is
	// active return 0xFF unless this flag is set (matches SameBoy
	// Core/apu.c lines 984/995/1002 + 1129 — wave_form_just_read).
	waveFormJustRead bool
}

func newWaveChannel(apu *APU) *waveChannel {
	return &waveChannel{apu: apu}
}

// step advances the wave-channel period. Note: wave runs at twice the
// rate of the square channels (Pan Docs: period = (2048-freq)*2).
//
// Also maintains waveFormJustRead: true iff this chunk ended exactly
// on a fetch boundary (an advance fired on the chunk's last cycle and
// no further cycles followed). See SameBoy Core/apu.c lines 984-1003.
func (c *waveChannel) step(cycles int) {
	c.waveFormJustRead = false
	if !c.enabled || !c.dacOn {
		return
	}
	period := (2048 - int(c.frequency)) * 2
	c.periodTimer -= cycles
	for c.periodTimer <= 0 {
		c.periodTimer += period
		c.wavePos = (c.wavePos + 1) & 0x1F
		c.waveFormJustRead = true
	}
	if c.periodTimer != period {
		c.waveFormJustRead = false
	}
}

// currentSampleByte returns the byte (= 2 samples) currently being read
// by the channel. Used when CPU reads Wave RAM while channel is active.
func (c *waveChannel) currentSampleByte() byte {
	return c.apu.readWaveRAMByte(c.wavePos / 2)
}

// output returns the current sample in [0, 1] after applying volume code.
func (c *waveChannel) output() float64 {
	if !c.enabled || !c.dacOn {
		return 0
	}
	b := c.apu.readWaveRAMByte(c.wavePos / 2)
	var sample byte
	if c.wavePos&1 == 0 {
		sample = (b >> 4) & 0x0F
	} else {
		sample = b & 0x0F
	}
	var shift byte
	switch c.volumeCode {
	case 0:
		return 0 // muted
	case 1:
		shift = 0 // 100%
	case 2:
		shift = 1 // 50%
	case 3:
		shift = 2 // 25%
	}
	return float64(sample>>shift) / 15.0
}

func (c *waveChannel) clockLength() {
	if !c.lengthEnabled || c.lengthCounter == 0 {
		return
	}
	c.lengthCounter--
	if c.lengthCounter == 0 {
		c.enabled = false
	}
}

func (c *waveChannel) trigger() {
	wasActive := c.enabled

	// DMG-only wave-RAM-corruption-on-retrigger (Blargg dmg_sound test 10);
	// CGB hardware does not exhibit this bug (cgb_sound test 10 verifies
	// the wave RAM stays intact). See SameBoy Core/apu.c lines 1978-2003
	// — the corruption is gated on `!GB_is_cgb(gb)`.
	isDMG := c.apu.Mb == nil || !c.apu.Mb.Cgb
	if isDMG && wasActive && c.dacOn && c.periodTimer == 2 {
		offset := byte((c.wavePos + 1) >> 1)
		offset &= 0x0F
		if offset < 4 {
			c.apu.waveRAM[0] = c.apu.waveRAM[offset]
		} else {
			base := offset &^ 3
			copy(c.apu.waveRAM[0:4], c.apu.waveRAM[base:base+4])
		}
	}

	c.enabled = c.dacOn
	if c.lengthCounter == 0 {
		c.lengthCounter = 256
	}
	c.wavePos = 0
	c.periodTimer = (2048-int(c.frequency))*2 + 6
}

func (c *waveChannel) writeReg(n int, v byte, frameSeqStep uint8) {
	switch n {
	case 0:
		c.nr30 = v
		c.dacOn = v&0x80 != 0
		if !c.dacOn {
			c.enabled = false
		}
	case 1:
		c.nr31 = v
		c.lengthLoad = v
		c.lengthCounter = uint16(256 - int(c.lengthLoad))
	case 2:
		c.nr32 = v
		c.volumeCode = (v >> 5) & 0x03
	case 3:
		c.nr33 = v
		c.frequency = (c.frequency & 0x0700) | uint16(v)
	case 4:
		c.writeNR34(v, frameSeqStep)
	}
}

// writeNR34 implements the NR34 (trigger / length-enable) write with
// the DMG length-clock-on-trigger quirk. Same semantics as the square
// channel (see writeNRx4) except: wave channel length max is 256, not
// 64; max-1 is 255. SameBoy Core/apu.c lines 2015-2038.
func (c *waveChannel) writeNR34(v byte, frameSeqStep uint8) {
	c.nr34 = v
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
				c.lengthCounter = 255
			} else {
				c.enabled = false
			}
		}
	}

	c.lengthEnabled = newLengthEnable
}

func (c *waveChannel) readReg(n int) byte {
	switch n {
	case 0:
		return c.nr30
	case 1:
		return c.nr31
	case 2:
		return c.nr32
	case 3:
		return c.nr33
	case 4:
		return c.nr34
	}
	return 0
}

func (c *waveChannel) reset() {
	apu := c.apu
	*c = waveChannel{apu: apu}
}

func (c *waveChannel) powerOff() {
	c.nr30, c.nr31, c.nr32, c.nr33, c.nr34 = 0, 0, 0, 0, 0
	c.volumeCode = 0
	c.frequency = 0
	c.lengthEnabled = false
	c.dacOn = false
	c.enabled = false
	c.periodTimer = 0
	c.wavePos = 0
	c.lengthCounter = 0
	c.lengthLoad = 0
}

func (c *waveChannel) serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, c.enabled)
	binary.Write(buf, binary.LittleEndian, c.dacOn)
	binary.Write(buf, binary.LittleEndian, c.nr30)
	binary.Write(buf, binary.LittleEndian, c.nr31)
	binary.Write(buf, binary.LittleEndian, c.nr32)
	binary.Write(buf, binary.LittleEndian, c.nr33)
	binary.Write(buf, binary.LittleEndian, c.nr34)
	binary.Write(buf, binary.LittleEndian, c.lengthLoad)
	binary.Write(buf, binary.LittleEndian, c.lengthCounter)
	binary.Write(buf, binary.LittleEndian, c.lengthEnabled)
	binary.Write(buf, binary.LittleEndian, c.volumeCode)
	binary.Write(buf, binary.LittleEndian, c.frequency)
	binary.Write(buf, binary.LittleEndian, int32(c.periodTimer))
	binary.Write(buf, binary.LittleEndian, int32(c.wavePos))
	return buf
}

func (c *waveChannel) deserialize(data *bytes.Buffer) error {
	fields := []any{
		&c.enabled, &c.dacOn,
		&c.nr30, &c.nr31, &c.nr32, &c.nr33, &c.nr34,
		&c.lengthLoad, &c.lengthCounter, &c.lengthEnabled,
		&c.volumeCode, &c.frequency,
	}
	for _, f := range fields {
		if err := binary.Read(data, binary.LittleEndian, f); err != nil {
			return err
		}
	}
	var pt, wp int32
	if err := binary.Read(data, binary.LittleEndian, &pt); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &wp); err != nil {
		return err
	}
	c.periodTimer = int(pt)
	c.wavePos = int(wp)
	return nil
}
