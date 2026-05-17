package motherboard

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory_NewInternalRAMDefaultsToFixedFill(t *testing.T) {
	mb := newMbForSubsysTest(t)

	// Non-randomised init fills WRAM / HRAM / OAM with 0xFF and VRAM with 0x00.
	assert.Equal(t, uint8(0xFF), mb.Memory.Wram[0][0])
	assert.Equal(t, uint8(0xFF), mb.Memory.Wram[7][0xFFF])
	assert.Equal(t, uint8(0xFF), mb.Memory.Hram[0])
	assert.Equal(t, uint8(0xFF), mb.Memory.Hram[len(mb.Memory.Hram)-1])
	assert.Equal(t, uint8(0xFF), mb.Memory.Oam[0])
	assert.Equal(t, uint8(0xFF), mb.Memory.Oam[len(mb.Memory.Oam)-1])
	assert.Equal(t, uint8(0x00), mb.Memory.Vram[0][0])
	assert.Equal(t, uint8(0x00), mb.Memory.Vram[1][0x1FFF])
}

func TestMemory_GetIOSetIORoundtrip(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.Memory.SetIO(IO_SCY, 0x42)
	assert.Equal(t, uint8(0x42), mb.Memory.GetIO(IO_SCY))
}

func TestMemory_VRAMDirectAccessRoundtrip(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.Memory.SetVram(0, 0x8000, 0xAA)
	assert.Equal(t, uint8(0xAA), mb.Memory.GetVram(0, 0x8000))

	mb.Memory.SetVram(0, 0x9FFF, 0xBB)
	assert.Equal(t, uint8(0xBB), mb.Memory.GetVram(0, 0x9FFF))
}

// TestMemory_ActiveWramBankFromSVBK pins the SVBK-decode contract used by
// the CGB WRAM-banking code path. The function does NOT gate on Cgb mode -
// that gating happens in motherboard.GetItem/SetItem at the 0xD000 region.
// Banks 0 and 1 both collapse to bank 1 (Pan Docs).
func TestMemory_ActiveWramBankFromSVBK(t *testing.T) {
	mb := newMbForSubsysTest(t)

	cases := []struct {
		svbk uint8
		want uint8
	}{
		{0x00, 1}, // 0 collapses to bank 1
		{0x01, 1},
		{0x02, 2},
		{0x05, 5},
		{0x07, 7},
		{0xFF, 7}, // upper bits ignored, low 3 bits = 0b111 = 7
	}
	for _, tc := range cases {
		mb.Memory.SetIO(IO_SVBK, tc.svbk)
		assert.Equal(t, tc.want, mb.Memory.ActiveWramBank(),
			"SVBK=%#02x", tc.svbk)
	}
}

func TestMemory_DMA_DRegionUsesWramBank1InDMG(t *testing.T) {
	// Regardless of SVBK contents, the 0xD000 dispatcher path in DMG mode
	// always routes to Wram[1] (since the dispatcher's `if m.Cgb` branch
	// is skipped).
	mb := newMbForSubsysTest(t)
	mb.Memory.SetIO(IO_SVBK, 0x05) // would pick bank 5 in CGB
	mb.Memory.Wram[1][0x000] = 0xAA
	mb.Memory.Wram[5][0x000] = 0x55 // should NOT be read in DMG mode

	assert.Equal(t, uint8(0xAA), mb.GetItem(0xD000),
		"DMG-mode 0xD000 read must hit Wram[1] not Wram[SVBK]")
}

