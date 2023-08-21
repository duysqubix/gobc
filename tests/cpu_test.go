package tests

import (
	"bytes"
	"testing"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
	mb "github.com/duysqubix/gobc/internal/motherboard"
)

var m = mb.NewMotherboard(&mb.MotherboardParams{})
var logger = internal.Logger

func TestNewCpu(t *testing.T) {
	cpu := mb.NewCpu(m)
	if cpu.Registers.A != 0 || cpu.Registers.B != 0 || cpu.Registers.C != 0 || cpu.Registers.D != 0 || cpu.Registers.E != 0 || cpu.Registers.F != 0 || cpu.Registers.H != 0 || cpu.Registers.L != 0 || cpu.Registers.SP != 0 || cpu.Registers.PC != 0 {
		t.Errorf("NewCpu(m) failed, expected all registers to be 0, got non-zero values")
	}
}

func TestSetBC(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetBC(0x1234)
	if cpu.Registers.B != 0x12 || cpu.Registers.C != 0x34 {
		t.Errorf("SetBC() failed, expected B=0x12 and C=0x34, got B=%x and C=%x", cpu.Registers.B, cpu.Registers.C)
	}
}

func TestSetDE(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetDE(0x1234)
	if cpu.Registers.D != 0x12 || cpu.Registers.E != 0x34 {
		t.Errorf("SetDE() failed, expected D=0x12 and E=0x34, got D=%x and E=%x", cpu.Registers.D, cpu.Registers.E)
	}
}

func TestBC(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetBC(0x1234)
	if cpu.BC() != 0x1234 {
		t.Errorf("BC() failed, expected 0x1234, got %x", cpu.BC())
	}
}

func TestDE(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetDE(0x1234)
	if cpu.DE() != 0x1234 {
		t.Errorf("DE() failed, expected 0x1234, got %x", cpu.DE())
	}
}

func TestHL(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.Registers.H = 0x12
	cpu.Registers.L = 0x34
	if cpu.HL() != 0x1234 {
		t.Errorf("HL() failed, expected 0x1234, got %x", cpu.HL())
	}
}

func TestIsFlagZSet(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.Registers.F = 0x80
	if !cpu.IsFlagZSet() {
		t.Errorf("IsFlagZSet() failed, expected true, got false")
	}
}

func TestIsFlagNSet(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.Registers.F = 0x40
	if !cpu.IsFlagNSet() {
		t.Errorf("IsFlagNSet() failed, expected true, got false")
	}
}

func TestIsFlagHSet(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.Registers.F = 0x20
	if !cpu.IsFlagHSet() {
		t.Errorf("IsFlagHSet() failed, expected true, got false")
	}
}

func TestIsFlagCSet(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.Registers.F = 0x10
	if !cpu.IsFlagCSet() {
		t.Errorf("IsFlagCSet() failed, expected true, got false")
	}
}

func TestSetFlagZ(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagZ()
	if !cpu.IsFlagZSet() {
		t.Errorf("SetFlagZ() failed, expected true, got false")
	}
}

func TestSetFlagH(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagH()
	if !cpu.IsFlagHSet() {
		t.Errorf("SetFlagH() failed, expected true, got false")
	}
}

func TestSetFlagC(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagC()
	if !cpu.IsFlagCSet() {
		t.Errorf("SetFlagC() failed, expected true, got false")
	}
}

func TestSetFlagN(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagN()
	if !cpu.IsFlagNSet() {
		t.Errorf("SetFlagN() failed, expected true, got false")
	}
}

func TestResetFlagZ(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagZ()
	cpu.ResetFlagZ()
	if cpu.IsFlagZSet() {
		t.Errorf("ResetFlagZ() failed, expected false, got true")
	}
}

func TestResetFlagH(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagH()
	cpu.ResetFlagH()
	if cpu.IsFlagHSet() {
		t.Errorf("ResetFlagH() failed, expected false, got true")
	}
}

func TestResetFlagN(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagN()
	cpu.ResetFlagN()
	if cpu.IsFlagNSet() {
		t.Errorf("ResetFlagN() failed, expected false, got true")
	}
}

func TestResetFlagC(t *testing.T) {
	cpu := mb.NewCpu(m)
	cpu.SetFlagC()
	cpu.ResetFlagC()
	if cpu.IsFlagCSet() {
		t.Errorf("ResetFlagC() failed, expected false, got true")
	}
}

