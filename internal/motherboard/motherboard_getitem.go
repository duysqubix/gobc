package motherboard

func (m *Motherboard) GetItem(addr uint16) uint8 {

	// debugging
	switch {
	/*
	*
	* READ: ROM BANK 0
	*
	 */
	case addr < 0x4000: // ROM bank 0
		if m.BootRomEnabled() && (addr < 0x100 || (m.Cgb && 0x200 <= addr && addr < 0x900)) {
			return m.BootRom.GetItem(addr)
		} else {
			return m.Cartridge.CartType.GetItem(addr)
		}

	/*
	*
	* READ: SWITCHABLE ROM BANK
	*
	 */
	case 0x4000 <= addr && addr < 0x8000: // Switchable ROM bank
		return m.Cartridge.CartType.GetItem(addr)

	/*
	*
	* READ: VIDEO RAM
	*
	 */
	case 0x8000 <= addr && addr < 0xA000: // 8K Video RAM
		if m.Cgb {
			activeBank := m.Memory.ActiveVramBank()
			// return m.Memory.GetItemVRAM(activeBank, addr-0x8000)
			return m.Memory.Vram[activeBank][addr-0x8000]
		}

		return m.Memory.Vram[0][addr-0x8000]

	/*
	*
	* READ: EXTERNAL RAM
	*
	 */
	case 0xA000 <= addr && addr < 0xC000: // 8K External RAM (Cartridge)
		return m.Cartridge.CartType.GetItem(addr)

	/*
	*
	* READ: WORK RAM BANK 0
	*
	 */
	case 0xC000 <= addr && addr < 0xD000: // 4K Work RAM bank 0
		return m.Memory.Wram[0][addr-0xC000]

	/*
	*
	* READ: WORK 4K RAM BANK 1 (or switchable bank 1)
	*
	 */
	case 0xD000 <= addr && addr < 0xE000:
		if m.Cgb {
			bank := m.Memory.ActiveWramBank()
			return m.Memory.Wram[bank][addr-0xD000]
		}
		return m.Memory.Wram[1][addr-0xD000]

	/*
	*
	* READ: ECHO OF 8K INTERNAL RAM
	*
	 */
	case 0xE000 <= addr && addr < 0xFE00:
		addr = addr - 0x2000 - 0xC000
		if addr >= 0x1000 {
			addr -= 0x1000
			if m.Cgb {
				bank := m.Memory.ActiveWramBank()
				return m.Memory.Wram[bank][addr]
			}
			return m.Memory.Wram[1][addr]
		}
		return m.Memory.Wram[0][addr]

	/*
	*
	* READ: SPRITE ATTRIBUTE TABLE (OAM)
	*
	 */
	case 0xFE00 <= addr && addr < 0xFEA0:
		return m.Memory.Oam[addr-0xFE00]

	/*
	*
	* READ: NOT USABLE
	*
	 */
	case 0xFEA0 <= addr && addr < 0xFF00:

	/*
	*
	* READ: I/O REGISTERS
	*
	 */
	case 0xFF00 <= addr && addr < 0xFF80:

		switch addr {

		case 0xFF00: /* P1 */
			return m.Memory.IO[IO_P1_JOYP-IO_START_ADDR]

		case 0xFF04: /* DIV */
			return uint8(m.Timer.DIV)

		case 0xFF05: /* TIMA */
			return uint8(m.Timer.TIMA)

		case 0xFF06: /* TMA */
			return uint8(m.Timer.TMA)

		case 0xFF07: /* TAC */
			return uint8(m.Timer.TAC)

		case 0xFF0F: /* IF */
			return m.Cpu.Interrupts.IF | 0xE0

		case 0xFF40: /* LCDC */
			return m.Memory.IO[IO_LCDC-IO_START_ADDR]

		case 0xFF41: /* STAT */
			return m.Memory.IO[IO_STAT-IO_START_ADDR]

		case 0xFF44: /* LY */
			return m.Memory.IO[IO_LY-IO_START_ADDR]

		case 0xFF46: /* DMA */
			return 0x00

		case 0xFF4D: /* KEY1 */
			// TODO: implement double speed mode
			return 0xFF
		case 0xFF4F: /* VBK */
			if m.Cgb {
				return m.Memory.IO[IO_VBK-IO_START_ADDR] | 0xFE
			}
			return 0xFF

		case 0xFF50: /* Disable Boot ROM */
			return 0xFF

		case 0xFF55: /* HDMA5 */
			if m.Cgb {
				return m.Memory.IO[IO_HDMA5-IO_START_ADDR]
			}

		case 0xFF68: /* BG Palette Index */
			if m.Cgb {
				return m.BGPalette.readIndex()
			}
			return 0x00

		case 0xFF69: /* BG Palette Data */
			if m.Cgb {
				return m.BGPalette.read()
			}
			return 0x00

		case 0xFF6A: /* Sprite Palette Index */
			if m.Cgb {
				return m.SpritePalette.readIndex()
			}
			return 0x00

		case 0xFF6B: /* Sprite Palette Data */
			if m.Cgb {
				return m.SpritePalette.read()
			}
			return 0x00

		default:
			return m.Memory.IO[addr-IO_START_ADDR]
		}

	/*
	*
	* READ: HIGH RAM
	*
	 */
	case 0xFF80 <= addr && addr < 0xFFFF:
		return m.Memory.Hram[addr-0xFF80]

	/*
	*
	* READ: INTERRUPT ENABLE REGISTER
	*
	 */
	case addr == 0xFFFF:
		return m.Cpu.Interrupts.IE

	default:
		logger.Panicf("Memory read error! Can't read from %#x\n", addr)
	}

	return 0xFF
}