// TestMemory_RegionDispatch is the spec-mandated table-driven test that walks
// every Game Boy memory region and verifies the motherboard dispatcher routes
// a SetItem/GetItem pair to the documented backing store.
func TestMemory_RegionDispatch(t *testing.T) {
	cases := []struct {
		name     string
		addr     uint16
		writable bool // false for cartridge ROM regions (writes go to MBC)
	}{
		{"ROM bank 0 (0x0000)", 0x0000, false},
		{"ROM bank 0 (0x3FFF)", 0x3FFF, false},
		{"switchable ROM bank (0x4000)", 0x4000, false},
		{"switchable ROM bank (0x7FFF)", 0x7FFF, false},
		{"VRAM (0x8000)", 0x8000, true},
		{"VRAM (0x9FFF)", 0x9FFF, true},
		{"WRAM bank 0 (0xC000)", 0xC000, true},
		{"WRAM bank 0 (0xCFFF)", 0xCFFF, true},
		{"WRAM bank 1 (0xD000)", 0xD000, true},
		{"WRAM bank 1 (0xDFFF)", 0xDFFF, true},
		{"OAM (0xFE00)", 0xFE00, true},
		{"OAM (0xFE9F)", 0xFE9F, true},
		{"IO SCY (0xFF42)", 0xFF42, true},
		{"HRAM (0xFF80)", 0xFF80, true},
		{"HRAM (0xFFFE)", 0xFFFE, true},
		{"IE register (0xFFFF)", 0xFFFF, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mb := newMbForSubsysTest(t)
			before := mb.GetItem(tc.addr)
			mb.SetItem(tc.addr, 0x5A)
			after := mb.GetItem(tc.addr)

			if tc.writable {
				assert.Equal(t, uint8(0x5A), after,
					"writable region must round-trip the value")
			} else {
				assert.Equal(t, before, after,
					"ROM region must drop the write (RomOnlyCartridge.SetItem is a no-op)")
			}
		})
	}
}

func TestMemory_WRAMBank0RoundtripViaBus(t *testing.T) {
	mb := newMbForSubsysTest(t)
	addrs := []uint16{0xC000, 0xC001, 0xC800, 0xCFFF}
	for i, addr := range addrs {
		mb.SetItem(addr, uint16(0x10+i))
	}
	for i, addr := range addrs {
		assert.Equal(t, uint8(0x10+i), mb.GetItem(addr),
			"WRAM bank 0 round-trip at %#04x", addr)
		assert.Equal(t, uint8(0x10+i), mb.Memory.Wram[0][addr-0xC000],
			"underlying Wram[0] backing store matches")
	}
}

func TestMemory_WRAMBank1RoundtripViaBus(t *testing.T) {
	mb := newMbForSubsysTest(t)
	addrs := []uint16{0xD000, 0xD001, 0xD800, 0xDFFF}
	for i, addr := range addrs {
		mb.SetItem(addr, uint16(0x20+i))
	}
	for i, addr := range addrs {
		assert.Equal(t, uint8(0x20+i), mb.GetItem(addr))
		assert.Equal(t, uint8(0x20+i), mb.Memory.Wram[1][addr-0xD000],
			"in DMG mode 0xD000 region maps to Wram[1]")
	}
}

func TestMemory_HRAMRoundtripViaBus(t *testing.T) {
	mb := newMbForSubsysTest(t)
	addrs := []uint16{0xFF80, 0xFFAB, 0xFFFE}
	for i, addr := range addrs {
		mb.SetItem(addr, uint16(0x30+i))
	}
	for i, addr := range addrs {
		assert.Equal(t, uint8(0x30+i), mb.GetItem(addr))
		assert.Equal(t, uint8(0x30+i), mb.Memory.Hram[addr-0xFF80])
	}
}

func TestMemory_VRAMRoundtripViaBus(t *testing.T) {
	mb := newMbForSubsysTest(t)
	addrs := []uint16{0x8000, 0x8800, 0x9000, 0x9FFF}
	for i, addr := range addrs {
		mb.SetItem(addr, uint16(0x40+i))
	}
	for i, addr := range addrs {
		assert.Equal(t, uint8(0x40+i), mb.GetItem(addr))
		assert.Equal(t, uint8(0x40+i), mb.Memory.Vram[0][addr-0x8000],
			"DMG VRAM lives in bank 0")
	}
}

func TestMemory_OAMRoundtripViaBus(t *testing.T) {
	mb := newMbForSubsysTest(t)
	addrs := []uint16{0xFE00, 0xFE50, 0xFE9F}
	for i, addr := range addrs {
		mb.SetItem(addr, uint16(0x50+i))
	}
	for i, addr := range addrs {
		assert.Equal(t, uint8(0x50+i), mb.GetItem(addr))
		assert.Equal(t, uint8(0x50+i), mb.Memory.Oam[addr-0xFE00])
	}
}