func TestInterrupts(t *testing.T) {
	i := &mb.Interrupts{
		Master_Enable: false,
		IF:            0,
		IE:            0,
	}

	// Test setting bits
	i.SetVBlank()
	if !i.IsVBlankSet() {
		t.Errorf("Expected VBlank bit to be set")
	}

	i.SetLCDStat()
	if !i.IsLCDStatSet() {
		t.Errorf("Expected LCDStat bit to be set")
	}

	i.SetTimer()
	if !i.IsTimerSet() {
		t.Errorf("Expected Timer bit to be set")
	}

	i.SetSerial()
	if !i.IsSerialSet() {
		t.Errorf("Expected Serial bit to be set")
	}

	i.SetHighToLow()
	if !i.IsHighToLowSet() {
		t.Errorf("Expected HighToLow bit to be set")
	}

	// Test resetting bits
	i.ResetVBlank()
	if i.IsVBlankSet() {
		t.Errorf("Expected VBlank bit to be reset")
	}

	i.ResetLCDStat()
	if i.IsLCDStatSet() {
		t.Errorf("Expected LCDStat bit to be reset")
	}

	i.ResetTimer()
	if i.IsTimerSet() {
		t.Errorf("Expected Timer bit to be reset")
	}

	i.ResetSerial()
	if i.IsSerialSet() {
		t.Errorf("Expected Serial bit to be reset")
	}

	i.ResetHighToLow()
	if i.IsHighToLowSet() {
		t.Errorf("Expected HighToLow bit to be reset")
	}
}

func TestCheckValidInterrupts(t *testing.T) {
	i := &mb.Interrupts{
		IE: 0b10101,
		IF: 0b01101,
	}

	expected := uint8(0b00101)
	result := i.CheckValidInterrupts()

	if result != expected {
		t.Errorf("Expected %08b but got %08b", expected, result)
	}
}

func TestSetInterruptFlag(t *testing.T) {
	i := &mb.Interrupts{
		IF: 0,
	}

	i.SetInterruptFlag(mb.INTR_VBLANK)

	if !internal.IsBitSet(i.IF, mb.INTR_VBLANK) {
		t.Errorf("Expected VBlank bit to be set in IF")
	}
}

func TestSetInterruptEnable(t *testing.T) {
	i := &mb.Interrupts{
		IE: 0,
	}

	i.SetInterruptEnable(mb.INTR_VBLANK)

	if !internal.IsBitSet(i.IE, mb.INTR_VBLANK) {
		t.Errorf("Expected VBlank bit to be set in IE")
	}
}

func TestCheckForInterrupts_NoInterrupts(t *testing.T) {
	c := &mb.CPU{
		Interrupts: &mb.Interrupts{
			IE: 0,
			IF: 0,
		},
	}

	result := c.CheckForInterrupts()

	if result != false {
		t.Errorf("Expected false but got %v", result)
	}
}

func TestCheckForInterrupts_InterruptQueued(t *testing.T) {
	c := &mb.CPU{
		Interrupts: &mb.Interrupts{
			Queued: true,
		},
		Mb: m,
	}

	result := c.CheckForInterrupts()

	if result != false {
		t.Errorf("Expected false but got %v", result)
	}
}

func TestCheckForInterrupts_ValidInterrupts(t *testing.T) {

	c := mb.NewCpu(m)
	c.Interrupts = &mb.Interrupts{
		IE:     0b10101,
		IF:     0b01101,
		Queued: false,
	}

	c.CheckForInterrupts()

	if !c.Interrupts.Queued {
		t.Errorf("Expected interrupt to be queued")
	}
}

// test that the correct number of cycles are returned
// and registers are updated correctly
func TestExecuteInstruction8bitImmediate(t *testing.T) {
	// create cpu instructions that perform an 8bit immediate, e.g. LD A, 0x12
	// and test that the correct number of cycles are returned
	// and registers are updated correctly

	rom_bank := bytes.Repeat([]byte{0xff}, int(cartridge.MEMORY_BANK_SIZE))

	rom_bank[0x150] = 0x00 // NOP  4 cycles
	rom_bank[0x151] = 0x00 // NOP  4 cycles
	rom_bank[0x152] = 0x3e // LD A 8 cycles
	rom_bank[0x153] = 0x12 // 0x12
	rom_bank[0x154] = 0x76 // halt 4 cycles

	rom_banks := cartridge.LoadRomBanks(rom_bank)

	dummy_cart := &cartridge.Cartridge{RomBanks: rom_banks, RomBanksCount: 1, RomBankSelected: 0}
	dummy_cart.CartType = cartridge.CARTRIDGE_TABLE[0x00](dummy_cart) // ROM_ONLY

	_m := mb.Motherboard{
		Cartridge: dummy_cart,
	}
	_m.Cpu = mb.NewCpu(&_m)

	c := _m.Cpu

	c.Registers.PC = 0x150

	var expected_cycles mb.OpCycles = 20
	var cycles mb.OpCycles = 0
	for i := 0; i < 4; i++ {
		opcode := _m.GetItem(&c.Registers.PC)
		opcode_str := mb.OPCODE_NAMES[opcode]
		cycles += c.ExecuteInstruction()
		logger.Infof("Executing %s [%#x]...Cycles: %d", opcode_str, opcode, cycles)

	}

	if cycles != expected_cycles {
		t.Errorf("Expected %d cycles, got %d", expected_cycles, cycles)
	}

	if c.Registers.A != 0x12 {
		t.Errorf("Expected A to be 0x12, got %#x", c.Registers.A)
	}

}

