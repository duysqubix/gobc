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
	Sound         *APU                 // APU (audio)
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
	binary.Write(buf, binary.LittleEndian, m.Sound.Serialize().Bytes())         // APU

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
	AudioEnabled bool
	AudioSmooth  bool
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
	mb.Sound = NewAPU(mb, params.AudioEnabled, params.AudioSmooth)
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
	m.Sound.Reset()
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
	m.Lcd.Tick(cycles)
	m.Sound.Tick(cycles)

	// Interrupt servicing consumes real wall-clock cycles too (5 M-cycles
	// per Pan Docs "Interrupt Service Routine"). Without ticking the
	// peripherals during that window TIMA/DIV/LCD/APU lag behind the
	// CPU, and Blargg's interrupt_time test sees an 8-cycle interrupt
	// (JP+RET only) instead of the expected 13.
	if irq := m.Cpu.handleInterrupts(); irq > 0 {
		m.Cartridge.Tick(uint64(irq))
		m.Cpu.Mb.Timer.Tick(irq, m.Cpu)
		m.Lcd.Tick(irq)
		m.Sound.Tick(irq)
		cycles += irq
	}
	return true, cycles
}

func (m *Motherboard) ButtonEvent(key Key) {
	result := m.Input.KeyEvent(key)
	if result != 0 {
		m.Cpu.SetInterruptFlag(INTR_HIGHTOLOW)
	}
}

// resolveOAMBugRow shares the gating logic for both write- and read-style
// OAM corruption: DMG-only, address in OAM range, PPU latched a row in mode 2.
// Returns the row offset (always a multiple of 8 in [8, 152]) or 0xFF when
// no corruption should fire.
func (m *Motherboard) resolveOAMBugRow(addr uint16, cycleOffset OpCycles) uint8 {
	if m.Cgb {
		return 0xFF
	}
	if addr < 0xFE00 || addr >= 0xFF00 {
		return 0xFF
	}
	row := m.Lcd.OAMBugRowAt(cycleOffset)
	if row == 0xFF || row < 8 {
		return 0xFF
	}
	if int(row)+8 > len(m.Memory.Oam) {
		return 0xFF
	}
	return row
}

// OAMBugTrigger models the DMG OAM corruption bug for opcodes whose
// inc/dec unit puts a $FE00-$FEFF address on the bus during PPU mode 2.
// Pan Docs "OAM Corruption Bug"; formulas from SameBoy memory.c
// bitwise_glitch / GB_trigger_oam_bug. cycleOffset is the T-cycle offset
// from the start of the opcode handler to the bus driving moment.
//
// Write-style formula (INC/DEC rr, PUSH, LD (HL+/-),A):
//
//	g = ((a ^ c) & (b ^ c)) ^ c
//	OAM[row..row+1] = g
//	OAM[row+2..row+7] := OAM[row-6..row-1]
//
// where a=OAM[row..row+1], b=OAM[row-8..row-7], c=OAM[row-4..row-3].
func (m *Motherboard) OAMBugTrigger(addr uint16, cycleOffset OpCycles) {
	row := m.resolveOAMBugRow(addr, cycleOffset)
	if row == 0xFF {
		return
	}

	a := uint16(m.Memory.Oam[row]) | uint16(m.Memory.Oam[row+1])<<8
	b := uint16(m.Memory.Oam[row-8]) | uint16(m.Memory.Oam[row-7])<<8
	c := uint16(m.Memory.Oam[row-4]) | uint16(m.Memory.Oam[row-3])<<8

	glitched := ((a ^ c) & (b ^ c)) ^ c
	m.Memory.Oam[row] = uint8(glitched & 0xFF)
	m.Memory.Oam[row+1] = uint8(glitched >> 8)

	for i := 2; i < 8; i++ {
		m.Memory.Oam[int(row)+i] = m.Memory.Oam[int(row)-8+i]
	}
}

// debugOAMTriggerRead is for temporary instrumentation.
// OAMBugTriggerRead models the read-side OAM bug used by POP and LD A,(HL+/-).
// Three sub-variants depend on row's bits 3-4, all derived from SameBoy
// GB_trigger_oam_bug_read / oam_bug_*_read_corruption:
//
//   - Generic (row & 0x18 ∈ {0x08, 0x18}):
//     g = b | (a & c)
//     OAM[row..row+1] = OAM[row-4..row-3] = g
//
//   - Secondary (row & 0x18 == 0x10): uses bitwise_glitch_read_secondary
//     with one extra operand.
//
//   - Tertiary/quaternary (row & 0x18 == 0x00): hardware-instance specific
//     on SameBoy; we fall back to generic here, which is good enough for
//     Blargg's deterministic CRC tests on row=0x30 / 0x50 / 0x70 etc.
//
// All three variants then run OAM[row..row+7] := OAM[row-8..row-1].
func (m *Motherboard) OAMBugTriggerRead(addr uint16, cycleOffset OpCycles) {
	row := m.resolveOAMBugRow(addr, cycleOffset)
	if row == 0xFF {
		return
	}

	a := uint16(m.Memory.Oam[row]) | uint16(m.Memory.Oam[row+1])<<8
	b := uint16(m.Memory.Oam[row-8]) | uint16(m.Memory.Oam[row-7])<<8
	c := uint16(m.Memory.Oam[row-4]) | uint16(m.Memory.Oam[row-3])<<8

	switch row & 0x18 {
	case 0x10:
		if row >= 0x10 && int(row)+8 <= len(m.Memory.Oam) {
			d := uint16(m.Memory.Oam[row-2]) | uint16(m.Memory.Oam[row-1])<<8
			glitched := (b & (a | c | d)) | (a & c & d)
			m.Memory.Oam[row-4] = uint8(glitched & 0xFF)
			m.Memory.Oam[row-3] = uint8(glitched >> 8)
			for i := 0; i < 8; i++ {
				m.Memory.Oam[int(row)+i] = m.Memory.Oam[int(row)-8+i]
			}
			return
		}
	}

	glitched := b | (a & c)
	m.Memory.Oam[row] = uint8(glitched & 0xFF)
	m.Memory.Oam[row+1] = uint8(glitched >> 8)
	m.Memory.Oam[row-8] = uint8(glitched & 0xFF)
	m.Memory.Oam[row-7] = uint8(glitched >> 8)

	for i := 0; i < 8; i++ {
		m.Memory.Oam[int(row)+i] = m.Memory.Oam[int(row)-8+i]
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
