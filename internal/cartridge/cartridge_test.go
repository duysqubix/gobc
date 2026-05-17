package cartridge

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	internal.Logger.SetOutput(io.Discard)
	internal.Logger.SetLevel(logrus.PanicLevel)
	internal.Logger.ExitFunc = func(int) {}

	os.Exit(m.Run())
}

type romOption func(*[]byte)

func withType(b uint8) romOption {
	return func(r *[]byte) { (*r)[CARTRIDGE_TYPE_ADDR] = b }
}

func withRomSize(b uint8) romOption {
	return func(r *[]byte) {
		(*r)[ROM_SIZE_ADDR] = b
		banks, ok := romBankCountForByte(b)
		if !ok {
			return
		}
		want := int(banks) * int(MEMORY_BANK_SIZE)
		switch {
		case len(*r) < want:
			pad := make([]byte, want-len(*r))
			for i := range pad {
				pad[i] = 0xFF
			}
			*r = append(*r, pad...)
		case len(*r) > want:
			*r = (*r)[:want]
		}
	}
}

func withRamSize(b uint8) romOption {
	return func(r *[]byte) { (*r)[SRAM_SIZE_ADDR] = b }
}

func withCGBFlag(b uint8) romOption {
	return func(r *[]byte) { (*r)[CBG_FLAG_ADDR] = b }
}

func withSGBFlag(b uint8) romOption {
	return func(r *[]byte) { (*r)[SGB_FLAG_ADDR] = b }
}

func withTitle(s string) romOption {
	return func(r *[]byte) {
		for i := TITLE_START_ADDR; i <= TITLE_END_ADDR; i++ {
			(*r)[i] = 0x00
		}
		max := int(TITLE_END_ADDR-TITLE_START_ADDR) + 1
		title := []byte(s)
		if len(title) > max {
			title = title[:max]
		}
		copy((*r)[TITLE_START_ADDR:], title)
	}
}

func withOldLicensee(b uint8) romOption {
	return func(r *[]byte) { (*r)[OLD_LICENSEE_CODE_ADDR] = b }
}

func romBankCountForByte(b uint8) (uint16, bool) {
	switch b {
	case 0x00:
		return 2, true
	case 0x01:
		return 4, true
	case 0x02:
		return 8, true
	case 0x03:
		return 16, true
	case 0x04:
		return 32, true
	case 0x05:
		return 64, true
	case 0x06:
		return 128, true
	case 0x07:
		return 256, true
	case 0x08:
		return 512, true
	}
	return 0, false
}

func buildROM(opts ...romOption) []byte {
	rom := make([]byte, int(MEMORY_BANK_SIZE)*2)
	for i := range rom {
		rom[i] = 0xFF
	}

	rom[CARTRIDGE_TYPE_ADDR] = 0x00
	rom[ROM_SIZE_ADDR] = 0x00
	rom[SRAM_SIZE_ADDR] = 0x00
	rom[CBG_FLAG_ADDR] = 0x00
	rom[SGB_FLAG_ADDR] = 0x00
	rom[OLD_LICENSEE_CODE_ADDR] = 0x00
	rom[DESTINATION_CODE_ADDR] = 0x00
	rom[MASK_ROM_VERSION_NUMBER_ADDR] = 0x00
	for i := TITLE_START_ADDR; i <= TITLE_END_ADDR; i++ {
		rom[i] = 0x00
	}

	for _, opt := range opts {
		opt(&rom)
	}

	var checksum uint8 = 0
	for i := TITLE_START_ADDR; i <= MASK_ROM_VERSION_NUMBER_ADDR; i++ {
		checksum -= rom[i] + 1
	}
	rom[HEADER_CHECKSUM_ADDR] = checksum

	return rom
}

func writeTempROM(t *testing.T, rom []byte) *pathlib.Path {
	t.Helper()
	dir := t.TempDir()
	fp := filepath.Join(dir, "fake.gb")
	require.NoError(t, os.WriteFile(fp, rom, 0o644))
	return pathlib.NewPath(fp)
}