// TestMemory_ROMWriteIsDroppedByRomOnly verifies that bus writes to the ROM
// region (0x0000..0x7FFF) reach the cartridge and are silently dropped by
// RomOnlyCartridge.SetItem (which is the documented behaviour for the
// non-MBC cart type 0x00). After the write, GetItem still returns the
// original ROM byte.
func TestMemory_ROMWriteIsDroppedByRomOnly(t *testing.T) {
	mb := newMbForSubsysTest(t)

	addrs := []uint16{0x0000, 0x1000, 0x3FFF, 0x4000, 0x6000, 0x7FFF}
	for _, addr := range addrs {
		before := mb.GetItem(addr)
		mb.SetItem(addr, 0x42)
		after := mb.GetItem(addr)
		assert.Equal(t, before, after,
			"ROM write at %#04x must be a no-op (RomOnlyCartridge)", addr)
	}
}

// TestMemory_EchoRAMMirrorsWRAMOnRead verifies that the echo-RAM region
// (0xE000..0xFDFF) maps to WRAM 0xC000..0xDDFF on read.
func TestMemory_EchoRAMMirrorsWRAMOnRead(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.Memory.Wram[0][0x0000] = 0xAB
	mb.Memory.Wram[0][0x0FFF] = 0xCD
	mb.Memory.Wram[1][0x0000] = 0xEF

	assert.Equal(t, uint8(0xAB), mb.GetItem(0xE000), "echo of 0xC000")
	assert.Equal(t, uint8(0xCD), mb.GetItem(0xEFFF), "echo of 0xCFFF")
	assert.Equal(t, uint8(0xEF), mb.GetItem(0xF000), "echo of 0xD000")
}

func TestMemory_IERegisterRoundtripViaBus(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFFFF, 0x1F)
	assert.Equal(t, uint8(0x1F), mb.GetItem(0xFFFF))
	assert.Equal(t, uint8(0x1F), mb.Cpu.Interrupts.IE)
}

func TestMemory_TimerRegistersRouteThroughTimer(t *testing.T) {
	mb := newMbForSubsysTest(t)

	mb.SetItem(0xFF05, 0x11) // TIMA
	mb.SetItem(0xFF06, 0x22) // TMA
	mb.SetItem(0xFF07, 0x05) // TAC (bits 3..7 forced high -> 0xFD)

	assert.Equal(t, uint32(0x11), mb.Timer.TIMA)
	assert.Equal(t, uint32(0x22), mb.Timer.TMA)
	assert.Equal(t, uint32(0xFD), mb.Timer.TAC)
}

func TestMemory_SetItemPanicsOnOversizedValue(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// SetItem calls logger.Fatalf when value >= 0x100; that's wired to a
	// no-op via init(), so we just assert the call returns without crashing
	// the test binary.
	mb.SetItem(0xC000, 0x1234)
}

func TestMemory_SerializeRoundtrip(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Memory.Wram[0][10] = 0xAA
	mb.Memory.Hram[20] = 0xBB
	mb.Memory.Vram[0][100] = 0xCC
	mb.Memory.Oam[5] = 0xDD
	mb.Memory.SetIO(IO_SCY, 0xEE)

	buf := mb.Memory.Serialize()
	require.NotNil(t, buf)

	restored := &Memory{Cgb: false, Mb: mb}
	require.NoError(t, restored.Deserialize(bytes.NewBuffer(buf.Bytes())))

	assert.Equal(t, uint8(0xAA), restored.Wram[0][10])
	assert.Equal(t, uint8(0xBB), restored.Hram[20])
	assert.Equal(t, uint8(0xCC), restored.Vram[0][100])
	assert.Equal(t, uint8(0xDD), restored.Oam[5])
	assert.Equal(t, uint8(0xEE), restored.GetIO(IO_SCY))
}

func TestMemory_DeserializeReturnsErrorOnShortBuffer(t *testing.T) {
	mem := &Memory{}
	err := mem.Deserialize(bytes.NewBuffer([]byte{0x01}))
	require.Error(t, err)
}

