package cartridge

import (
	"testing"

	"github.com/duysqubix/gobc/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mbcMakeBankedROM(numBanks int) [][]uint8 {
	rom := make([][]uint8, numBanks)
	for i := 0; i < numBanks; i++ {
		bank := make([]uint8, MEMORY_BANK_SIZE)
		fill := uint8(i)
		for j := range bank {
			bank[j] = fill
		}
		rom[i] = bank
	}
	return rom
}

func mbcNewTestCart(romBanks, ramBanks int) *Cartridge {
	return &Cartridge{
		RomBanks:      mbcMakeBankedROM(romBanks),
		RomBanksCount: uint16(romBanks),
		RamBankCount:  uint16(ramBanks),
		MemoryModel:   0,
	}
}

func mbcResetGrtc() { *Grtc = RTC{} }

func TestROMOnly_GetItem(t *testing.T) {
	cart := mbcNewTestCart(2, 0)
	rom := &RomOnlyCartridge{parent: cart}

	t.Run("bank0_read", func(t *testing.T) {
		assert.Equal(t, uint8(0), rom.GetItem(0x0000))
		assert.Equal(t, uint8(0), rom.GetItem(0x1234))
		assert.Equal(t, uint8(0), rom.GetItem(0x3FFF))
	})

	t.Run("bank1_read", func(t *testing.T) {
		assert.Equal(t, uint8(1), rom.GetItem(0x4000))
		assert.Equal(t, uint8(1), rom.GetItem(0x5555))
		assert.Equal(t, uint8(1), rom.GetItem(0x7FFF))
	})

	t.Run("ram_returns_ff", func(t *testing.T) {
		for _, addr := range []uint16{0xA000, 0xABCD, 0xBFFF} {
			assert.Equal(t, uint8(0xFF), rom.GetItem(addr))
		}
	})

	t.Run("default_returns_zero", func(t *testing.T) {
		assert.Equal(t, uint8(0), rom.GetItem(0x8000))
		assert.Equal(t, uint8(0), rom.GetItem(0xC000))
		assert.Equal(t, uint8(0), rom.GetItem(0xFFFF))
	})
}

func TestROMOnly_SetItem_NoOp(t *testing.T) {
	cart := mbcNewTestCart(2, 0)
	rom := &RomOnlyCartridge{parent: cart}

	originalBank0 := append([]uint8(nil), cart.RomBanks[0]...)
	originalBank1 := append([]uint8(nil), cart.RomBanks[1]...)

	for _, addr := range []uint16{0x0000, 0x1FFF, 0x2000, 0x3FFF, 0x4000,
		0x7FFF, 0xA000, 0xBFFF, 0xC000, 0xFFFF} {
		rom.SetItem(addr, 0x42)
	}

	assert.Equal(t, originalBank0, cart.RomBanks[0])
	assert.Equal(t, originalBank1, cart.RomBanks[1])
}

func TestROMOnly_Init_NoOp(t *testing.T) {
	cart := mbcNewTestCart(2, 0)
	rom := &RomOnlyCartridge{parent: cart}
	assert.NotPanics(t, func() { rom.Init() })
}

func TestROMOnly_SerializeRoundtrip(t *testing.T) {
	cart := mbcNewTestCart(2, 0)
	rom := &RomOnlyCartridge{parent: cart}

	buf := rom.Serialize()
	require.NotNil(t, buf)
	require.NoError(t, rom.Deserialize(buf))
}

func mbcNewMBC1(t *testing.T, romBanks, ramBanks int) (*Cartridge, *Mbc1Cartridge) {
	t.Helper()
	cart := mbcNewTestCart(romBanks, ramBanks)
	mbc := &Mbc1Cartridge{parent: cart, romBankSelect: 1, mode: false}
	cart.CartType = mbc
	return cart, mbc
}

func TestMBC1_GetItem_FixedBank0(t *testing.T) {
	_, mbc := mbcNewMBC1(t, 8, 1)
	assert.Equal(t, uint8(0), mbc.GetItem(0x0000))
	assert.Equal(t, uint8(0), mbc.GetItem(0x2000))
	assert.Equal(t, uint8(0), mbc.GetItem(0x3FFF))
}

