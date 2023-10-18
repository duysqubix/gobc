package motherboard

import (
	"bytes"
	"encoding/binary"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
)

const (
	sampleRate           = 44100
	maxFrameBufferLength = 5000
)

func init() {
	speaker.Init(beep.SampleRate(sampleRate), maxFrameBufferLength)
}

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

	audioBuffer chan [2]byte
}

func NewAPU(mb *Motherboard) *APU {

	apu := &APU{
		mb:    mb,
		NR50:  0x77,
		NR51:  0xF3,
		NR52:  0xF1,
		Chan1: soundChannel{0x80, 0xBF, 0xF3, 0xFF, 0xBF},
		Chan2: soundChannel{0x00, 0x3F, 0x00, 0xFF, 0xBF},
		Chan3: soundChannel{0x7F, 0xFF, 0x9F, 0xFF, 0xBF},
		Chan4: soundChannel{0x00, 0xFF, 0x00, 0x00, 0xBF},
	}
	apu.audioBuffer = make(chan [2]byte, maxFrameBufferLength)
	return apu

}

func (a *APU) SetItem(addr uint16, value uint8) {

	a.mb.Memory.SetIO(addr, value) // update internal memory

	// now actually do something with values, in terms of emulation and sound

	switch addr {

	}

}

func (a *APU) GetItem(addr uint16) uint8 {
	if addr == 0xFF26 {
		return 0x80
	}

	return a.mb.Memory.GetIO(addr)
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
