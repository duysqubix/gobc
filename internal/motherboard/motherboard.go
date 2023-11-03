package motherboard

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/chigopher/pathlib"
	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/bootrom"
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
	BootRom       *bootrom.BootRom     // Boot ROM
	Timer         *Timer               // Timer
	Lcd           *LCD                 // LCD
	Input         *Input               // Input
	Sound         *APU                 // Sound
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

func (m *Motherboard) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, m.HdmaActive)                        // HDMA active
	binary.Write(buf, binary.LittleEndian, m.HdmaLength)                        // HDMA length
	binary.Write(buf, binary.LittleEndian, m.doubleSpeed)                       // Double speed mode
	binary.Write(buf, binary.LittleEndian, m.Cpu.Serialize().Bytes())           // CPU
	binary.Write(buf, binary.LittleEndian, m.Memory.Serialize().Bytes())        // Memory
	binary.Write(buf, binary.LittleEndian, m.Lcd.Serialize().Bytes())           // LCD
	binary.Write(buf, binary.LittleEndian, m.Input.Serialize().Bytes())         // Input
	binary.Write(buf, binary.LittleEndian, m.Timer.Serialize().Bytes())         // Timer
	binary.Write(buf, binary.LittleEndian, m.BGPalette.Serialize().Bytes())     // BG Palette
	binary.Write(buf, binary.LittleEndian, m.SpritePalette.Serialize().Bytes()) // Sprite Palette
	binary.Write(buf, binary.LittleEndian, m.Cartridge.Serialize().Bytes())     // Cartridge
	binary.Write(buf, binary.LittleEndian, m.Sound.Serialize().Bytes())         // Sound

	return buf
}

func (m *Motherboard) Deserialize(data *bytes.Buffer) error {
	// Read the data from the buffer
	if err := binary.Read(data, binary.LittleEndian, &m.HdmaActive); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &m.HdmaLength); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &m.doubleSpeed); err != nil {
		return err
	}
	if err := m.Cpu.Deserialize(data); err != nil {
		return err
	}
	if err := m.Memory.Deserialize(data); err != nil {
		return err
	}
	if err := m.Lcd.Deserialize(data); err != nil {
		return err
	}
	if err := m.Input.Deserialize(data); err != nil {
		return err
	}
	if err := m.Timer.Deserialize(data); err != nil {
		return err
	}
	if err := m.BGPalette.Deserialize(data); err != nil {
		return err
	}
	if err := m.SpritePalette.Deserialize(data); err != nil {
		return err
	}
	if err := m.Cartridge.Deserialize(data); err != nil {
		return err
	}
	if err := m.Sound.Deserialize(data); err != nil {
		return err
	}

	return nil
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
	mb.Memory = NewInternalRAM(mb, params.Randomize)
	mb.Lcd = NewLCD(mb)
	mb.Sound = NewAPU(mb, true) // forces enable sound
	mb.BootRom = bootrom.NewBootRom(mb.Cgb)
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
	m.Cartridge.Tick(uint64(cycles))

	m.Cpu.Mb.Timer.Tick(cycles, m.Cpu)
	cycles += m.Cpu.handleInterrupts()
	m.Lcd.Tick(cycles)

	m.Sound.Tick(cycles)
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

	srcH := uint16(m.Memory.GetIO(IO_HDMA1)) << 8
	srcL := uint16(m.Memory.GetIO(IO_HDMA2)) & 0xF0
	dstH := (uint16(m.Memory.GetIO(IO_HDMA3)) << 8) & 0x1F00
	dstL := uint16(m.Memory.GetIO(IO_HDMA4)) & 0xF0

	source := srcH | srcL
	destination := dstH | dstL | 0x8000

	// copy the data
	for i := uint16(0); i < length; i++ {
		bank := m.Memory.GetIO(IO_VBK) & 0x01
		m.Memory.SetVram(bank, destination, m.GetItem(source))
		source++
		destination++
	}

	// update the source and destination in RAM
	m.Memory.SetIO(IO_HDMA1, uint8(source>>8))
	m.Memory.SetIO(IO_HDMA2, uint8(source&0xFF))
	m.Memory.SetIO(IO_HDMA3, uint8(destination>>8))
	m.Memory.SetIO(IO_HDMA4, uint8(destination&0xF0))
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
	if m.HdmaActive && !internal.IsBitSet(value, 7) {
		// Abort a HDMA transfer
		m.HdmaActive = false
		m.Memory.SetIO(IO_HDMA5, m.Memory.GetIO(IO_HDMA5)|0x80)

		return
	}

	length := ((uint16(value) & 0x7F) + 1) * 0x10
	// The 7th bit is DMA mode
	if !internal.IsBitSet(value, 7) {
		// Mode 0, general purpose DMA
		m.performNewDMATransfer(length)
		m.Memory.SetIO(IO_HDMA5, 0xFF)
	} else {
		// Mode 1, H-Blank DMA
		internal.ResetBit(&value, 7)
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
		m.Memory.SetIO(IO_HDMA5, m.HdmaLength)
	} else {
		m.HdmaActive = false
		m.Memory.SetIO(IO_HDMA5, 0xFF)
	}
}