func TestMBC1_GetItem_SwitchableBank(t *testing.T) {
	cases := []struct {
		name     string
		writeVal uint8
		wantBank uint8
		assertAt uint16
	}{
		{"write_1_selects_1", 0x01, 0x01, 0x4000},
		{"write_2_selects_2", 0x02, 0x02, 0x5000},
		{"write_5_selects_5", 0x05, 0x05, 0x7FFF},
		{"write_0_becomes_1", 0x00, 0x01, 0x4000},
		{"upper_bits_masked_off", 0xE5, 0x05, 0x6000},
		{"write_0x1F_selects_31", 0x1F, 0x1F, 0x4000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, mbc := mbcNewMBC1(t, 32, 1)
			mbc.SetItem(0x2000, tc.writeVal)
			got := mbc.GetItem(tc.assertAt)
			assert.Equal(t, tc.wantBank, got)
		})
	}
}

// High bank bits live in the 0x4000-0x5FFF register, not the lower-byte
// register. Writing 1 to each selects bank (1<<5)|1 == 0x21.
func TestMBC1_UpperBankBits_Mode0(t *testing.T) {
	_, mbc := mbcNewMBC1(t, 64, 1)
	mbc.SetItem(0x2000, 0x01)
	mbc.SetItem(0x4000, 0x01)
	assert.Equal(t, uint8(0x21), mbc.GetItem(0x4000))
}

func TestMBC1_UpperBankBits_MaskedTo2Bits(t *testing.T) {
	_, mbc := mbcNewMBC1(t, 128, 1)
	mbc.SetItem(0x2000, 0x01)
	mbc.SetItem(0x4000, 0xFF)
	assert.Equal(t, uint8(0x61), mbc.GetItem(0x4000))
}

func TestMBC1_BankingMode_Register(t *testing.T) {
	cart, mbc := mbcNewMBC1(t, 64, 4)

	mbc.SetItem(0x6000, 0x01)
	assert.True(t, mbc.mode)
	assert.Equal(t, uint8(1), cart.MemoryModel)

	mbc.SetItem(0x7FFF, 0x00)
	assert.False(t, mbc.mode)
	assert.Equal(t, uint8(0), cart.MemoryModel)

	mbc.SetItem(0x6000, 0xFE)
	assert.False(t, mbc.mode)
	mbc.SetItem(0x6000, 0xFF)
	assert.True(t, mbc.mode)
}

func TestMBC1_Mode1_AffectsLowBankRead(t *testing.T) {
	_, mbc := mbcNewMBC1(t, 64, 1)
	mbc.SetItem(0x4000, 0x01)
	mbc.SetItem(0x6000, 0x01)
	assert.Equal(t, uint8(0x20), mbc.GetItem(0x0000))
}

func TestMBC1_RAMEnable(t *testing.T) {
	cart, mbc := mbcNewMBC1(t, 8, 1)

	assert.False(t, cart.RamBankEnabled)

	mbc.SetItem(0x0000, 0x0A)
	assert.True(t, cart.RamBankEnabled)

	mbc.SetItem(0x0000, 0x00)
	assert.False(t, cart.RamBankEnabled)

	mbc.SetItem(0x0000, 0x0A)
	assert.True(t, cart.RamBankEnabled)
	mbc.SetItem(0x1FFF, 0xFF)
	assert.False(t, cart.RamBankEnabled)

	mbc.SetItem(0x0000, 0x1A)
	assert.True(t, cart.RamBankEnabled)
}

func TestMBC1_RAM_DisabledReadsFF(t *testing.T) {
	_, mbc := mbcNewMBC1(t, 8, 1)
	for _, addr := range []uint16{0xA000, 0xABCD, 0xBFFF} {
		assert.Equal(t, uint8(0xFF), mbc.GetItem(addr))
	}
}

func TestMBC1_RAM_DisabledWritesDropped(t *testing.T) {
	cart, mbc := mbcNewMBC1(t, 8, 1)
	mbc.SetItem(0xA000, 0x42)
	assert.Equal(t, uint8(0x00), cart.RamBanks[0][0])
}

