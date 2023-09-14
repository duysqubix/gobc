package motherboard

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
)

var (
	dmgBootRom = filepath.Join(os.TempDir(), "bootrom_dmg.bin")
	cgbBootRom = filepath.Join(os.TempDir(), "bootrom_cgb.bin")
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		logger.Fatal("No caller information")
	}

	dir := filepath.Dir(filename)

	// Now you can use dir to build a path to your file
	localDMGBin := filepath.Join(dir, "bootrom_dmg.bin")
	localCGBBin := filepath.Join(dir, "bootrom_cgb.bin")

	outDMGBin, err := os.Create(filepath.Join(os.TempDir(), "bootrom_dmg.bin"))
	if err != nil {
		logger.Fatal(err)
	}
	defer outDMGBin.Close()

	outCGBBin, err := os.Create(filepath.Join(os.TempDir(), "bootrom_cgb.bin"))
	if err != nil {
		logger.Fatal(err)
	}
	defer outCGBBin.Close()

	err = copyFile(localDMGBin, outDMGBin.Name())
	if err != nil {
		logger.Fatal(err)
	}

	err = copyFile(localCGBBin, outCGBBin.Name())
	if err != nil {
		logger.Fatal(err)
	}

	// Continue with your file processing...
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	err = out.Sync()
	if err != nil {
		return err
	}
	return nil
}

func readBootRomBin(fname string) []uint8 {
	// Open the file.
	f, err := os.Open(fname)
	if err != nil {
		logger.Fatal(err)
	}
	defer f.Close()

	// Get the file size.
	fi, err := f.Stat()
	if err != nil {
		logger.Fatal(err)
	}
	size := fi.Size()

	// Read the file.
	data := make([]uint8, size)
	count, err := f.Read(data)
	if err != nil {
		logger.Fatal(err)
	}
	if int64(count) != size {
		logger.Fatal("read error")
	}

	// calcuate checksum
	var checksum uint8
	for _, b := range data {
		checksum = checksum - b - 1
	}

	logger.Infof("Fname: %s, Size: %d, Checksum: %#x", fname, size, checksum)

	return data
}

func NewBootRom(cgb bool) *BootRom {
	var bootrom []uint8
	if cgb {
		logger.Info("Using CGB bootrom")
		bootrom = readBootRomBin(cgbBootRom)
	} else {
		logger.Info("Using DMG bootrom")
		bootrom = readBootRomBin(dmgBootRom)
	}

	return &BootRom{
		bootrom:   bootrom,
		IsEnabled: true,
	}
}

type BootRom struct {
	bootrom   []uint8
	IsEnabled bool
}

func (br *BootRom) GetItem(addr uint16) uint8 {
	return br.bootrom[addr]
}

func (br *BootRom) Enabled() bool {
	return br.IsEnabled
}

func (br *BootRom) Disable() {
	br.IsEnabled = false
}

func (br *BootRom) Enable() {
	br.IsEnabled = true
}