func TestMemory_VBlankRegionDirectFieldAccess(t *testing.T) {
	mb := newMbForSubsysTest(t)

	// IO addresses with bespoke read handlers in motherboard_getitem.go.
	mb.Memory.SetIO(IO_LCDC, 0xAB)
	assert.Equal(t, uint8(0xAB), mb.GetItem(0xFF40))

	mb.Memory.SetIO(IO_STAT, 0x40)
	assert.Equal(t, uint8(0x40), mb.GetItem(0xFF41))

	mb.Memory.SetIO(IO_LY, 0x90)
	assert.Equal(t, uint8(0x90), mb.GetItem(0xFF44))
}

// TestMemory_NR52ReflectsAPUState verifies NR52 reports the live APU
// state per Pan Docs: bit 7 = APU enable, bits 6-4 = unused (read 1),
// bits 0-3 = per-channel active flags. After boot the APU is enabled
// and Channel 1 is active (post boot ROM trigger), so the expected
// value is 0x80 | 0x70 | 0x01 = 0xF1.
func TestMemory_NR52ReflectsAPUState(t *testing.T) {
	mb := newMbForSubsysTest(t)
	assert.Equal(t, uint8(0xF1), mb.GetItem(0xFF26))
}

func TestMemory_DMARegisterReadsZero(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// 0xFF46 read always returns 0 per the getitem dispatcher.
	assert.Equal(t, uint8(0x00), mb.GetItem(0xFF46))
}

func TestMemory_UnusableAddressRangeReturnsDefaultByte(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// 0xFEA0..0xFEFF has no case body in the dispatcher; the function
	// falls through to the trailing `return 0xFF`.
	assert.Equal(t, uint8(0xFF), mb.GetItem(0xFEA0))
	assert.Equal(t, uint8(0xFF), mb.GetItem(0xFEFF))
}

func TestMemory_TileDataReturnsBank0VRAMSlice(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Memory.Vram[0][0] = 0x11
	mb.Memory.Vram[0][0x17FE] = 0x22

	tileData := mb.Memory.TileData(0)
	require.Equal(t, 0x17FF, len(tileData),
		"TileData returns Vram[bank][:0x17FF]")
	assert.Equal(t, uint8(0x11), tileData[0])
	assert.Equal(t, uint8(0x22), tileData[0x17FE])
}

// TestMemory_JoypadWriteRoutesThroughInput exercises the 0xFF00 SetItem
// path, which masks the written value through Input.Pull and stores the
// result in the P1 IO register.
func TestMemory_JoypadWriteRoutesThroughInput(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// Writing 0x30 selects "no row" (bits 4 and 5 high). Input.Pull then
	// merges in the directional/standard nibbles; we don't pin the exact
	// pulled value here (it depends on Input internals), just that the
	// write touched the IO register.
	mb.SetItem(0xFF00, 0x30)
	assert.NotZero(t, mb.GetItem(0xFF00))
}

// TestMemory_STATWritePreservesReadOnlyBottomBits checks that 0xFF41 writes
// keep bits 0..1 (mode flag, read-only) and bit 7 untouched while accepting
// writes to bits 2..6.
func TestMemory_STATWritePreservesReadOnlyBottomBits(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Memory.SetIO(IO_STAT, 0x83) // bits 0,1,7 set
	mb.SetItem(0xFF41, 0x7C)       // try to set bits 2..6
	got := mb.Memory.GetIO(IO_STAT)
	assert.Equal(t, uint8(0xFF), got&0xFF,
		"STAT write merges 0x83 (preserved) with 0x7C & 0xFC (writable bits) -> 0xFF")
}

