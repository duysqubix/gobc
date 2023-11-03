package motherboard

import (
	"math"

	"github.com/gopxl/beep"
)

// WaveGenerator is a function which can be used for generating waveform
// samples for different channels.
// type WaveGenerator func(t float64) byte

// Square returns a square wave generator with a given mod. This is used
// for channels 1 and 2.
type genSquare struct {
	mod float64
	t   float64
}

func (g genSquare) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		samples[i][0] = 0.0
		samples[i][1] = 0.0

		if math.Sin(g.t) <= g.mod {
			samples[i][0] = 0xFF
			samples[i][1] = 0xFF

		}
	}
	return len(samples), true
}

func (g genSquare) Err() error {
	return nil
}

func Square(sampleRate beep.SampleRate, mod float64) (beep.Streamer, error) {
	dt := 1.0 / float64(sampleRate)
	return &genSquare{mod: mod, t: dt}, nil
}

// Waveform returns a wave generator for some waveform ram. This is used
// by channel 3.
// func GenWaveform(ram func(i int) byte) WaveGenerator {
// 	return func(t float64) byte {
// 		idx := int(math.Floor(t/twoPi*32)) % 0x20
// 		return ram(idx)
// 	}
// }

// Noise returns a wave generator for a noise channel. This is used by
// channel 4.
// func GenNoise() WaveGenerator {
// 	var last float64
// 	var val byte
// 	return func(t float64) byte {
// 		if t-last > twoPi {
// 			last = t
// 			val = byte(rand.Intn(2)) * 0xFF
// 		}
// 		return val
// 	}
// }