func TestMBC1_RAM_EnabledReadWrite(t *testing.T) {
	cart, mbc := mbcNewMBC1(t, 8, 4)

	mbc.SetItem(0x0000, 0x0A)
	mbc.SetItem(0xA000, 0xAB)
	mbc.SetItem(0xA001, 0xCD)
	assert.Equal(t, uint8(0xAB), mbc.GetItem(0xA000))
	assert.Equal(t, uint8(0xCD), mbc.GetItem(0xA001))
	assert.Equal(t, uint8(0xAB), cart.RamBanks[0][0])
}

func TestMBC1_RAM_Mode1_BankSwitch(t *testing.T) {
	cart, mbc := mbcNewMBC1(t, 8, 4)

	mbc.SetItem(0x0000, 0x0A)
	mbc.SetItem(0x6000, 0x01)
	mbc.SetItem(0x4000, 0x02)

	mbc.SetItem(0xA000, 0x77)
	assert.Equal(t, uint8(0x77), cart.RamBanks[2][0])

	mbc.SetItem(0x4000, 0x03)
	mbc.SetItem(0xA000, 0x88)
	assert.Equal(t, uint8(0x88), cart.RamBanks[3][0])
	assert.Equal(t, uint8(0x77), cart.RamBanks[2][0])

	mbc.SetItem(0x4000, 0x02)
	assert.Equal(t, uint8(0x77), mbc.GetItem(0xA000))
	mbc.SetItem(0x4000, 0x03)
	assert.Equal(t, uint8(0x88), mbc.GetItem(0xA000))
}

func TestMBC1_SerializeRoundtrip(t *testing.T) {
	_, mbc := mbcNewMBC1(t, 8, 1)
	mbc.SetItem(0x2000, 0x05)
	mbc.SetItem(0x4000, 0x02)
	mbc.SetItem(0x6000, 0x01)
	mbc.hasBattery = true

	buf := mbc.Serialize()
	other := &Mbc1Cartridge{parent: mbcNewTestCart(8, 1)}
	require.NoError(t, other.Deserialize(buf))

	assert.Equal(t, uint16(5), other.romBankSelect)
	assert.Equal(t, uint16(2), other.ramBankSelect)
	assert.True(t, other.mode)
	assert.True(t, other.hasBattery)
}

func TestMBC1_Init_NoBatteryIsNoOp(t *testing.T) {
	_, mbc := mbcNewMBC1(t, 8, 0)
	assert.NotPanics(t, func() { mbc.Init() })
}

func mbcNewMBC3(t *testing.T, romBanks, ramBanks int, hasRTC bool) (*Cartridge, *Mbc3Cartridge) {
	t.Helper()
	cart := mbcNewTestCart(romBanks, ramBanks)
	cart.RomBankSelected = 1
	mbc := &Mbc3Cartridge{parent: cart, hasRTC: hasRTC}
	cart.CartType = mbc
	if hasRTC {
		mbcResetGrtc()
	}
	return cart, mbc
}

func TestMBC3_GetItem_FixedBank0(t *testing.T) {
	_, mbc := mbcNewMBC3(t, 8, 1, false)
	assert.Equal(t, uint8(0), mbc.GetItem(0x0000))
	assert.Equal(t, uint8(0), mbc.GetItem(0x1234))
	assert.Equal(t, uint8(0), mbc.GetItem(0x3FFF))
}

func TestMBC3_GetItem_SwitchableBank(t *testing.T) {
	cases := []struct {
		name     string
		writeVal uint8
		wantBank uint8
	}{
		{"write_1_selects_1", 0x01, 0x01},
		{"write_5_selects_5", 0x05, 0x05},
		{"write_0_becomes_1", 0x00, 0x01},
		{"write_0x7F_selects_127", 0x7F, 0x7F},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cart, mbc := mbcNewMBC3(t, 128, 1, false)
			mbc.SetItem(0x2000, tc.writeVal)
			assert.Equal(t, uint16(tc.wantBank), cart.RomBankSelected)
			assert.Equal(t, tc.wantBank, mbc.GetItem(0x4000))
		})
	}
}

func TestMBC3_GetItem_BankWrapAround(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, false)
	mbc.SetItem(0x2000, 0x09)
	assert.Equal(t, uint16(9), cart.RomBankSelected)
	assert.Equal(t, uint8(1), mbc.GetItem(0x4000))
}

