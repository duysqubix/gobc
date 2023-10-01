package motherboard

import "github.com/duysqubix/gobc/internal"

func (m *Motherboard) SetItem(addr uint16, value uint16) {
	if value >= 0x100 {
		logger.Fatalf("Memory write error! Can't write %#x to %#x\n", value, addr)
	}

	if m.Decouple {
		logger.Warn("Decoupled Motherboard from other components. Memory write is mocked")
		return
	}
	v := uint8(value)

	switch {
	/*
	*
	* WRITE: ROM BANK 0
	*
	 */
	case addr < 0x4000:
		if m.BootRomEnabled() && (addr < 0x100 || (m.Cgb && 0x200 <= addr && addr < 0x900)) {
			logger.Errorf("Can't write to ROM bank 0 when boot ROM is enabled")
			return
		}

		m.Cartridge.CartType.SetItem(addr, v)

	/*
	*
	* WRITE: SWITCHABLE ROM BANK
	*
	 */
	case 0x4000 <= addr && addr < 0x8000:
		m.Cartridge.CartType.SetItem(addr, v)

	/*
	*
	* WRITE: VIDEO RAM
	*
	 */
	case 0x8000 <= addr && addr < 0xA000:
		if m.Cgb {
			bank := m.Memory.GetIO(IO_VBK) & 0x01

			m.Memory.SetVram(bank, addr, v)
			return
		}
		m.Memory.SetVram(0, addr, v)

	/*
	*
	* WRITE: EXTERNAL RAM
	*
	 */
	case 0xA000 <= addr && addr < 0xC000:
		m.Cartridge.CartType.SetItem(addr, v)

	/*
	*
	* WRITE: WORK RAM BANK 0
	*
	 */
	case 0xC000 <= addr && addr < 0xD000:
		m.Memory.Wram[0][addr-0xC000] = v

	/*
	*
	* WRITE: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= addr && addr < 0xE000:

		// check if CGB mode
		if m.Cgb {
			// check what bank to read from
			bank := m.Memory.ActiveWramBank()
			m.Memory.Wram[bank][addr-0xD000] = v
			return
		}
		m.Memory.Wram[1][addr-0xD000] = v

	/*
	*
	* WRITE: ECHO OF 8K INTERNAL RAM
	*
	 */
	case 0xE000 <= addr && addr < 0xF000:
		m.Memory.Wram[0][addr-0x2000-0xC000] = v

	case 0xF000 <= addr && addr < 0xFE00:
		// fmt.Printf("ECHO OF 8K INTERNAL RAM: %#x, %d\n", addr, addr-0x2000-0xD000)
		m.Memory.Wram[1][addr-0x2000-0xD000] = v

	/*
	*
	* WRITE: SPRITE ATTRIBUTE TABLE (OAM)
	*
	 */
	case 0xFE00 <= addr && addr < 0xFEA0:
		m.Memory.Oam[addr-0xFE00] = v
	/*
	*
	* WRITE: NOT USABLE
	*
	 */
	case 0xFEA0 <= addr && addr < 0xFF00:

	/*
	*
	* WRITE: I/O REGISTERS
	*
	 */
	case 0xFF00 <= addr && addr < 0xFF80:

		switch addr {
		case 0xFF00: /* P1 */
			m.Memory.SetIO(IO_P1_JOYP, m.Input.Pull(v))

		case 0xFF04: /* DIV */
			m.Timer.TimaCounter = 0
			m.Timer.DivCounter = 0
			m.Timer.DIV = 0
			return

		case 0xFF05: /* TIMA */
			m.Timer.TIMA = uint32(v)
			return

		case 0xFF06: /* TMA */
			m.Timer.TMA = uint32(v)
			return

		case 0xFF07: /* TAC */
			currentFreq := m.Timer.TAC & 0x03
			m.Timer.TAC = uint32(v) | 0xF8
			newFreq := m.Timer.TAC & 0x03
			if currentFreq != newFreq {
				m.Timer.TimaCounter = 0
			}
			return

		case 0xFF0F: /* IF */
			m.Cpu.Interrupts.IF = v
			return

		case 0xFF41: /* STAT */
			// do not set bits 0-1, they are read_only bits, bit 7 always reads 1
			m.Memory.SetIO(IO_STAT, (m.Memory.GetIO(IO_STAT)&0x83)|(v&0xFC))

		case 0xFF44: /* LY */
			m.Memory.SetIO(IO_LY, 0)

		case 0xFF46: /* DMA */
			m.doDMATransfer(v)

		case 0xFF4D: /* KEY1 */
			//TODO: implement double speed mode

		case 0xFF4F: /* VBK */
			if m.Cgb && !m.HdmaActive {
				m.Memory.SetIO(IO_VBK, v&0x01)
			}

		case 0xFF50: /* Disable Boot ROM */
			if !m.BootRomEnabled() {
				logger.Warnf("Writing to 0xFF50 when boot ROM is disabled")
			}

			if m.BootRomEnabled() {
				logger.Debugf("CGB: %t, Value: %#x", m.Cgb, v)
				if m.Cgb && v == 0x11 || !m.Cgb && v == 0x1 {

					logger.Warnf("Disabling boot rom")
					m.BootRom.Disable()
					m.Cpu.Registers.PC = ROM_START_ADDR - 2 // PC will be incremented by 2
				}
			}
			return

		case 0xFF55: /* HDMA5 */
			if m.Cgb {
				m.doNewDMATransfer(v)
			}

		case 0xFF68: /* BG Palette Index */
			if m.Cgb {
				m.BGPalette.updateIndex(v)
			}
			return

		case 0xFF69: /* BG Palette Data */
			if m.Cgb {
				m.BGPalette.write(v)
			}

			return

		case 0xFF6A: /* Sprite Palette Index */
			if m.Cgb {
				m.SpritePalette.updateIndex(v)
			}
			return

		case 0xFF6B: /* Sprite Palette Data */
			if m.Cgb {
				m.SpritePalette.write(v)
			}
			return

		case 0xFF70: /* WRAM Bank */
			if m.Cgb {
				m.Memory.SetIO(IO_SVBK, v&0x07)
			}

		default:
			m.Memory.SetIO(addr, v)
		}

		/// prints serial output to terminal ///
		// if v == 0x81 && addr == IO_SC {
		// }
		////////////////////////////////////

	/*
	*
	* WRITE: HIGH RAM
	*
	 */
	case 0xFF80 <= addr && addr < 0xFFFF:
		m.Memory.Hram[addr-0xFF80] = v

	/*
	*
	* WRITE: INTERRUPT ENABLE REGISTER
	*
	 */
	case addr == IE:
		m.Cpu.Interrupts.IE = v
	default:
		internal.Logger.Panicf("Memory write error! Can't write `%#x` to `%#x`\n", value, addr)
	}

}