func makeFakeROM(t *testing.T, opts ...romOption) *pathlib.Path {
	t.Helper()
	return writeTempROM(t, buildROM(opts...))
}

func newCartFromHeader(bank0 []uint8) *Cartridge {
	banks := make([][]uint8, 1)
	banks[0] = make([]uint8, MEMORY_BANK_SIZE)
	copy(banks[0], bank0)
	return &Cartridge{RomBanks: banks, RomBanksCount: 1}
}

func typeNameOf(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
}

func TestMakeFakeROM_DefaultsAreValid(t *testing.T) {
	path := makeFakeROM(t)
	data, err := os.ReadFile(path.String())
	require.NoError(t, err)
	assert.Len(t, data, int(MEMORY_BANK_SIZE)*2, "default ROM should be 32 KiB")
	assert.Equal(t, uint8(0x00), data[CARTRIDGE_TYPE_ADDR])
	assert.Equal(t, uint8(0x00), data[ROM_SIZE_ADDR])

	cart := NewCartridge(path)
	require.NotNil(t, cart)
	calc, ok := cart.ValidateChecksum()
	assert.True(t, ok, "default ROM header checksum should be valid")
	assert.Equal(t, cart.RomBanks[0][HEADER_CHECKSUM_ADDR], calc)
}

func TestCartridge_HeaderChecksum_Valid(t *testing.T) {
	rom := buildROM(withTitle("TESTGAME"))
	cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
	calc, ok := cart.ValidateChecksum()
	assert.True(t, ok)
	assert.Equal(t, rom[HEADER_CHECKSUM_ADDR], calc,
		"computed checksum should match the byte stored at 0x14D")
}

func TestCartridge_HeaderChecksum_Invalid(t *testing.T) {
	rom := buildROM(withTitle("BROKEN"))
	rom[HEADER_CHECKSUM_ADDR] ^= 0xFF
	cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
	_, ok := cart.ValidateChecksum()
	assert.False(t, ok, "corrupted header should not validate")
}

func TestCartridge_HeaderChecksum_FormulaMatchesPanDocs(t *testing.T) {
	rom := buildROM()
	for i := TITLE_START_ADDR; i <= MASK_ROM_VERSION_NUMBER_ADDR; i++ {
		rom[i] = 0x00
	}
	var sum uint8
	for i := TITLE_START_ADDR; i <= MASK_ROM_VERSION_NUMBER_ADDR; i++ {
		sum -= rom[i] + 1
	}
	rom[HEADER_CHECKSUM_ADDR] = sum

	cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
	calc, ok := cart.ValidateChecksum()
	assert.True(t, ok)
	assert.Equal(t, uint8(0xE7), calc,
		"all-zero header bytes 0x134..0x14C should produce checksum 0xE7")
}

func TestCartridge_CGBFlag(t *testing.T) {
	cases := []struct {
		name string
		flag uint8
		want bool
	}{
		{"DMG only (0x00)", 0x00, false},
		{"CGB supported (0x80)", 0x80, true},
		{"CGB only (0xC0)", 0xC0, true},
		{"unrelated byte (0x42)", 0x42, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rom := buildROM(withCGBFlag(tc.flag))
			cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
			assert.Equal(t, tc.want, cart.CgbModeEnabled())
		})
	}
}

func TestCartridge_CGBFlag_NoRomBanksReturnsFalse(t *testing.T) {
	cart := &Cartridge{RomBanks: nil, RomBanksCount: 0}
	assert.False(t, cart.CgbModeEnabled())
}

func TestCartridge_SGBFlag(t *testing.T) {
	cases := []struct {
		name     string
		flag     uint8
		wantSubs string
	}{
		{"SGB off (0x00)", 0x00, "No"},
		{"SGB on (0x03)", 0x03, "Yes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rom := buildROM(withSGBFlag(tc.flag))
			cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
			var buf bytes.Buffer
			cart.Dump(&buf)
			assert.Contains(t, buf.String(), "SBG Mode")
			assert.Contains(t, buf.String(), tc.wantSubs)
		})
	}
}