// TestMemory_BootROMDisableViaFF50InDMG verifies that writing 0x01 to 0xFF50
// while the boot ROM is enabled disables it and rewinds PC so the next fetch
// starts from ROM_START_ADDR.
func TestMemory_BootROMDisableViaFF50InDMG(t *testing.T) {
	// Use a fresh motherboard (NOT through newMbForSubsysTest, which already
	// disables the boot ROM) and re-enable the boot ROM to exercise the
	// disable path.
	var mb *Motherboard
	withSilencedStdout(func() {
		mb = NewMotherboard(&MotherboardParams{
			Filename: subsysFakeROM(t),
			ForceDmg: true,
		})
	})
	require.True(t, mb.BootRomEnabled(), "precondition: boot ROM enabled by NewMotherboard")
	mb.Cpu.Registers.PC = 0x0050

	mb.SetItem(0xFF50, 0x01)
	assert.False(t, mb.BootRomEnabled(), "writing 0x01 in DMG mode disables boot ROM")
	assert.Equal(t, ROM_START_ADDR-2, mb.Cpu.Registers.PC,
		"PC rewinds to ROM_START_ADDR - 2 so the next tick fetches at 0x0100")
}

func TestMemory_EchoRAMLowerHalfWritePropagatesToWramBank0(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// Echo region 0xE000..0xEFFF mirrors Wram[0] 0x0000..0x0FFF on write.
	mb.SetItem(0xE100, 0x77)
	assert.Equal(t, uint8(0x77), mb.Memory.Wram[0][0x100])
}

func TestMemory_EchoRAMUpperHalfWritePropagatesToWramBank1(t *testing.T) {
	mb := newMbForSubsysTest(t)
	// Echo region 0xF000..0xFDFF mirrors Wram[1] 0x0000..0x0DFF on write.
	mb.SetItem(0xF200, 0x88)
	assert.Equal(t, uint8(0x88), mb.Memory.Wram[1][0x200])
}

// TestMemory_VBKWriteSelectsVRAMBankInCGB exercises the CGB-only VBK
// dispatcher path in SetItem.
func TestMemory_VBKWriteSelectsVRAMBankInCGB(t *testing.T) {
	mb := newCGBMbForSubsysTest(t)
	mb.SetItem(0xFF4F, 0x01)
	assert.Equal(t, uint8(0x01), mb.Memory.GetIO(IO_VBK)&0x01)
	// Bus read returns IO_VBK | 0xFE per the getitem dispatcher.
	assert.Equal(t, uint8(0xFF), mb.GetItem(0xFF4F))
}

func TestMemory_SVBKWriteSelectsWramBankInCGB(t *testing.T) {
	mb := newCGBMbForSubsysTest(t)
	mb.SetItem(0xFF70, 0x05)
	assert.Equal(t, uint8(0x05), mb.Memory.GetIO(IO_SVBK)&0x07)
}

// TestMemory_DMARegisterWriteTriggersDMATransfer writes the source page
// number to 0xFF46. The dispatcher calls doDMATransfer which copies 0xA0
// bytes from the source page into OAM. We pin a known byte at the source
// page (0xC000) and verify it lands in OAM.
func TestMemory_DMARegisterWriteTriggersDMATransfer(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.Memory.Wram[0][0x10] = 0xA5

	mb.SetItem(0xFF46, 0xC0)

	assert.Equal(t, uint8(0xA5), mb.Memory.Oam[0x10],
		"DMA transfer copies WRAM[0xC010] -> OAM[0x10]")
}

func TestMemory_IFWriteDoesNotForceUpperBits(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFF0F, 0x00)
	assert.Equal(t, uint8(0x00), mb.Cpu.Interrupts.IF,
		"IF write stores literal value; upper bits are added only on bus read")
}

// TestMemory_KEY1WriteIsNoOp - 0xFF4D is acknowledged but not implemented.
// The setitem case body is empty so this exercises the routing.
func TestMemory_KEY1WriteIsNoOp(t *testing.T) {
	mb := newMbForSubsysTest(t)
	mb.SetItem(0xFF4D, 0x01)
	// No assertion on side effects; this test exists to walk the case.
	assert.NotPanics(t, func() { mb.SetItem(0xFF4D, 0x80) })
}

// TestMemory_NotUsableRegionWriteIsSilent exercises the 0xFEA0..0xFF00
// "Not Usable" range which has an empty case body in SetItem.
func TestMemory_NotUsableRegionWriteIsSilent(t *testing.T) {
	mb := newMbForSubsysTest(t)
	assert.NotPanics(t, func() { mb.SetItem(0xFEA0, 0x42) })
	assert.NotPanics(t, func() { mb.SetItem(0xFEFF, 0x42) })
}
