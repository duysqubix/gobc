package cartridge

import (
	"bytes"
	"encoding/binary"

	"github.com/duysqubix/gobc/internal"
)

const (
	RTCCycles = 4194304 //- (32800) // CPU Ticks per second

	TIMER_HALT_BIT  = 6
	TIMER_CARRY_BIT = 7

	MaskS  = 0b00111111
	MaskM  = 0b00111111
	MaskH  = 0b00011111
	MaskDL = 0b11111111
	MaskDH = 0b11000001

	MAX_SECONDS = 60
	MAX_MINUTES = 60
	MAX_HOURS   = 24
	MAX_DL      = 255
	MAX_DH      = 1
)

const (
	SEC = iota
	MIN
	HOUR
	DAY
)

type RTC struct {
	internalCycleCounter uint64
	s                    uint8 // 6-bit seconds counter
	m                    uint8 // 6-bit minutes counter
	h                    uint8 // 5-bit hours counter
	dl                   uint8 // 8-bit lower 8-bits of day counter
	dh                   uint8 // 1-bit upper bit of day counter upper, bit 6 timer halt, bit 7 day counter carry

	// latched values, only copied to RTC registers when latch gate is set
	S        uint8
	M        uint8
	H        uint8
	DL       uint8
	DH       uint8
	latchSet bool
}

func NewRTC() *RTC {
	return &RTC{}
}

func (r *RTC) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, r.s)                    // 6-bit seconds counter
	binary.Write(buf, binary.LittleEndian, r.m)                    // 6-bit minutes counter
	binary.Write(buf, binary.LittleEndian, r.h)                    // 5-bit hours counter
	binary.Write(buf, binary.LittleEndian, r.dl)                   // 8-bit lower 8-bits of day counter
	binary.Write(buf, binary.LittleEndian, r.dh)                   // 1-bit upper bit of day counter upper, bit 6 timer halt, bit 7 day counter carry
	binary.Write(buf, binary.LittleEndian, r.S)                    // latched seconds
	binary.Write(buf, binary.LittleEndian, r.M)                    // latched minutes
	binary.Write(buf, binary.LittleEndian, r.H)                    // latched hours
	binary.Write(buf, binary.LittleEndian, r.DL)                   // latched lower 8-bits of day counter
	binary.Write(buf, binary.LittleEndian, r.DH)                   // latched upper bit of day counter upper, bit 6 timer halt, bit 7 day counter carry
	binary.Write(buf, binary.LittleEndian, r.latchSet)             // latch set
	binary.Write(buf, binary.LittleEndian, r.internalCycleCounter) // internal cycle counter
	logger.Debug("Serialized RTC state")
	return buf
}

func (r *RTC) Deserialize(data *bytes.Buffer) error {
	if err := binary.Read(data, binary.LittleEndian, &r.s); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.m); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.h); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.dl); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.dh); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.S); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.M); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.H); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.DL); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.DH); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.latchSet); err != nil {
		return err
	}

	if err := binary.Read(data, binary.LittleEndian, &r.internalCycleCounter); err != nil {
		return err
	}

	return nil
}

func (r *RTC) Latch() {
	r.S = r.s
	r.M = r.m
	r.H = r.h
	r.DL = r.dl
	r.DH = r.dh
	r.latchSet = true
}

func (r *RTC) GetItem(id uint16) uint8 {

	if r.latchSet {
		switch id {
		case 0x8:
			return r.S
		case 0x9:
			return r.M
		case 0xA:
			return r.H
		case 0xB:
			return r.DL
		case 0xC:
			return r.DH
		}
	}
	return 0xFF
}

func (r *RTC) SetItem(id uint16, value uint8) {
	// logger.Debugf("Setting RTC item %#x to %#x", id, value)

	switch id {
	case 0x8:
		r.s = value & MaskS
		r.S = value & MaskS
		// logger.Debugf("Resetting RTC")
		r.internalCycleCounter = 0 // reset internal cycle counter

	case 0x9:
		r.m = value & MaskM
		r.M = value & MaskM
	case 0xA:
		r.h = value & MaskH
		r.H = value & MaskH
	case 0xB:
		r.dl = value & MaskDL
		r.DL = value & MaskDL
	case 0xC:
		r.dh = value & MaskDH
		r.DH = value & MaskDH
	}
}

func (r *RTC) internalDayCounter() uint16 {
	return uint16(r.dh&0x1)<<8 | uint16(r.dl)
}

func (r *RTC) isDayCounterOverflow() bool {
	return r.internalDayCounter() > 0x1FF
}

func (r *RTC) Tick(cycles uint64) {
	if internal.IsBitSet(r.dh, TIMER_HALT_BIT) {
		return
	}

	r.internalCycleCounter += cycles

	if r.internalCycleCounter >= RTCCycles {

		var overflow uint8 = 0 // bit 0: seconds, bit 1: minutes, bit 2: hours, bit 3: day counter

		r.s++
		if r.s == MAX_SECONDS {
			r.s = 0
			internal.SetBit(&overflow, SEC)
		}

		if internal.IsBitSet(overflow, SEC) {
			r.m++
			if r.m == MAX_MINUTES {
				r.m = 0
				internal.SetBit(&overflow, MIN)
			}
		}

		if internal.IsBitSet(overflow, MIN) {
			r.h++
			if r.h == MAX_HOURS {
				r.h = 0
				internal.SetBit(&overflow, HOUR)
			}
		}

		if internal.IsBitSet(overflow, HOUR) {
			if r.dl < MAX_DL {
				r.dl++
			} else {
				if r.dl == MAX_DL && internal.IsBitSet(r.dh, 0) {
					r.dl = 0
					internal.ResetBit(&r.dh, 0)
					internal.SetBit(&r.dh, TIMER_CARRY_BIT)
				} else {
					r.dl++ // force overflow to 0
					r.dh |= 0x1
				}
			}
		}

		r.s &= MaskS
		r.m &= MaskM
		r.h &= MaskH

		// set day carry flag if day counter overflows
		if r.isDayCounterOverflow() {
			internal.SetBit(&r.dh, TIMER_CARRY_BIT)
			r.dl = 0
			r.dh = 0

		}
		r.internalCycleCounter -= RTCCycles
	}
}
