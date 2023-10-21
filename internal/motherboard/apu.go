package motherboard

import (
	"bytes"
	"math/rand"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
)

const (
	sampleRate           = 44100
	maxFrameBufferLength = 5000
)

var (
	NoiseBuffer *beep.Buffer
)

func init() {
	speaker.Init(beep.SampleRate(sampleRate), maxFrameBufferLength)
	format := beep.Format{SampleRate: sampleRate, NumChannels: 1, Precision: 1}
	NoiseBuffer = beep.NewBuffer(format)
	NoiseBuffer.Append(beep.Take(1000, Noise{}))

}

type Noise struct{}

func (no Noise) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		samples[i][0] = rand.Float64()*2 - 1
		samples[i][1] = rand.Float64()*2 - 1
	}
	return len(samples), true
}

func (no Noise) Err() error {
	return nil
}

type APU struct {
	mb          *Motherboard
	audioBuffer *beep.Buffer
}

func NewAPU(mb *Motherboard) *APU {

	audioFormat := beep.Format{
		SampleRate:  beep.SampleRate(sampleRate),
		NumChannels: 1,
		Precision:   3,
	}

	apu := &APU{
		mb:          mb,
		audioBuffer: beep.NewBuffer(audioFormat),
	}
	return apu

}

func (a *APU) Tick(cycles OpCycles) {

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
	return buf
}

func (a *APU) Deserialize(data *bytes.Buffer) error {
	return nil
}
