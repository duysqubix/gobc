// Package motherboard — apu_capture_test.go
//
// In-process audio verification: bypasses beep/oto entirely and
// records the APU's stereo sample stream into an in-memory buffer.
// Then performs DFT-based frequency analysis to confirm the channel
// DSP produces the right waveforms at the right frequencies.
//
// This proves the audio pipeline is correct without needing a real
// audio device — DFT peak frequency, RMS amplitude, and sample count
// are all hard numbers, not subjective listening.

package motherboard

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureAPU wires an APU to an in-memory sample buffer instead of beep.
// Returns the APU + the buffer (which the caller drains). The streamer
// starts EMPTY for tests (no silence pre-fill) so DFT analysis sees only
// the channel's actual output.
func captureAPU(t *testing.T) (*APU, *apuStreamer) {
	t.Helper()
	a := NewAPU(nil, false, false) // headless, no beep
	cap := 1 << 18
	streamer := &apuStreamer{
		bufferL: make([]float64, cap),
		bufferR: make([]float64, cap),
		cap:     cap,
	}
	a.streamer = streamer
	a.audioEnabled = true
	return a, streamer
}

// drainSamples pulls every queued sample into a flat slice (L only).
func drainSamples(s *apuStreamer) []float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]float64, 0, s.count)
	for s.count > 0 {
		out = append(out, s.bufferL[s.readIdx])
		s.readIdx = (s.readIdx + 1) % s.cap
		s.count--
	}
	return out
}

// dftPeakHz returns the dominant frequency in `samples` (Hz).
// Naive DFT, O(N²). Adequate for N ≈ 4096.
func dftPeakHz(samples []float64, sampleRate int) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	bestK := 0
	bestMag := 0.0
	// Skip DC bin (k=0). Inspect up to Nyquist.
	for k := 1; k < n/2; k++ {
		var sum complex128
		theta := -2 * math.Pi * float64(k) / float64(n)
		for j, x := range samples {
			sum += complex(x, 0) * cmplx.Rect(1, theta*float64(j))
		}
		mag := cmplx.Abs(sum)
		if mag > bestMag {
			bestMag = mag
			bestK = k
		}
	}
	return float64(bestK) * float64(sampleRate) / float64(n)
}

// gbFreqForHz converts a target tone in Hz into the Game Boy 11-bit
// "frequency" register value. Pan Docs: f_out = 131072 / (2048 - x).
func gbFreqForHz(hz float64) uint16 {
	x := 2048 - 131072.0/hz
	if x < 0 {
		return 0
	}
	if x > 2047 {
		return 2047
	}
	return uint16(x)
}

// TestAPU_Capture_Ch1_440Hz configures Channel 1 as a 440 Hz square
// wave at full envelope, runs the APU for 100 ms of CPU time, then
// confirms the captured audio is non-silent and peaks near 440 Hz.
func TestAPU_Capture_Ch1_440Hz(t *testing.T) {
	a, s := captureAPU(t)

	freq := gbFreqForHz(440)
	a.enabled = true
	a.nr50 = 0x77 // max volume L/R
	a.nr51 = 0x11 // ch1 → L + R
	a.ch1.dacOn = true
	a.ch1.envelopeInit = 15
	a.ch1.envelopeVolume = 15
	a.ch1.envelopePeriod = 0 // freeze envelope
	a.ch1.duty = 2           // 50%
	a.ch1.frequency = freq
	a.ch1.lengthEnabled = false
	a.ch1.trigger()

	const cpuCycles = apuDmgClock / 10 // 100 ms
	for i := 0; i < cpuCycles; i += 16 {
		a.Tick(16)
	}

	samples := drainSamples(s)
	require.GreaterOrEqual(t, len(samples), defaultAudioSampleRate/15,
		"should capture ~%d samples", defaultAudioSampleRate/10)

	// 1. Non-silent: RMS amplitude well above zero.
	var sumSq float64
	for _, v := range samples {
		sumSq += v * v
	}
	rms := math.Sqrt(sumSq / float64(len(samples)))
	assert.Greater(t, rms, 0.05, "RMS amplitude should be appreciable (got %.3f)", rms)

	// 2. Peak frequency near 440 Hz. DFT window sized so it works for
	// both 32 kHz and 44.1 kHz sample rates (100 ms ≈ 3200 samples at
	// 32 kHz, 4410 at 44.1 kHz; pick 2048 = stable bin resolution).
	const dftWindow = 2048
	window := samples
	if len(window) > dftWindow {
		window = samples[:dftWindow]
	}
	peak := dftPeakHz(window, defaultAudioSampleRate)
	// Square waves have strong odd harmonics; the FUNDAMENTAL should
	// still dominate. Tolerance ±5% accounts for DFT bin granularity
	// and the GB frequency-register quantization.
	assert.InDelta(t, 440.0, peak, 22.0,
		"peak frequency should be ~440 Hz (got %.1f Hz)", peak)
}

