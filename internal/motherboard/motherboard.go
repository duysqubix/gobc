package motherboard

import (
	"fmt"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
)

var logger = internal.Logger

type Breakpoints struct {
	Enabled     bool     // Breakpoints enabled
	Addrs       []uint16 // Breakpoints addresses
	LastElement uint16   // Last element of breakpoints
}

type Motherboard struct {
	Cpu           *CPU                 // CPU
	Cartridge     *cartridge.Cartridge // Cartridge
	Memory        *Memory              // Internal RAM
	BootRom       *BootRom             // Boot ROM
	Timer         *Timer               // Timer
	Lcd           *LCD                 // LCD
	Input         *Input               // Input
	Cgb           bool                 // Color Gameboy
	CpuFreq       uint32               // CPU frequency
	Randomize     bool                 // Randomize RAM on startup
	BGPalette     *cgbPalette          // Background palette
	SpritePalette *cgbPalette          // Sprite palette

	HdmaActive  bool  // HDMA active
	HdmaLength  uint8 // HDMA length
	doubleSpeed bool  // Double speed mode

	// debugging
	Decouple     bool         // Decouple Motherboard from other components, and all calls to read/write memory will be mocked
	Breakpoints  *Breakpoints // Breakpoints
	PanicOnStuck bool         // Panic when CPU is stuck
	GuiPause     bool         // Pause GUI
}

type MotherboardParams struct {
	Filename     *pathlib.Path
	Randomize    bool
	ForceCgb     bool
	ForceDmg     bool
	Breakpoints  []uint16
	Decouple     bool
	PanicOnStuck bool
}

func NewMotherboard(params *MotherboardParams) *Motherboard {

	var cart *cartridge.Cartridge = cartridge.NewCartridge(params.Filename)

	var bp *Breakpoints
	if len(params.Breakpoints) > 0 {
		logger.Debug("Breakpoints enabled")

		last_element := params.Breakpoints[len(params.Breakpoints)-1]

		bp = &Breakpoints{
			Enabled:     true,
			Addrs:       params.Breakpoints,
			LastElement: last_element,
		}
	} else {
		bp = &Breakpoints{
			Enabled: false,
		}
	}

	mb := &Motherboard{
		Cartridge:     cart,
		Randomize:     params.Randomize,
		Decouple:      params.Decouple,
		Timer:         NewTimer(),
		Breakpoints:   bp,
		PanicOnStuck:  params.PanicOnStuck,
		BGPalette:     NewPalette(),
		SpritePalette: NewPalette(),
	}

	mb.Cgb = mb.Cartridge.CgbModeEnabled() || params.ForceCgb

	if mb.Cgb && params.ForceDmg {
		mb.Cgb = false
	}

	mb.CpuFreq = internal.DMG_CLOCK_SPEED

	if mb.Cgb {
		mb.CpuFreq = internal.CGB_CLOCK_SPEED
	}

	mb.Input = NewInput(mb)
	mb.Cpu = NewCpu(mb)
	mb.Memory = NewInternalRAM(mb.Cgb, params.Randomize)
	mb.Lcd = NewLCD(mb)
	mb.BootRom = NewBootRom(mb.Cgb)
	mb.BootRom.Enable()
	// mb.BootRom.Disable()

	if !mb.BootRomEnabled() {
		logger.Info("Boot ROM not enabled. Jumping to 0x100")
		mb.Cpu.Registers.PC = ROM_START_ADDR
	} else {
		logger.Info("Boot ROM enabled. Jumping to 0x0")
		mb.Cpu.Registers.PC = BOOTROM_START_ADDR
	}

	return mb
}

func (m *Motherboard) Reset() {
	m.Cpu.Reset()
	m.Memory.Reset()
	m.Lcd.Reset()
	m.BootRom.Enable()
	// m.BootRom.Disable()
	m.Timer.Reset()

	if !m.BootRomEnabled() {
		logger.Info("Boot ROM not enabled. Jumping to 0x100")
		m.Cpu.Registers.PC = ROM_START_ADDR
	} else {
		logger.Info("Boot ROM enabled. Jumping to 0x0")
		m.Cpu.Registers.PC = BOOTROM_START_ADDR
	}
}

func (m *Motherboard) BootRomEnabled() bool {
	return m.BootRom.IsEnabled
}

