package motherboard

import (
	"bytes"
	"encoding/binary"
)

type soundChannel [5]uint8

func (s *soundChannel) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, s)
	return buf
}

func (s *soundChannel) Deserialize(data *bytes.Buffer) error {
	// Read the data from the buffer
	if err := binary.Read(data, binary.LittleEndian, s); err != nil {
		return err
	}
	return nil
}

type APU struct {
	mb *Motherboard
	// master Registers
	NR50    uint8
	NR51    uint8
	NR52    uint8
	WaveRam [16]uint8
	// Sound channels
	Chan1 soundChannel
	Chan2 soundChannel
	Chan3 soundChannel
	Chan4 soundChannel
}

func NewAPU(mb *Motherboard) *APU {

	return &APU{
		mb:    mb,
		NR50:  0x77,
		NR51:  0xF3,
		NR52:  0xF1,
		Chan1: soundChannel{0x80, 0xBF, 0xF3, 0xFF, 0xBF},
		Chan2: soundChannel{0x00, 0x3F, 0x00, 0xFF, 0xBF},
		Chan3: soundChannel{0x7F, 0xFF, 0x9F, 0xFF, 0xBF},
		Chan4: soundChannel{0x00, 0xFF, 0x00, 0x00, 0xBF},
	}
}

func (a *APU) SetItem(addr uint16, value uint8) {
	switch addr {
	case 0xFF10:
		a.Chan1[0] = value
	case 0xFF11:
		a.Chan1[1] = value
	case 0xFF12:
		a.Chan1[2] = value
	case 0xFF13:
		a.Chan1[3] = value
	case 0xFF14:
		a.Chan1[4] = value
	case 0xFF16:
		a.Chan2[1] = value
	case 0xFF17:
		a.Chan2[2] = value
	case 0xFF18:
		a.Chan2[3] = value
	case 0xFF19:
		a.Chan2[4] = value
	case 0xFF1A:
		a.Chan3[0] = value
	case 0xFF1B:
		a.Chan3[1] = value
	case 0xFF1C:
		a.Chan3[2] = value
	case 0xFF1D:
		a.Chan3[3] = value
	case 0xFF1E:
		a.Chan3[4] = value
	case 0xFF20:
		a.Chan4[1] = value
	case 0xFF21:
		a.Chan4[2] = value
	case 0xFF22:
		a.Chan4[3] = value
	case 0xFF23:
		a.Chan4[4] = value
	case 0xFF24:
		a.NR50 = value
	case 0xFF25:
		a.NR51 = value
	case 0xFF26:
		a.NR52 = value
	case 0xFF30:
		a.WaveRam[0] = value
	case 0xFF31:
		a.WaveRam[1] = value
	case 0xFF32:
		a.WaveRam[2] = value
	default:
		logger.Fatalf("Can't write to %#x\n", addr)
	}
}

func (a *APU) GetItem(addr uint16) uint8 {
	switch addr {
	case 0xFF10:
		return a.Chan1[0]
	case 0xFF11:
		return a.Chan1[1]
	case 0xFF12:
		return a.Chan1[2]
	case 0xFF13:
		return a.Chan1[3]
	case 0xFF14:
		return a.Chan1[4]
	case 0xFF16:
		return a.Chan2[1]
	case 0xFF17:
		return a.Chan2[2]
	case 0xFF18:
		return a.Chan2[3]
	case 0xFF19:
		return a.Chan2[4]
	case 0xFF1A:
		return a.Chan3[0]
	case 0xFF1B:
		return a.Chan3[1]
	case 0xFF1C:
		return a.Chan3[2]
	case 0xFF1D:
		return a.Chan3[3]
	case 0xFF1E:
		return a.Chan3[4]
	case 0xFF20:
		return a.Chan4[1]
	case 0xFF21:
		return a.Chan4[2]
	case 0xFF22:
		return a.Chan4[3]
	case 0xFF23:
		return a.Chan4[4]
	case 0xFF24:
		return a.NR50
	case 0xFF25:
		return a.NR51
	case 0xFF26:
		return 0x80
		// return a.NR52
	case 0xFF30:
		return a.WaveRam[0]
	case 0xFF31:
		return a.WaveRam[1]
	case 0xFF32:
		return a.WaveRam[2]

	default:
		logger.Fatalf("Can't read from %#x\n", addr)
	}
	return 0
}

func (a *APU) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, a.NR50)    // NR50
	binary.Write(buf, binary.LittleEndian, a.NR51)    // NR51
	binary.Write(buf, binary.LittleEndian, a.NR52)    // NR52
	binary.Write(buf, binary.LittleEndian, a.WaveRam) // Wave RAM
	binary.Write(buf, binary.LittleEndian, a.Chan1)   // Channel 1
	binary.Write(buf, binary.LittleEndian, a.Chan2)   // Channel 2
	binary.Write(buf, binary.LittleEndian, a.Chan3)   // Channel 3
	binary.Write(buf, binary.LittleEndian, a.Chan4)   // Channel 4
	return buf
}

func (a *APU) Deserialize(data *bytes.Buffer) error {

	if err := binary.Read(data, binary.LittleEndian, &a.NR50); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &a.NR51); err != nil {
		return err

	}
	if err := binary.Read(data, binary.LittleEndian, &a.NR52); err != nil {
		return err

	}
	if err := binary.Read(data, binary.LittleEndian, &a.WaveRam); err != nil {
		return err

	}
	if err := binary.Read(data, binary.LittleEndian, &a.Chan1); err != nil {
		return err

	}
	if err := binary.Read(data, binary.LittleEndian, &a.Chan2); err != nil {
		return err

	}
	if err := binary.Read(data, binary.LittleEndian, &a.Chan3); err != nil {
		return err

	}
	if err := binary.Read(data, binary.LittleEndian, &a.Chan4); err != nil {
		return err

	}
	return nil
}