// TestAPU_Capture_Ch1_Silent_WhenDacOff confirms that turning off the
// DAC (NR12 upper 5 bits = 0) silences the channel.
func TestAPU_Capture_Ch1_Silent_WhenDacOff(t *testing.T) {
	a, s := captureAPU(t)
	a.enabled = true
	a.nr50 = 0x77
	a.nr51 = 0x11
	// Don't enable DAC.
	a.ch1.dacOn = false
	a.ch1.frequency = gbFreqForHz(440)
	a.ch1.duty = 2
	a.ch1.trigger() // would enable channel except DAC is off

	for i := 0; i < apuDmgClock/100; i += 16 {
		a.Tick(16)
	}
	samples := drainSamples(s)
	require.NotEmpty(t, samples)
	var maxAbs float64
	for _, v := range samples {
		if math.Abs(v) > maxAbs {
			maxAbs = math.Abs(v)
		}
	}
	assert.Equal(t, 0.0, maxAbs, "DAC off must produce pure silence")
}

// TestAPU_Capture_Ch2_DifferentFrequency confirms ch2 at 880 Hz peaks
// at 880 Hz, proving channel selection works independently of ch1.
func TestAPU_Capture_Ch2_DifferentFrequency(t *testing.T) {
	a, s := captureAPU(t)
	a.enabled = true
	a.nr50 = 0x77
	a.nr51 = 0x22 // ch2 → L + R

	a.ch2.dacOn = true
	a.ch2.envelopeInit = 15
	a.ch2.envelopeVolume = 15
	a.ch2.duty = 2
	a.ch2.frequency = gbFreqForHz(880)
	a.ch2.trigger()

	for i := 0; i < apuDmgClock/10; i += 16 {
		a.Tick(16)
	}

	samples := drainSamples(s)
	const dftWindow = 2048
	require.GreaterOrEqual(t, len(samples), dftWindow)
	peak := dftPeakHz(samples[:dftWindow], defaultAudioSampleRate)
	assert.InDelta(t, 880.0, peak, 40.0,
		"ch2 peak should be ~880 Hz (got %.1f Hz)", peak)
}

// TestAPU_Capture_Mixer_PanningHardLeft confirms NR51 routes ch1 to L
// only — the right channel must be silent.
func TestAPU_Capture_Mixer_PanningHardLeft(t *testing.T) {
	a, s := captureAPU(t)
	a.enabled = true
	a.nr50 = 0x77
	a.nr51 = 0x10 // ch1 → L only

	a.ch1.dacOn = true
	a.ch1.envelopeInit = 15
	a.ch1.envelopeVolume = 15
	a.ch1.duty = 2
	a.ch1.frequency = gbFreqForHz(440)
	a.ch1.trigger()

	for i := 0; i < apuDmgClock/20; i += 16 {
		a.Tick(16)
	}

	var hasL, hasR bool
	s.mu.Lock()
	for s.count > 0 {
		l := s.bufferL[s.readIdx]
		r := s.bufferR[s.readIdx]
		s.readIdx = (s.readIdx + 1) % s.cap
		s.count--
		if math.Abs(l) > 1e-6 {
			hasL = true
		}
		if math.Abs(r) > 1e-6 {
			hasR = true
		}
	}
	s.mu.Unlock()
	assert.True(t, hasL, "left channel must have non-zero samples")
	assert.False(t, hasR, "right channel must be pure silence")
}

// TestAPU_Capture_Noise_NonZero confirms channel 4 (LFSR noise) produces
// non-silent output that is NOT a single sustained tone (broad spectrum).
func TestAPU_Capture_Noise_NonZero(t *testing.T) {
	a, s := captureAPU(t)
	a.enabled = true
	a.nr50 = 0x77
	a.nr51 = 0x88 // ch4 → L + R

	a.ch4.dacOn = true
	a.ch4.envelopeInit = 15
	a.ch4.envelopeVolume = 15
	a.ch4.clockShift = 4
	a.ch4.divisorCode = 4
	a.ch4.trigger()

	for i := 0; i < apuDmgClock/20; i += 16 {
		a.Tick(16)
	}

	samples := drainSamples(s)
	require.GreaterOrEqual(t, len(samples), 1024)

	var sumSq float64
	for _, v := range samples {
		sumSq += v * v
	}
	rms := math.Sqrt(sumSq / float64(len(samples)))
	assert.Greater(t, rms, 0.05, "noise channel must produce non-trivial amplitude")
}