func (m *Motherboard) Tick() (bool, OpCycles) {
	defer func() {
		if r := recover(); r != nil {
			df := internal.StateDumpFile()
			logger.SetOutput(df)
			// dump CPU State
			fmt.Fprintln(df, "-----CRASH DETECTED-----")
			fmt.Fprintln(df, "----- CPU STATE -----")
			m.Cpu.DumpState(df)
			fmt.Fprintln(df, "----- MEMORY STATE -----")
			m.Memory.DumpState(df)
			fmt.Fprintln(df, "-----END CRASH-----")
			fmt.Println("Crash detected.. check log file for more details")
			logger.Panic()
		}
	}()
	var cycles OpCycles = 4

	if m.Cpu.Stopped || m.Cpu.IsStuck || m.GuiPause {
		return false, cycles
	}

	if !m.Cpu.Halted {
		cycles = m.Cpu.Tick()
	}

	m.Cpu.Mb.Timer.Tick(cycles, m.Cpu)
	m.Lcd.Tick(cycles)

	cycles += m.Cpu.handleInterrupts()
	return true, cycles
}

func (m *Motherboard) ButtonEvent(key Key) {
	result := m.Input.KeyEvent(key)
	if result != 0 {
		m.Cpu.SetInterruptFlag(INTR_HIGHTOLOW)
	}
}

func (m *Motherboard) performNewDMATransfer(length uint16) {

	// load the source and destination from RAM
	source := (uint16(m.Memory.IO[IO_HDMA1-IO_START_ADDR])<<8 | uint16(m.Memory.IO[IO_HDMA2-IO_START_ADDR])) & 0xFFF0
	destination := (uint16(m.Memory.IO[IO_HDMA3-IO_START_ADDR])<<8 | uint16(m.Memory.IO[IO_HDMA4-IO_START_ADDR])) & 0x1FF0
	destination |= 0x8000

	srcH := uint16(m.Memory.IO[IO_HDMA1-IO_START_ADDR]) << 8
	srcL := uint16(m.Memory.IO[IO_HDMA2-IO_START_ADDR]) & 0xF0
	dstH := (uint16(m.Memory.IO[IO_HDMA3-IO_START_ADDR]) << 8) & 0x1F00
	dstL := uint16(m.Memory.IO[IO_HDMA4-IO_START_ADDR]) & 0xF0

	source = srcH | srcL
	destination = dstH | dstL

	// copy the data
	for i := uint16(0); i < length; i++ {
		// m.SetItem(destination, uint16(m.GetItem(source)))
		// srcData :=
		m.Memory.Vram[m.Memory.ActiveVramBank()][destination] = m.GetItem(source)
		source++
		destination++
	}

	// update the source and destination in RAM
	m.Memory.IO[IO_HDMA1-IO_START_ADDR] = uint8(source >> 8)
	m.Memory.IO[IO_HDMA2-IO_START_ADDR] = uint8(source & 0xFF)
	m.Memory.IO[IO_HDMA3-IO_START_ADDR] = uint8(destination >> 8)
	m.Memory.IO[IO_HDMA4-IO_START_ADDR] = uint8(destination & 0xF0)
}

// Perform a DMA transfer.
func (m *Motherboard) doDMATransfer(value byte) {
	address := uint16(value) << 8 // (data * 100)

	var i uint16
	for i = 0; i < 0xA0; i++ {
		m.SetItem(0xFE00+i, uint16(m.GetItem(address+i)))
	}
}

// Start a CGB DMA transfer.
func (m *Motherboard) doNewDMATransfer(value byte) {
	// if m.HdmaActive && bits.Val(value, 7) == 0 {

	if m.HdmaActive && !internal.IsBitSet(value, 7) {
		// Abort a HDMA transfer
		m.HdmaActive = false
		// m.Memory.Hram[0x55] |= 0x80 // Set bit 7
		m.Memory.IO[IO_HDMA5-IO_START_ADDR] |= 0x80

		return
	}

	length := ((uint16(value) & 0x7F) + 1) * 0x10
	// The 7th bit is DMA mode
	if !internal.IsBitSet(value, 7) {
		// Mode 0, general purpose DMA
		m.performNewDMATransfer(length)
		// m.Memory.Hram[0x55] = 0xFF
		m.Memory.IO[IO_HDMA5-IO_START_ADDR] = 0xFF
	} else {
		// Mode 1, H-Blank DMA
		logger.Debugf("Starting HDMA transfer of %d bytes", length)
		m.HdmaLength = uint8(value)
		m.HdmaActive = true
	}
}

func (m *Motherboard) DoHDMATransfer() {
	if !m.HdmaActive {
		return
	}
	m.performNewDMATransfer(0x10)
	if m.HdmaLength > 0 {
		m.HdmaLength--
		m.Memory.IO[IO_HDMA5-IO_START_ADDR] = m.HdmaLength
	} else {
		m.HdmaActive = false
		m.Memory.IO[IO_HDMA5-IO_START_ADDR] = 0xFF
	}
}