func TestMBC3_RAMEnable(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, false)
	assert.False(t, cart.RamBankEnabled)

	mbc.SetItem(0x0000, 0x0A)
	assert.True(t, cart.RamBankEnabled)
	mbc.SetItem(0x0000, 0x00)
	assert.False(t, cart.RamBankEnabled)
	mbc.SetItem(0x0000, 0x1A)
	assert.True(t, cart.RamBankEnabled)
	mbc.SetItem(0x0000, 0x42)
	assert.False(t, cart.RamBankEnabled)
}

func TestMBC3_RAM_DisabledReadsFF(t *testing.T) {
	_, mbc := mbcNewMBC3(t, 8, 1, false)
	for _, addr := range []uint16{0xA000, 0xABCD, 0xBFFF} {
		assert.Equal(t, uint8(0xFF), mbc.GetItem(addr))
	}
}

func TestMBC3_RAM_EnabledReadWrite(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 4, false)
	mbc.SetItem(0x0000, 0x0A)

	mbc.SetItem(0x4000, 0x02)
	mbc.SetItem(0xA000, 0xAB)
	mbc.SetItem(0xA001, 0xCD)
	assert.Equal(t, uint8(0xAB), cart.RamBanks[2][0])
	assert.Equal(t, uint8(0xAB), mbc.GetItem(0xA000))
	assert.Equal(t, uint8(0xCD), mbc.GetItem(0xA001))

	mbc.SetItem(0x4000, 0x03)
	mbc.SetItem(0xA000, 0x99)
	assert.Equal(t, uint8(0x99), cart.RamBanks[3][0])
	assert.Equal(t, uint8(0xAB), cart.RamBanks[2][0])
}

func TestMBC3_RAM_DisabledWritesDropped(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, false)
	mbc.SetItem(0xA000, 0x42)
	assert.Equal(t, uint8(0x00), cart.RamBanks[0][0])
}

func TestMBC3_RTC_RegisterSelect(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, true)
	for sel := uint16(0x08); sel <= 0x0C; sel++ {
		mbc.SetItem(0x4000, uint8(sel))
		assert.Equal(t, sel, cart.RamBankSelected)
	}
}

func TestMBC3_RTC_LatchSequence(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, true)

	Grtc.s, Grtc.m, Grtc.h = 30, 15, 5
	Grtc.dl, Grtc.dh = 0x40, 0x00

	mbc.SetItem(0x0000, 0x0A)
	cart.RamBankSelected = 0x08
	assert.Equal(t, uint8(0xFF), mbc.GetItem(0xA000))

	mbc.SetItem(0x6000, 0x00)
	mbc.SetItem(0x6000, 0x01)
	assert.True(t, Grtc.latchSet)
	assert.Equal(t, uint8(30), Grtc.S)
	assert.Equal(t, uint8(15), Grtc.M)
	assert.Equal(t, uint8(5), Grtc.H)
	assert.Equal(t, uint8(0x40), Grtc.DL)

	cart.RamBankSelected = 0x08
	assert.Equal(t, uint8(30), mbc.GetItem(0xA000))
	cart.RamBankSelected = 0x09
	assert.Equal(t, uint8(15), mbc.GetItem(0xA000))
	cart.RamBankSelected = 0x0A
	assert.Equal(t, uint8(5), mbc.GetItem(0xA000))
	cart.RamBankSelected = 0x0B
	assert.Equal(t, uint8(0x40), mbc.GetItem(0xA000))
}

func TestMBC3_RTC_LatchRequires_0Then1(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, true)
	mbc.SetItem(0x0000, 0x0A)
	cart.RamBankSelected = 0x08

	Grtc.s = 10
	mbc.SetItem(0x6000, 0x00)
	mbc.SetItem(0x6000, 0x01)
	assert.Equal(t, uint8(10), Grtc.S)

	Grtc.s = 20
	mbc.SetItem(0x6000, 0x01)
	assert.Equal(t, uint8(10), Grtc.S)

	mbc.SetItem(0x6000, 0x00)
	mbc.SetItem(0x6000, 0x01)
	assert.Equal(t, uint8(20), Grtc.S)
}