// test that the correct number of cycles are returned
// and registers are updated correctly
func TestExecuteInstruction16bitImmediate(t *testing.T) {
	// create cpu instructions that perform an 16bit immediate, e.g. LD DE, 0xbeef
	// and test that the correct number of cycles are returned
	// and registers are updated correctly

	rom_bank := bytes.Repeat([]byte{0xff}, int(cartridge.MEMORY_BANK_SIZE))

	// LD A, 0x12 (Total Cycles: 24)
	rom_bank[0x150] = 0x00 // NOP   4 cycles
	rom_bank[0x151] = 0x00 // NOP   4 cycles
	rom_bank[0x152] = 0x11 // LD DE 12 cycles
	rom_bank[0x153] = 0xef // DE_lo  8 cycles
	rom_bank[0x154] = 0xbe // DE_hi  8 cycles
	rom_bank[0x155] = 0x76 // halt   4 cycles

	rom_banks := cartridge.LoadRomBanks(rom_bank)

	dummy_cart := &cartridge.Cartridge{RomBanks: rom_banks, RomBanksCount: 1, RomBankSelected: 0}
	dummy_cart.CartType = cartridge.CARTRIDGE_TABLE[0x00](dummy_cart) // ROM_ONLY

	_m := mb.Motherboard{
		Cartridge: dummy_cart,
	}
	_m.Cpu = mb.NewCpu(&_m)

	c := _m.Cpu

	c.Registers.PC = 0x150

	var expected_cycles mb.OpCycles = 24
	var cycles mb.OpCycles = 0

	// make sure to only iterate through the number of commands
	// NOP, NOP, (LD DE, $12), HALT == 4 commands
	for i := 0; i < 4; i++ {
		opcode := _m.GetItem(&c.Registers.PC)
		opcode_str := mb.OPCODE_NAMES[opcode]
		cycles += c.ExecuteInstruction()
		logger.Infof("(%d) Executing %s [%#x]...Cycles: %d | PC: $%X", i, opcode_str, opcode, cycles, c.Registers.PC)

	}

	if cycles != expected_cycles {
		t.Errorf("Expected %d cycles, got %d", expected_cycles, cycles)
	}

	if c.DE() != 0xbeef {
		t.Errorf("Expected DE to be 0xbeef, got %#x", c.DE())
	}
}

// test that the correct number of cycles are returned
// and registers are updated correctly
func TestExecuteInstructionCB(t *testing.T) {
	// create cpu instructions that perform a CB instruction, like SET 2, B (0xcb 0xc4)
	// and test that the correct number of cycles are returned

	rom_bank := bytes.Repeat([]byte{0xff}, int(cartridge.MEMORY_BANK_SIZE))

	rom_bank[0x150] = 0x00 // NOP   4 cycles
	rom_bank[0x151] = 0x00 // NOP   4 cycles
	rom_bank[0x152] = 0xcb // SET 2, 8 cycles
	rom_bank[0x153] = 0xd0 //
	rom_bank[0x154] = 0x76 // halt   4 cycles

	rom_banks := cartridge.LoadRomBanks(rom_bank)

	dummy_cart := &cartridge.Cartridge{RomBanks: rom_banks, RomBanksCount: 1, RomBankSelected: 0}
	dummy_cart.CartType = cartridge.CARTRIDGE_TABLE[0x00](dummy_cart) // ROM_ONLY

	_m := mb.Motherboard{
		Cartridge: dummy_cart,
	}
	_m.Cpu = mb.NewCpu(&_m)

	c := _m.Cpu

	c.Registers.PC = 0x150
	c.Registers.B = 0x00

	var expected_cycles mb.OpCycles = 20
	var cycles mb.OpCycles = 0

	// make sure to only iterate through the number of commands
	// NOP, NOP, (LD DE, $12), HALT == 4 commands
	for i := 0; i < 4; i++ {
		// opcode := _m.GetItem(&c.Registers.PC)
		// opcode_str := mb.OPCODE_NAMES[opcode]
		cycles += c.ExecuteInstruction()
		// logger.Infof("(%d) Executing %s [%#x]...Cycles: %d | PC: $%X", i, opcode_str, opcode, cycles, c.Registers.PC)

	}

	if cycles != expected_cycles {
		t.Errorf("Expected %d cycles, got %d", expected_cycles, cycles)
	}

	if !internal.IsBitSet(c.Registers.B, 2) {
		t.Errorf("Expected bit 2 to be set in B, got %#x", c.Registers.B)
	}
}