func TestCartridge_TitleExtraction(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  string
	}{
		{"short ASCII", "ZELDA", "ZELDA"},
		{"empty title", "", ""},
		{"14 char max", "AAAAAAAAAAAAAA", "AAAAAAAAAAAAAA"},
		{"longer than 14 truncated", "ABCDEFGHIJKLMNOP", "ABCDEFGHIJKLMN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rom := buildROM(withTitle(tc.title))
			cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
			got := cart.GetTitle()
			padded := []byte(tc.want)
			for len(padded) < 14 {
				padded = append(padded, 0x00)
			}
			assert.Equal(t, string(padded), got)
		})
	}
}

func TestCartridge_GetFilename_StripsExtension(t *testing.T) {
	cart := &Cartridge{Filename: "/some/dir/game.gb"}
	assert.Equal(t, "game", cart.GetFilename())

	cart2 := &Cartridge{Filename: "POKEMON.gbc"}
	assert.Equal(t, "POKEMON", cart2.GetFilename())
}

func TestRomSizeMap_KnownEntries(t *testing.T) {
	cases := []struct {
		b         uint8
		wantName  string
		wantBanks uint16
	}{
		{0x00, "32 KiB", 2},
		{0x01, "64 KiB", 4},
		{0x02, "128 KiB", 8},
		{0x03, "256 KiB", 16},
		{0x04, "512 KiB", 32},
		{0x05, "1 MiB", 64},
		{0x06, "2 MiB", 128},
		{0x07, "4 MiB", 256},
		{0x08, "8 MiB", 512},
	}
	for _, tc := range cases {
		entry, ok := RomSizeMap[tc.b]
		require.True(t, ok, "rom size byte %#x should be in map", tc.b)
		assert.Equal(t, tc.wantName, entry.name)
		assert.Equal(t, tc.wantBanks, entry.value)
	}
}

func TestRamSizeMap_KnownEntries(t *testing.T) {
	cases := []struct {
		b         uint8
		wantBanks uint16
	}{
		{0x00, 0},
		{0x02, 1},
		{0x03, 4},
		{0x04, 16},
		{0x05, 8},
	}
	for _, tc := range cases {
		entry, ok := RamSizeMap[tc.b]
		require.True(t, ok, "ram size byte %#x should be in map", tc.b)
		assert.Equal(t, tc.wantBanks, entry.value)
	}
}

func TestTupleString_FormatsHumanReadable(t *testing.T) {
	tu := tuple{name: "32 KiB", value: 2}
	assert.Equal(t, "32 KiB (2, 16KiB banks)", tu.String())
}

func TestCartridge_LicenseeCode_NewCodePath(t *testing.T) {
	cases := []struct {
		b    uint8
		want string
	}{
		{0x00, "None"},
		{0x01, "Nintendo R&D1"},
		{0x08, "Capcom"},
		{0x13, "Electronic Arts"},
		{0x34, "Konami"},
		{0xA4, "Konami (Yu-Gi-Oh!)"},
	}
	for _, tc := range cases {
		got, ok := NewLicenseeCodeMap[tc.b]
		require.True(t, ok, "new licensee %#x missing from map", tc.b)
		assert.Equal(t, tc.want, got)
	}
}

func TestCartridge_LicenseeCode_OldCodePath(t *testing.T) {
	cases := []struct {
		b    uint8
		want string
	}{
		{0x00, "None"},
		{0x01, "Nintendo"},
		{0x08, "Capcom"},
		{0x33, "Use New Licensee Code"},
	}
	for _, tc := range cases {
		got, ok := OldLicenseeCodeMap[tc.b]
		require.True(t, ok, "old licensee %#x missing from map", tc.b)
		assert.Equal(t, tc.want, got)
	}
}

