package cartridge

import "github.com/duysqubix/gobc/internal"

const (
	RTCCycles = 4194304 // CPU Ticks per second

	TIMER_HALT_BIT  = 6
	TIMER_CARRY_BIT = 7
)

type RTC struct {
	internalCycleCounter uint64
	S                    uint8 // 6-bit seconds counter
	M                    uint8 // 6-bit minutes counter
	H                    uint8 // 5-bit hours counter
	DL                   uint8 // 8-bit lower 8-bits of day counter
	DH                   uint8 // 1-bit upper bit of day counter upper, bit 6 timer halt, bit 7 day counter carry
	IsLatched            bool  // Latch flag
}

func NewRTC() *RTC {
	return &RTC{}
}

func (r *RTC) GetItem(id uint8) uint8 {

	if r.IsLatched {
		switch id {
		case 0x8:
			return r.S | 0xC0
		case 0x9:
			return r.M | 0xC0
		case 0xA:
			return r.H | 0xE0
		case 0xB:
			return r.DL
		case 0xC:
			return r.DH | 0x3E
		}
	}
	return 0xFF
}

func (r *RTC) SetItem(id uint8, value uint8) {
	if r.IsLatched {
		switch id {
		case 0x8:
			r.S = value & 0x3F
			r.internalCycleCounter = 0 // reset internal cycle counter

		case 0x9:
			r.M = value & 0x3F
		case 0xA:
			r.H = value & 0x1F
		case 0xB:
			r.DL = value & 0xFF
		case 0xC:
			r.DH = value & 0xC1
		}
	}
}

func (r *RTC) Tick(cycles uint64) {
	if internal.IsBitSet(r.DH, TIMER_HALT_BIT) {
		return
	}

	r.internalCycleCounter += cycles

	if r.internalCycleCounter >= RTCCycles {
		r.S++
		if r.S > 60 {
			r.S = 0
		} else if r.S == 60 {
			r.S = 0
			r.M++
			if r.M > 60 {
				r.M = 0
			} else if r.M == 60 {
				r.M = 0
				r.H++
				if r.H > 24 {
					r.H = 0
				} else if r.H == 24 {
					r.H = 0
					r.DL++
					if r.DL == 0 {
						r.DH |= 1 << 7
					}
				}
			}
		}

		// set day carry flag if day counter overflows
		if uint16(r.DH&0x1)<<8|uint16(r.DL) >= 0x200 {
			internal.SetBit(&r.DH, TIMER_CARRY_BIT)

		}
		r.internalCycleCounter %= RTCCycles
	}
}
