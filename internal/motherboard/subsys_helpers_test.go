// Test helpers for the timer, interrupts and memory subsystem tests.
//
// init() below performs load-bearing test infrastructure that MUST stay in
// place: internal.Logger.ExitFunc is set to a no-op so that logger.Fatalf
// calls inside the cartridge loader and the memory dispatchers do not tear
// down the test binary. The logger output writer and level are also pinned
// so the verbose chatter from NewMotherboard does not crowd test output.
//
// stdout redirection is intentionally NOT done globally - that would silence
// `go test -v` itself. Instead newMbForSubsysTest swaps os.Stdout only for
// the duration of NewMotherboard (which calls cartridge.Dump unconditionally
// to os.Stdout).
//
// Helpers in this file are named with a "subsys" / "Mb" prefix to avoid
// colliding with helpers introduced by the parallel CPU/opcode test agent
// that also writes into this package.

package motherboard

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func init() {
	internal.Logger.SetOutput(io.Discard)
	internal.Logger.SetLevel(logrus.PanicLevel)
	internal.Logger.ExitFunc = func(int) {}
}

func withSilencedStdout(fn func()) {
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		fn()
		return
	}
	defer devnull.Close()
	original := os.Stdout
	os.Stdout = devnull
	defer func() { os.Stdout = original }()
	fn()
}

func subsysFakeROM(t *testing.T) *pathlib.Path {
	t.Helper()

	// Game Boy ROM header layout (Pan Docs):
	//   0x0134..0x0142  title (ASCII)
	//   0x0143          CGB flag           (0x00 = DMG only)
	//   0x0146          SGB flag           (0x00 = no SGB)
	//   0x0147          cartridge type     (0x00 = ROM_ONLY)
	//   0x0148          ROM size code      (0x00 = 32 KiB / 2 banks)
	//   0x0149          RAM size code      (0x00 = no SRAM)
	//   0x014A          destination code
	//   0x014B          old licensee
	//   0x014C          mask ROM version
	//   0x014D          header checksum    (computed below)
	const (
		memBank       = 0x4000
		cartTypeAddr  = 0x0147
		romSizeAddr   = 0x0148
		ramSizeAddr   = 0x0149
		cgbFlagAddr   = 0x0143
		sgbFlagAddr   = 0x0146
		oldLicAddr    = 0x014B
		destCodeAddr  = 0x014A
		maskRomVerAdr = 0x014C
		titleStart    = 0x0134
		titleEnd      = 0x0142
		hdrChecksum   = 0x014D
	)

	rom := make([]byte, memBank*2)
	for i := range rom {
		rom[i] = 0xFF
	}
	rom[cartTypeAddr] = 0x00
	rom[romSizeAddr] = 0x00
	rom[ramSizeAddr] = 0x00
	rom[cgbFlagAddr] = 0x00
	rom[sgbFlagAddr] = 0x00
	rom[oldLicAddr] = 0x00
	rom[destCodeAddr] = 0x00
	rom[maskRomVerAdr] = 0x00
	for i := titleStart; i <= titleEnd; i++ {
		rom[i] = 0x00
	}

	// Header checksum (Pan Docs algorithm), computed over 0x0134..0x014C.
	var checksum uint8 = 0
	for i := titleStart; i <= maskRomVerAdr; i++ {
		checksum -= rom[i] + 1
	}
	rom[hdrChecksum] = checksum

	dir := t.TempDir()
	fp := filepath.Join(dir, "subsys.gb")
	require.NoError(t, os.WriteFile(fp, rom, 0o644))
	return pathlib.NewPath(fp)
}

func newMbForSubsysTest(t *testing.T) *Motherboard {
	t.Helper()
	return newMbForSubsysTestMode(t, false)
}

func newCGBMbForSubsysTest(t *testing.T) *Motherboard {
	t.Helper()
	return newMbForSubsysTestMode(t, true)
}

func newMbForSubsysTestMode(t *testing.T, cgb bool) *Motherboard {
	t.Helper()

	var mb *Motherboard
	withSilencedStdout(func() {
		mb = NewMotherboard(&MotherboardParams{
			Filename:     subsysFakeROM(t),
			Randomize:    false,
			ForceCgb:     cgb,
			ForceDmg:     !cgb,
			Breakpoints:  nil,
			Decouple:     false,
			PanicOnStuck: false,
		})
	})

	// Bring the motherboard to a deterministic post-bootrom state so reads
	// against 0x0000..0x00FF route to the cartridge rather than the boot ROM.
	mb.BootRom.Disable()
	mb.Cpu.Registers.PC = ROM_START_ADDR
	mb.Cpu.Registers.SP = 0xFFFE
	mb.Cpu.Interrupts.IE = 0
	mb.Cpu.Interrupts.IF = 0
	mb.Cpu.Interrupts.InterruptsOn = false
	mb.Cpu.Interrupts.InterruptsEnabling = false
	mb.Timer.Reset()

	return mb
}