func TestCartridge_LicenseeCode_ExposedViaDump(t *testing.T) {
	rom := buildROM(withOldLicensee(0x01))
	cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
	var buf bytes.Buffer
	cart.Dump(&buf)
	assert.Contains(t, buf.String(), "Old Licensee Code")
	assert.Contains(t, buf.String(), "Nintendo")
}

func TestCartridgeTypeMap_CoversCommonVariants(t *testing.T) {
	cases := map[uint8]string{
		0x00: "ROM ONLY",
		0x01: "MBC1",
		0x02: "MBC1+RAM",
		0x03: "MBC1+RAM+BATTERY",
		0x05: "MBC2",
		0x0F: "MBC3+TIMER+BATTERY",
		0x10: "MBC3+TIMER+RAM+BATTERY",
		0x13: "MBC3+RAM+BATTERY",
		0x19: "MBC5",
		0x1B: "MBC5+RAM+BATTERY",
	}
	for b, want := range cases {
		got, ok := CartridgeTypeMap[b]
		require.True(t, ok, "byte %#x should be in CartridgeTypeMap", b)
		assert.Equal(t, want, got)
	}
}

func TestCartridge_GetCartType_ReturnsHumanReadableLabel(t *testing.T) {
	rom := buildROM(withType(0x13))
	cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
	assert.Equal(t, "MBC3+RAM+BATTERY", cart.GetCartType())
}

func TestLoadRomBanks_DummyData(t *testing.T) {
	banks := LoadRomBanks(nil, true)
	require.Len(t, banks, 1)
	assert.Equal(t, uint8(0xFF), banks[0][0])
	assert.Equal(t, uint8(0x00), banks[0][CARTRIDGE_TYPE_ADDR])
}

func TestLoadRomBanks_RealData(t *testing.T) {
	rom := buildROM(withRomSize(0x01))
	banks := LoadRomBanks(rom, false)
	assert.Len(t, banks, 4)
	for i, b := range banks {
		assert.Equal(t, int(MEMORY_BANK_SIZE), len(b),
			"bank %d should be exactly MEMORY_BANK_SIZE bytes", i)
	}
}