func TestMBC3_RTC_WriteRegisters(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, true)
	mbc.SetItem(0x0000, 0x0A)
	cart.RamBankSelected = 0x08
	mbc.SetItem(0xA000, 42)
	assert.Equal(t, uint8(42), Grtc.s)
	assert.Equal(t, uint8(42), Grtc.S)

	cart.RamBankSelected = 0x09
	mbc.SetItem(0xA000, 30)
	assert.Equal(t, uint8(30), Grtc.m)
}

func TestMBC3_NoRTC_LatchIsNoOp(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, false)
	mbc.SetItem(0x0000, 0x0A)
	cart.RamBankSelected = 0x08

	mbcResetGrtc()
	mbc.SetItem(0x6000, 0x00)
	mbc.SetItem(0x6000, 0x01)
	assert.False(t, Grtc.latchSet)
	assert.Equal(t, uint8(0xFF), mbc.GetItem(0xA000))
}

func TestMBC3_SerializeRoundtrip(t *testing.T) {
	_, mbc := mbcNewMBC3(t, 8, 1, true)
	mbcResetGrtc()
	mbc.hasBattery = true
	mbc.latchGate1 = true
	Grtc.S = 33

	buf := mbc.Serialize()
	other := &Mbc3Cartridge{parent: mbcNewTestCart(8, 1)}
	require.NoError(t, other.Deserialize(buf))
	assert.True(t, other.hasBattery)
	assert.True(t, other.hasRTC)
	assert.True(t, other.latchGate1)
	assert.Equal(t, uint8(33), Grtc.S)
}

func TestMBC3_Init_NoBatteryNoRTC(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 0, false)
	mbc.Init()
	assert.False(t, cart.RtcEnabled)
}

func TestMBC3_Init_RTCEnabled(t *testing.T) {
	cart, mbc := mbcNewMBC3(t, 8, 1, true)
	mbc.Init()
	assert.True(t, cart.RtcEnabled)
}

func mbcNewMBC5(t *testing.T, romBanks, ramBanks int) (*Cartridge, *Mbc5Cartridge) {
	t.Helper()
	cart := mbcNewTestCart(romBanks, ramBanks)
	mbc := &Mbc5Cartridge{parent: cart}
	mbc.Init()
	cart.CartType = mbc
	return cart, mbc
}

func TestMBC5_GetItem_FixedBank0(t *testing.T) {
	_, mbc := mbcNewMBC5(t, 16, 1)
	assert.Equal(t, uint8(0), mbc.GetItem(0x0000))
	assert.Equal(t, uint8(0), mbc.GetItem(0x3FFF))
}

func TestMBC5_GetItem_SwitchableBank(t *testing.T) {
	cases := []struct {
		name     string
		writeVal uint8
		wantBank uint8
	}{
		{"write_0_selects_0_no_quirk", 0x00, 0x00},
		{"write_1_selects_1", 0x01, 0x01},
		{"write_5_selects_5", 0x05, 0x05},
		{"write_0x0F_selects_15", 0x0F, 0x0F},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, mbc := mbcNewMBC5(t, 16, 1)
			mbc.SetItem(0x2000, tc.writeVal)
			assert.Equal(t, tc.wantBank, mbc.GetItem(0x4000))
		})
	}
}

func TestMBC5_ROMBankLow_8bit(t *testing.T) {
	_, mbc := mbcNewMBC5(t, 16, 1)
	mbc.SetItem(0x2000, 0xFF)
	assert.Equal(t, uint16(0xFF), mbc.GetRomBank())
}

func TestMBC5_ROMBankHigh_9thBit(t *testing.T) {
	_, mbc := mbcNewMBC5(t, 16, 1)
	mbc.SetItem(0x2000, 0x55)
	mbc.SetItem(0x3000, 0x01)
	assert.Equal(t, uint16(0x155), mbc.GetRomBank())

	mbc.SetItem(0x3000, 0xFE)
	assert.Equal(t, uint16(0x055), mbc.GetRomBank())
}

func TestMBC5_NoBankZeroQuirk_ReadsBank0(t *testing.T) {
	cart, mbc := mbcNewMBC5(t, 16, 1)
	mbc.SetItem(0x2000, 0x00)
	assert.Equal(t, uint8(0), mbc.GetItem(0x4000))
	assert.NotEqual(t, cart.RomBanks[1][0], mbc.GetItem(0x4000))
}

