package cartridge

import (
	"bytes"
	"testing"

	"github.com/duysqubix/gobc/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTC_NewRTC_ZeroValues(t *testing.T) {
	r := NewRTC()
	require.NotNil(t, r)
	assert.Equal(t, uint8(0), r.s)
	assert.Equal(t, uint8(0), r.m)
	assert.Equal(t, uint8(0), r.h)
	assert.Equal(t, uint8(0), r.dl)
	assert.Equal(t, uint8(0), r.dh)
	assert.False(t, r.latchSet)
}

func TestRTC_Latch_CopiesInternalsToLatchedFields(t *testing.T) {
	r := NewRTC()
	r.s, r.m, r.h, r.dl, r.dh = 11, 22, 23, 254, 0x01
	r.Latch()
	assert.Equal(t, uint8(11), r.S)
	assert.Equal(t, uint8(22), r.M)
	assert.Equal(t, uint8(23), r.H)
	assert.Equal(t, uint8(254), r.DL)
	assert.Equal(t, uint8(0x01), r.DH)
	assert.True(t, r.latchSet)
}

func TestRTC_GetItem_WithoutLatchReturnsFF(t *testing.T) {
	r := NewRTC()
	r.s = 7
	assert.Equal(t, uint8(0xFF), r.GetItem(0x08),
		"GetItem without a prior Latch should return 0xFF")
}

func TestRTC_GetItem_WithLatchReturnsLatched(t *testing.T) {
	r := NewRTC()
	r.s, r.m, r.h, r.dl, r.dh = 1, 2, 3, 4, 5
	r.Latch()
	assert.Equal(t, uint8(1), r.GetItem(0x8))
	assert.Equal(t, uint8(2), r.GetItem(0x9))
	assert.Equal(t, uint8(3), r.GetItem(0xA))
	assert.Equal(t, uint8(4), r.GetItem(0xB))
	assert.Equal(t, uint8(5), r.GetItem(0xC))
}

func TestRTC_GetItem_UnknownRegisterReturnsFF(t *testing.T) {
	r := NewRTC()
	r.Latch()
	assert.Equal(t, uint8(0xFF), r.GetItem(0x07))
	assert.Equal(t, uint8(0xFF), r.GetItem(0x0D))
}

func TestRTC_SetItem_AppliesMasks(t *testing.T) {
	r := NewRTC()

	r.SetItem(0x8, 0xFF)
	assert.Equal(t, uint8(0x3F), r.s, "seconds register is 6 bits wide (MaskS=0x3F)")
	assert.Equal(t, uint8(0x3F), r.S)

	r.SetItem(0x9, 0xFF)
	assert.Equal(t, uint8(0x3F), r.m, "minutes register is 6 bits wide (MaskM=0x3F)")

	r.SetItem(0xA, 0xFF)
	assert.Equal(t, uint8(0x1F), r.h, "hours register is 5 bits wide (MaskH=0x1F)")

	r.SetItem(0xB, 0xAB)
	assert.Equal(t, uint8(0xAB), r.dl, "day-low register is 8 bits wide (MaskDL=0xFF)")

	r.SetItem(0xC, 0xFF)
	assert.Equal(t, uint8(0xC1), r.dh, "day-high register MaskDH is 0b11000001 = 0xC1")
}

func TestRTC_SetItem_SecondsResetsInternalCycle(t *testing.T) {
	r := NewRTC()
	r.internalCycleCounter = 12345
	r.SetItem(0x8, 0x00)
	assert.Equal(t, uint64(0), r.internalCycleCounter,
		"writing to seconds register should reset the internal cycle counter")
}

func TestRTC_Tick_AdvancesSecondsAtRTCCycles(t *testing.T) {
	r := NewRTC()
	r.Tick(RTCCycles)
	assert.Equal(t, uint8(1), r.s, "one second's worth of cycles should bump s by 1")
}

func TestRTC_Tick_NoopWhenHaltBitSet(t *testing.T) {
	r := NewRTC()
	internal.SetBit(&r.dh, TIMER_HALT_BIT)
	r.Tick(RTCCycles * 10)
	assert.Equal(t, uint8(0), r.s, "halt bit should freeze the clock")
}

func TestRTC_Tick_AccumulatesPartialCycles(t *testing.T) {
	r := NewRTC()
	r.Tick(RTCCycles / 2)
	assert.Equal(t, uint8(0), r.s, "half a second is not enough to bump seconds")
	r.Tick(RTCCycles / 2)
	assert.Equal(t, uint8(1), r.s, "the rest of the second should now bump seconds")
}

func TestRTC_InternalDayCounter_CombinesDhAndDl(t *testing.T) {
	r := NewRTC()
	r.dh = 0x01
	r.dl = 0x34
	assert.Equal(t, uint16(0x0134), r.internalDayCounter())
}

func TestRTC_IsDayCounterOverflow(t *testing.T) {
	r := NewRTC()
	r.dh = 0x01
	r.dl = 0xFF
	assert.False(t, r.isDayCounterOverflow(),
		"0x1FF is the inclusive upper bound, not yet overflow")
}

func TestRTC_Serialize_Deserialize_RoundTrip(t *testing.T) {
	src := NewRTC()
	src.s, src.m, src.h, src.dl, src.dh = 12, 34, 5, 200, 0x01
	src.S, src.M, src.H, src.DL, src.DH = 12, 34, 5, 200, 0x01
	src.latchSet = true
	src.internalCycleCounter = 999999

	buf := src.Serialize()
	dst := NewRTC()
	require.NoError(t, dst.Deserialize(bytes.NewBuffer(buf.Bytes())))

	assert.Equal(t, src.s, dst.s)
	assert.Equal(t, src.m, dst.m)
	assert.Equal(t, src.h, dst.h)
	assert.Equal(t, src.dl, dst.dl)
	assert.Equal(t, src.dh, dst.dh)
	assert.Equal(t, src.S, dst.S)
	assert.Equal(t, src.M, dst.M)
	assert.Equal(t, src.H, dst.H)
	assert.Equal(t, src.DL, dst.DL)
	assert.Equal(t, src.DH, dst.DH)
	assert.Equal(t, src.latchSet, dst.latchSet)
	assert.Equal(t, src.internalCycleCounter, dst.internalCycleCounter)
}