func TestNewCartridge_HeaderParsing(t *testing.T) {
	cases := []struct {
		name        string
		typeByte    uint8
		wantTypeFmt string
		check       func(t *testing.T, cart *Cartridge)
	}{
		{
			name:        "ROM_ONLY (0x00)",
			typeByte:    0x00,
			wantTypeFmt: "*cartridge.RomOnlyCartridge",
		},
		{
			name:        "MBC1 (0x01)",
			typeByte:    0x01,
			wantTypeFmt: "*cartridge.Mbc1Cartridge",
			check: func(t *testing.T, cart *Cartridge) {
				mbc := cart.CartType.(*Mbc1Cartridge)
				assert.False(t, mbc.hasBattery, "plain MBC1 has no battery")
			},
		},
		{
			name:        "MBC1+RAM (0x02)",
			typeByte:    0x02,
			wantTypeFmt: "*cartridge.Mbc1Cartridge",
		},
		{
			name:        "MBC1+RAM+BATTERY (0x03)",
			typeByte:    0x03,
			wantTypeFmt: "*cartridge.Mbc1Cartridge",
			check: func(t *testing.T, cart *Cartridge) {
				mbc := cart.CartType.(*Mbc1Cartridge)
				assert.True(t, mbc.hasBattery)
			},
		},
		{
			name:        "MBC3+TIMER+BATTERY (0x0F)",
			typeByte:    0x0F,
			wantTypeFmt: "*cartridge.Mbc3Cartridge",
			check: func(t *testing.T, cart *Cartridge) {
				mbc := cart.CartType.(*Mbc3Cartridge)
				assert.True(t, mbc.hasBattery)
				assert.True(t, mbc.hasRTC)
				assert.True(t, cart.RtcEnabled, "MBC3+TIMER should enable RTC")
			},
		},
		{
			name:        "MBC3+TIMER+RAM+BATTERY (0x10)",
			typeByte:    0x10,
			wantTypeFmt: "*cartridge.Mbc3Cartridge",
			check: func(t *testing.T, cart *Cartridge) {
				mbc := cart.CartType.(*Mbc3Cartridge)
				assert.True(t, mbc.hasBattery)
				assert.True(t, mbc.hasRTC)
			},
		},
		{
			name:        "MBC3+RAM+BATTERY (0x13)",
			typeByte:    0x13,
			wantTypeFmt: "*cartridge.Mbc3Cartridge",
			check: func(t *testing.T, cart *Cartridge) {
				mbc := cart.CartType.(*Mbc3Cartridge)
				assert.True(t, mbc.hasBattery)
				assert.False(t, mbc.hasRTC, "0x13 should not enable RTC")
			},
		},
		{
			name:        "MBC5 (0x19)",
			typeByte:    0x19,
			wantTypeFmt: "*cartridge.Mbc5Cartridge",
			check: func(t *testing.T, cart *Cartridge) {
				mbc := cart.CartType.(*Mbc5Cartridge)
				assert.Equal(t, uint8(1), mbc.romBankLow)
				assert.Equal(t, uint8(0), mbc.romBankHi)
			},
		},
		{
			name:        "MBC5+RAM+BATTERY (0x1B)",
			typeByte:    0x1b,
			wantTypeFmt: "*cartridge.Mbc5Cartridge",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := makeFakeROM(t, withType(tc.typeByte))
			cart := NewCartridge(path)
			require.NotNil(t, cart)
			gotType := typeNameOf(cart.CartType)
			assert.Equal(t, tc.wantTypeFmt, gotType,
				"cartridge byte %#x should map to type %s", tc.typeByte, tc.wantTypeFmt)
			if tc.check != nil {
				tc.check(t, cart)
			}
		})
	}
}

func TestNewCartridge_ROMSize(t *testing.T) {
	cases := []struct {
		b         uint8
		wantBanks uint16
		humanSize string
	}{
		{0x00, 2, "32 KiB"},
		{0x01, 4, "64 KiB"},
		{0x02, 8, "128 KiB"},
		{0x05, 64, "1 MiB"},
		{0x07, 256, "4 MiB"},
	}
	for _, tc := range cases {
		t.Run(tc.humanSize, func(t *testing.T) {
			path := makeFakeROM(t, withRomSize(tc.b))
			cart := NewCartridge(path)
			require.NotNil(t, cart)
			assert.Equal(t, tc.wantBanks, cart.RomBanksCount)
			assert.Len(t, cart.RomBanks, int(tc.wantBanks))
		})
	}
}

func TestNewCartridge_RAMSize(t *testing.T) {
	cases := []struct {
		b         uint8
		wantBanks uint16
		label     string
	}{
		// MBC1+RAM (type 0x02) with SRAM_SIZE=0 in the header is the
		// "phantom RAM" case: real hardware ships some such carts with
		// 8 KiB of RAM wired on the MBC itself, and Blargg's halt_bug /
		// interrupt_time / mem_timing ROMs rely on it for the cart-RAM
		// pass/fail signature at $A000-$A003. Promote 0 -> 1 bank.
		{0x00, 1, "phantom 8 KiB (type 0x02 with size byte 0)"},
		{0x02, 1, "8 KiB"},
		{0x03, 4, "32 KiB"},
		{0x04, 16, "128 KiB"},
		{0x05, 8, "64 KiB"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			path := makeFakeROM(t, withType(0x02), withRamSize(tc.b))
			cart := NewCartridge(path)
			require.NotNil(t, cart)
			assert.Equal(t, tc.wantBanks, cart.RamBankCount)
		})
	}
}

