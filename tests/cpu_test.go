package tests

import (
	"fmt"
	"testing"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cpu"
)

type MockMotherboard struct{}

func (m MockMotherboard) SetItem(addr *uint16, value *uint16) {
	fmt.Println("Addr: ", addr, "Value: ", value)
}
func (m MockMotherboard) GetItem(addr *uint16) uint8 { return 0 }

var m = MockMotherboard{}

func TestNewCpu(t *testing.T) {
	cpu := cpu.NewCpu(m)
	if cpu.Registers.A != 0 || cpu.Registers.B != 0 || cpu.Registers.C != 0 || cpu.Registers.D != 0 || cpu.Registers.E != 0 || cpu.Registers.F != 0 || cpu.Registers.H != 0 || cpu.Registers.L != 0 || cpu.Registers.SP != 0 || cpu.Registers.PC != 0 {
		t.Errorf("NewCpu(m) failed, expected all registers to be 0, got non-zero values")
	}
}

func TestSetBC(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetBC(0x1234)
	if cpu.Registers.B != 0x12 || cpu.Registers.C != 0x34 {
		t.Errorf("SetBC() failed, expected B=0x12 and C=0x34, got B=%x and C=%x", cpu.Registers.B, cpu.Registers.C)
	}
}

func TestSetDE(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetDE(0x1234)
	if cpu.Registers.D != 0x12 || cpu.Registers.E != 0x34 {
		t.Errorf("SetDE() failed, expected D=0x12 and E=0x34, got D=%x and E=%x", cpu.Registers.D, cpu.Registers.E)
	}
}

func TestBC(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetBC(0x1234)
	if cpu.BC() != 0x1234 {
		t.Errorf("BC() failed, expected 0x1234, got %x", cpu.BC())
	}
}

func TestDE(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetDE(0x1234)
	if cpu.DE() != 0x1234 {
		t.Errorf("DE() failed, expected 0x1234, got %x", cpu.DE())
	}
}

func TestHL(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.Registers.H = 0x12
	cpu.Registers.L = 0x34
	if cpu.HL() != 0x1234 {
		t.Errorf("HL() failed, expected 0x1234, got %x", cpu.HL())
	}
}

func TestIsFlagZSet(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.Registers.F = 0x80
	if !cpu.IsFlagZSet() {
		t.Errorf("IsFlagZSet() failed, expected true, got false")
	}
}

func TestIsFlagNSet(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.Registers.F = 0x40
	if !cpu.IsFlagNSet() {
		t.Errorf("IsFlagNSet() failed, expected true, got false")
	}
}

func TestIsFlagHSet(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.Registers.F = 0x20
	if !cpu.IsFlagHSet() {
		t.Errorf("IsFlagHSet() failed, expected true, got false")
	}
}

func TestIsFlagCSet(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.Registers.F = 0x10
	if !cpu.IsFlagCSet() {
		t.Errorf("IsFlagCSet() failed, expected true, got false")
	}
}

func TestSetFlagZ(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagZ()
	if !cpu.IsFlagZSet() {
		t.Errorf("SetFlagZ() failed, expected true, got false")
	}
}

func TestSetFlagH(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagH()
	if !cpu.IsFlagHSet() {
		t.Errorf("SetFlagH() failed, expected true, got false")
	}
}

func TestSetFlagC(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagC()
	if !cpu.IsFlagCSet() {
		t.Errorf("SetFlagC() failed, expected true, got false")
	}
}

func TestSetFlagN(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagN()
	if !cpu.IsFlagNSet() {
		t.Errorf("SetFlagN() failed, expected true, got false")
	}
}

func TestResetFlagZ(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagZ()
	cpu.ResetFlagZ()
	if cpu.IsFlagZSet() {
		t.Errorf("ResetFlagZ() failed, expected false, got true")
	}
}

func TestResetFlagH(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagH()
	cpu.ResetFlagH()
	if cpu.IsFlagHSet() {
		t.Errorf("ResetFlagH() failed, expected false, got true")
	}
}

func TestResetFlagN(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagN()
	cpu.ResetFlagN()
	if cpu.IsFlagNSet() {
		t.Errorf("ResetFlagN() failed, expected false, got true")
	}
}

func TestResetFlagC(t *testing.T) {
	cpu := cpu.NewCpu(m)
	cpu.SetFlagC()
	cpu.ResetFlagC()
	if cpu.IsFlagCSet() {
		t.Errorf("ResetFlagC() failed, expected false, got true")
	}
}

func TestInterrupts(t *testing.T) {
	i := &cpu.Interrupts{
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
	i := &cpu.Interrupts{
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
	i := &cpu.Interrupts{
		IF: 0,
	}

	i.SetInterruptFlag(cpu.INTR_VBLANK)

	if !internal.IsBitSet(i.IF, cpu.INTR_VBLANK) {
		t.Errorf("Expected VBlank bit to be set in IF")
	}
}

func TestSetInterruptEnable(t *testing.T) {
	i := &cpu.Interrupts{
		IE: 0,
	}

	i.SetInterruptEnable(cpu.INTR_VBLANK)

	if !internal.IsBitSet(i.IE, cpu.INTR_VBLANK) {
		t.Errorf("Expected VBlank bit to be set in IE")
	}
}

func TestCheckForInterrupts_NoInterrupts(t *testing.T) {
	c := &cpu.Cpu{
		Interrupts: &cpu.Interrupts{
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
	c := &cpu.Cpu{
		Interrupts: &cpu.Interrupts{
			Queued: true,
		},
		Mb: &m,
	}

	result := c.CheckForInterrupts()

	if result != false {
		t.Errorf("Expected false but got %v", result)
	}
}

func TestCheckForInterrupts_ValidInterrupts(t *testing.T) {

	c := cpu.NewCpu(&m)
	c.Interrupts = &cpu.Interrupts{
		IE:     0b10101,
		IF:     0b01101,
		Queued: false,
	}

	c.CheckForInterrupts()

	if !c.Interrupts.Queued {
		t.Errorf("Expected interrupt to be queued")
	}
}