func TestMBC5_RAMBankSelect(t *testing.T) {
	cart, mbc := mbcNewMBC5(t, 16, 4)
	mbc.SetItem(0x4000, 0x05)
	assert.Equal(t, uint16(0x05), cart.RamBankSelected)
	mbc.SetItem(0x4000, 0xFA)
	assert.Equal(t, uint16(0x0A), cart.RamBankSelected)
}

func TestMBC5_RAMEnable(t *testing.T) {
	cart, mbc := mbcNewMBC5(t, 16, 1)
	assert.False(t, cart.RamBankEnabled)

	mbc.SetItem(0x0000, 0x0A)
	assert.True(t, cart.RamBankEnabled)
	mbc.SetItem(0x0000, 0x00)
	assert.False(t, cart.RamBankEnabled)
	mbc.SetItem(0x1FFF, 0x1A)
	assert.True(t, cart.RamBankEnabled)
	mbc.SetItem(0x1FFF, 0x09)
	assert.False(t, cart.RamBankEnabled)
}

func TestMBC5_RAM_DisabledReadsFF(t *testing.T) {
	_, mbc := mbcNewMBC5(t, 16, 1)
	assert.Equal(t, uint8(0xFF), mbc.GetItem(0xA000))
	assert.Equal(t, uint8(0xFF), mbc.GetItem(0xBFFF))
}

// RamBankCount=1 is the smallest non-degenerate setup that keeps the
// production indexing `RamBankSelected & RamBankCount` inside the array.
func TestMBC5_RAM_EnabledReadWrite(t *testing.T) {
	cart, mbc := mbcNewMBC5(t, 16, 1)
	mbc.SetItem(0x0000, 0x0A)
	mbc.SetItem(0x4000, 0x00)
	mbc.SetItem(0xA000, 0xAB)
	assert.Equal(t, uint8(0xAB), cart.RamBanks[0][0])
	assert.Equal(t, uint8(0xAB), mbc.GetItem(0xA000))
}

func TestMBC5_RAM_DisabledWritesDropped(t *testing.T) {
	cart, mbc := mbcNewMBC5(t, 16, 1)
	mbc.SetItem(0xA000, 0x42)
	assert.Equal(t, uint8(0x00), cart.RamBanks[0][0])
}

func TestMBC5_GetRomBank_Composition(t *testing.T) {
	_, mbc := mbcNewMBC5(t, 16, 1)
	mbc.romBankLow = 0x42
	mbc.romBankHi = 0x01
	assert.Equal(t, uint16(0x142), mbc.GetRomBank())
}

func TestMBC5_SerializeRoundtrip(t *testing.T) {
	_, mbc := mbcNewMBC5(t, 16, 1)
	mbc.hasBattery = true
	mbc.hasRumble = true
	mbc.romBankLow = 0x42
	mbc.romBankHi = 0x01

	buf := mbc.Serialize()
	other := &Mbc5Cartridge{parent: mbcNewTestCart(16, 1)}
	require.NoError(t, other.Deserialize(buf))
	assert.True(t, other.hasBattery)
	assert.True(t, other.hasRumble)
	assert.Equal(t, uint8(0x42), other.romBankLow)
	assert.Equal(t, uint8(0x01), other.romBankHi)
}

func TestRTC_NewIsZero(t *testing.T) {
	r := NewRTC()
	require.NotNil(t, r)
	assert.Equal(t, uint8(0), r.s)
	assert.False(t, r.latchSet)
}

func TestRTC_SetGetItem_RequiresLatch(t *testing.T) {
	r := NewRTC()
	r.SetItem(0x08, 42)
	assert.Equal(t, uint8(0xFF), r.GetItem(0x08))
	assert.Equal(t, uint8(0xFF), r.GetItem(0x09))
}

func TestRTC_SetItem_MasksAndStores(t *testing.T) {
	r := NewRTC()
	r.SetItem(0x08, 0xFF)
	assert.Equal(t, uint8(0x3F), r.s)
	assert.Equal(t, uint8(0x3F), r.S)

	r.SetItem(0x09, 0x7E)
	assert.Equal(t, uint8(0x3E), r.m)

	r.SetItem(0x0A, 0xFF)
	assert.Equal(t, uint8(0x1F), r.h)

	r.SetItem(0x0B, 0xAB)
	assert.Equal(t, uint8(0xAB), r.dl)

	r.SetItem(0x0C, 0xFF)
	assert.Equal(t, uint8(0xC1), r.dh)
}