// TestNewCartridge_NoRAMTypeKeepsZero verifies the phantom-RAM
// promotion only kicks in for +RAM cart types; a plain ROM_ONLY
// cart with size=0 stays at 0 banks.
func TestNewCartridge_NoRAMTypeKeepsZero(t *testing.T) {
	path := makeFakeROM(t, withType(0x00), withRamSize(0x00))
	cart := NewCartridge(path)
	require.NotNil(t, cart)
	assert.Equal(t, uint16(0), cart.RamBankCount)
}

func TestNewCartridge_PreservesTitle(t *testing.T) {
	path := makeFakeROM(t, withTitle("TEST"))
	cart := NewCartridge(path)
	require.NotNil(t, cart)
	assert.Equal(t, "TEST\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", cart.GetTitle())
}

func TestNewCartridge_DetectsCGBFlag(t *testing.T) {
	pathDmg := makeFakeROM(t, withCGBFlag(0x00))
	pathCgb := makeFakeROM(t, withCGBFlag(0x80))
	pathCgbOnly := makeFakeROM(t, withCGBFlag(0xC0))

	assert.False(t, NewCartridge(pathDmg).CgbModeEnabled())
	assert.True(t, NewCartridge(pathCgb).CgbModeEnabled())
	assert.True(t, NewCartridge(pathCgbOnly).CgbModeEnabled())
}

func TestCartridge_Dump_ContainsExpectedFields(t *testing.T) {
	rom := buildROM(
		withTitle("DUMPTEST"),
		withType(0x13),
		withRomSize(0x01),
		withRamSize(0x03),
		withCGBFlag(0x80),
		withSGBFlag(0x03),
	)
	cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
	cart.Filename = "DUMPTEST.gb"

	var buf bytes.Buffer
	cart.Dump(&buf)
	out := buf.String()

	assert.Contains(t, out, "DUMPTEST")
	assert.Contains(t, out, "MBC3+RAM+BATTERY")
	assert.Contains(t, out, "64 KiB")
	assert.Contains(t, out, "32 KiB")
	assert.Contains(t, out, "CGB")
	assert.Contains(t, out, "SBG Mode")
}

func TestCartridge_RawHeaderDump_DoesNotPanic(t *testing.T) {
	rom := buildROM(withType(0x01))
	cart := newCartFromHeader(rom[:MEMORY_BANK_SIZE])
	assert.NotPanics(t, func() { cart.RawHeaderDump() })
}

func TestCartridge_Tick_AdvancesRtcWhenEnabled(t *testing.T) {
	cart := &Cartridge{RtcEnabled: true}
	Grtc = NewRTC()
	cart.Tick(RTCCycles)
	assert.Equal(t, uint8(1), Grtc.s, "tick of RTCCycles should advance seconds by 1")
}

func TestCartridge_Tick_NoopWhenRtcDisabled(t *testing.T) {
	cart := &Cartridge{RtcEnabled: false}
	Grtc = NewRTC()
	cart.Tick(RTCCycles * 5)
	assert.Equal(t, uint8(0), Grtc.s, "tick should be a no-op when RtcEnabled=false")
}

func TestCartridge_Serialize_Deserialize_RoundTrip(t *testing.T) {
	path := makeFakeROM(t, withType(0x01))
	src := NewCartridge(path)
	require.NotNil(t, src)
	src.RamBankCount = 1
	src.RamBankSelected = 0
	src.RamBankEnabled = true
	src.MemoryModel = 1
	src.RamBanks[0][0x10] = 0xAB

	buf := src.Serialize()

	dst := NewCartridge(path)
	require.NotNil(t, dst)
	require.NoError(t, dst.Deserialize(bytes.NewBuffer(buf.Bytes())))

	assert.Equal(t, src.MemoryModel, dst.MemoryModel)
	assert.Equal(t, src.RamBankEnabled, dst.RamBankEnabled)
	assert.Equal(t, src.RamBankSelected, dst.RamBankSelected)
	assert.Equal(t, src.RamBanks[0][0x10], dst.RamBanks[0][0x10])
}