func TestRTC_SetItem_ResetsCounter(t *testing.T) {
	r := NewRTC()
	r.internalCycleCounter = 1234
	r.SetItem(0x08, 10)
	assert.Equal(t, uint64(0), r.internalCycleCounter)
}

func TestRTC_Latch_CopiesLiveToLatched(t *testing.T) {
	r := NewRTC()
	r.s, r.m, r.h, r.dl, r.dh = 12, 34, 5, 0x67, 0x01
	r.Latch()
	assert.True(t, r.latchSet)
	assert.Equal(t, uint8(12), r.S)
	assert.Equal(t, uint8(34), r.M)
	assert.Equal(t, uint8(5), r.H)
	assert.Equal(t, uint8(0x67), r.DL)
	assert.Equal(t, uint8(0x01), r.DH)
}

func TestRTC_GetItem_AfterLatch(t *testing.T) {
	r := NewRTC()
	r.s, r.m, r.h, r.dl, r.dh = 1, 2, 3, 4, 5
	r.Latch()
	assert.Equal(t, uint8(1), r.GetItem(0x08))
	assert.Equal(t, uint8(2), r.GetItem(0x09))
	assert.Equal(t, uint8(3), r.GetItem(0x0A))
	assert.Equal(t, uint8(4), r.GetItem(0x0B))
	assert.Equal(t, uint8(5), r.GetItem(0x0C))
	assert.Equal(t, uint8(0xFF), r.GetItem(0x00))
}

func TestRTC_Tick_HaltBitPreventsAdvance(t *testing.T) {
	r := NewRTC()
	internal.SetBit(&r.dh, TIMER_HALT_BIT)
	r.Tick(RTCCycles * 2)
	assert.Equal(t, uint64(0), r.internalCycleCounter)
	assert.Equal(t, uint8(0), r.s)
}

func TestRTC_Tick_IncrementsSeconds(t *testing.T) {
	r := NewRTC()
	r.Tick(RTCCycles)
	assert.Equal(t, uint8(1), r.s)
	r.Tick(RTCCycles)
	assert.Equal(t, uint8(2), r.s)
}

func TestRTC_Tick_OverflowChainsToMinutes(t *testing.T) {
	r := NewRTC()
	r.s = MAX_SECONDS - 1
	r.Tick(RTCCycles)
	assert.Equal(t, uint8(0), r.s)
	assert.Equal(t, uint8(1), r.m)
}

func TestRTC_Tick_BelowThresholdAccumulates(t *testing.T) {
	r := NewRTC()
	r.Tick(RTCCycles - 1)
	assert.Equal(t, uint8(0), r.s)
	assert.Equal(t, uint64(RTCCycles-1), r.internalCycleCounter)
	r.Tick(1)
	assert.Equal(t, uint8(1), r.s)
}

func TestRTC_SerializeRoundtrip(t *testing.T) {
	r := NewRTC()
	r.s = 12
	r.m = 34
	r.h = 5
	r.dl = 0xAB
	r.dh = 0x01
	r.Latch()
	r.internalCycleCounter = 9999

	buf := r.Serialize()
	other := NewRTC()
	require.NoError(t, other.Deserialize(buf))
	assert.Equal(t, r.s, other.s)
	assert.Equal(t, r.m, other.m)
	assert.Equal(t, r.h, other.h)
	assert.Equal(t, r.dl, other.dl)
	assert.Equal(t, r.dh, other.dh)
	assert.Equal(t, r.S, other.S)
	assert.Equal(t, r.M, other.M)
	assert.Equal(t, r.H, other.H)
	assert.Equal(t, r.DL, other.DL)
	assert.Equal(t, r.DH, other.DH)
	assert.True(t, other.latchSet)
	assert.Equal(t, uint64(9999), other.internalCycleCounter)
}

var _ CartridgeType = (*RomOnlyCartridge)(nil)
var _ CartridgeType = (*Mbc1Cartridge)(nil)
var _ CartridgeType = (*Mbc3Cartridge)(nil)
var _ CartridgeType = (*Mbc5Cartridge)(nil)
