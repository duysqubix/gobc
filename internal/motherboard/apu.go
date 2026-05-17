// Package motherboard — apu.go
//
// Game Boy Audio Processing Unit (APU) — top-level type, register I/O,
// lifecycle and frame-sequencer scheduling.
//
// Channel DSP lives in apu_square.go / apu_wave.go / apu_noise.go.
// beep speaker wiring + sample ring buffer live in apu_streamer.go.
//
// References:
//   - Pan Docs: https://gbdev.io/pandocs/Audio.html
//   - Humpheh/goboy (cycle-driven sample architecture)
//   - HFO4/gameboy.live (beep usage idioms)

package motherboard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/duysqubix/gobc/internal"
)

// Audio constants.
const (
	defaultAudioSampleRate = 32000             // Default host output rate. 32 kHz gives 27% more cycle-budget headroom than 44.1 kHz while staying well above the Nyquist for any GB-audio spectral content.
	apuDmgClock            = 4194304           // DMG CPU clock (Hz).
	apuRingBufferCap       = 16384             // ~500 ms of stereo headroom at 32 kHz.
	apuBufferDuration      = 5                 // speaker.Init period divisor: time.Second/5 = ~200 ms latency.
	apuFrameSeqPeriod      = apuDmgClock / 512 // CPU cycles per 512 Hz frame-sequencer step.
	apuCh1Bit              = 0
	apuCh2Bit              = 1
	apuCh3Bit              = 2
	apuCh4Bit              = 3
)

// audioSampleRateOverride lets the user pin the audio rate via the
// --audio-rate CLI flag. Slow CPUs (e.g., WSL2 hosts that can't sustain
// 60 FPS of gobc emulation) need to lower this to match their actual
// sample-production rate, otherwise the producer underruns and audio
// chops. Zero = use defaultAudioSampleRate.
var audioSampleRateOverride int

// SetAudioSampleRateOverride sets the audio sample rate for newly-
// created APUs. Must be called BEFORE NewMotherboard. Pass 0 to clear.
func SetAudioSampleRateOverride(hz int) {
	audioSampleRateOverride = hz
}

func effectiveAudioSampleRate() int {
	if audioSampleRateOverride > 0 {
		return audioSampleRateOverride
	}
	return defaultAudioSampleRate
}

// APU register read masks. Bits set to 1 are OR'd into the read value
// (= unused / unreadable bits return 1, per Pan Docs).
//
//	0xFF10..0xFF3F (32 bytes incl. Wave RAM 0xFF30..0xFF3F).
var apuReadMask = [0x30]byte{
	0x80, 0x3F, 0x00, 0xFF, 0xBF, // FF10..FF14 (NR10..NR14)
	0xFF, 0x3F, 0x00, 0xFF, 0xBF, // FF15..FF19 (unused, NR21..NR24)
	0x7F, 0xFF, 0x9F, 0xFF, 0xBF, // FF1A..FF1E (NR30..NR34)
	0xFF,                   // FF1F (unused)
	0xFF, 0x00, 0x00, 0xBF, // FF20..FF23 (NR41..NR44)
	0x00, 0x00, 0x70, // FF24..FF26 (NR50, NR51, NR52)
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // FF27..FF2F (unused)
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // FF30..FF37 (Wave RAM)
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // FF38..FF3F (Wave RAM)
}

// APU implements the Game Boy 4-channel audio processing unit.
//
// All emulator-internal state lives on this struct; the host audio
// goroutine reads samples through the ring buffer in apu_streamer.go.
type APU struct {
	Mb *Motherboard

	// Master enable (NR52 bit 7).
	enabled bool

	// Channels (concrete types live in apu_*.go).
	ch1 *squareChannel // with sweep
	ch2 *squareChannel // no sweep
	ch3 *waveChannel
	ch4 *noiseChannel

	// Master volume (NR50). Bits 6-4 = left vol, bits 2-0 = right vol.
	// Bit 7 (Vin→L) and bit 3 (Vin→R) are stored but ignored (no commercial cart uses Vin).
	nr50 byte
	// Panning (NR51). High nibble → left, low nibble → right; bit per channel.
	nr51 byte

	// Wave RAM at 0xFF30..0xFF3F (16 bytes = 32 4-bit samples).
	waveRAM [16]byte

	// Frame sequencer state.
	frameSeqStep    uint8 // 0..7
	frameSeqCounter int   // CPU cycles toward next 512 Hz tick

	// Sample-rate downsampling. cyclesPerSampleQ16 is Q16 fixed-point
	// (CPU cycles per output sample × 65536). MUST stay at the spec
	// rate of DMG_clock / sampleRate — changing this dynamically
	// distorts audio pitch and tempo (the relationship between sample
	// index and game-time progression is wrong). Producer-consumer rate
	// mismatch on slow hosts has to be solved by a different mechanism
	// (resampler, demand-driven sampling, or consumer-rate match).
	cyclesPerSampleQ16 int
	sampleClockQ16     int
	sampleRate         int

	// Audio toggle (--no-audio CLI flag). If false: no speaker, no ring buffer push.
	audioEnabled bool

	// beep streamer & ring buffer. Allocated in NewAPU when audioEnabled.
	streamer *apuStreamer

	// Smooth-mode lazy init. When --audio-smooth is set, the speaker is
	// initialized AFTER a 500 ms benchmark to measure host throughput,
	// then opened at exactly the rate the host can sustain (= throughput
	// / specCyclesPerSample). Producer rate then equals consumer rate by
	// construction — no underruns, no drops. Cost: ~1.9% pitch drop on a
	// 98%-speed host. This is the inescapable price of smooth audio on a
	// CPU-bound emulator.
	smoothMode      bool
	smoothInitDone  bool
	smoothCycles    int
	smoothStartedAt time.Time
}

// AudioQueueFramesBuffered returns the current audio queue depth in
// terms of "frames worth of samples". 1.0 = enough for one 60 FPS frame
// of audio. The main loop's adaptive frame limiter uses this to pace
// the emulator: sleep when buffer is full, run free when it's empty.
// Mirrors PyBoy's _get_sound_frames_buffered().
func (a *APU) AudioQueueFramesBuffered() float64 {
	if a.streamer == nil {
		return 0
	}
	samplesPerFrame := float64(a.sampleRate) / 60.0
	if samplesPerFrame <= 0 {
		return 0
	}
	return float64(a.streamer.AvailableSamples()) / samplesPerFrame
}

// AudioEnabled reports whether the APU is currently producing audio
// (false in --no-audio mode or when audio init failed).
func (a *APU) AudioEnabled() bool { return a.audioEnabled }

// NewAPU constructs and initializes the APU. Safe to call even when
// audioEnabled is false — the APU still emulates registers correctly,
// it just doesn't produce audible output.
//
// In default mode the speaker opens at the spec sample rate (32 kHz).
// In smooth mode (smoothMode=true) the speaker init is deferred until
// the APU has measured the host's actual throughput for 500 ms; the
// speaker then opens at a rate matched to that throughput, so producer
// and consumer rates equal each other exactly — no underruns ever, at
// the cost of a ~1.9 % pitch drop on a 98 %-speed host (one third of a
// semitone — usually below the user-detectable threshold).
func NewAPU(mb *Motherboard, audioEnabled bool, smoothMode bool) *APU {
	a := &APU{
		Mb:           mb,
		audioEnabled: audioEnabled,
		sampleRate:   effectiveAudioSampleRate(),
		smoothMode:   smoothMode,
	}
	a.cyclesPerSampleQ16 = (apuDmgClock << 16) / a.sampleRate
	a.ch1 = newSquareChannel(true)  // sweep enabled
	a.ch2 = newSquareChannel(false) // no sweep
	a.ch3 = newWaveChannel(a)
	a.ch4 = newNoiseChannel()
	a.applyPostBootState()

	if audioEnabled && !smoothMode {
		if err := a.startStreamer(); err != nil {
			internal.Logger.Warnf("APU: failed to start audio output: %v (continuing silently)", err)
			a.audioEnabled = false
		}
	}
	// In smooth mode startStreamer is deferred — see Tick() / lazyInitSmoothAudio.
	return a
}

// applyPostBootState sets the "post boot ROM" register values for the
// APU. These match the values Pan Docs lists for power-on state after
// the boot ROM has run on a DMG.
func (a *APU) applyPostBootState() {
	// Power-on default register values (post boot ROM, DMG).
	// These match the values written by the boot ROM.
	a.enabled = true
	a.nr50 = 0x77
	a.nr51 = 0xF3
	// Wave RAM after boot ROM: alternating pattern.
	for i := 0; i < 16; i++ {
		if i&1 == 0 {
			a.waveRAM[i] = 0x00
		} else {
			a.waveRAM[i] = 0xFF
		}
	}
	// Channel 1 post-boot: NR11=0xBF, NR12=0xF3, NR14=0xBF triggered.
	a.ch1.writeReg(1, 0xBF, 0)
	a.ch1.writeReg(2, 0xF3, 0)
	a.ch1.writeReg(4, 0xBF, 0)
}

// lazyInitSmoothAudio runs the 500 ms throughput benchmark and opens
// the speaker at a host-matched rate so the producer and consumer
// rates equal each other exactly. Called from Tick() before audio
// emission starts in smooth mode.
//
// We can't run this from NewAPU because Motherboard.Tick depends on
// CPU + BootRom which are constructed AFTER NewAPU. Lazy init in
// Tick lets the gameLoop drive the benchmark naturally.
func (a *APU) lazyInitSmoothAudio(cycles int) {
	if a.smoothStartedAt.IsZero() {
		a.smoothStartedAt = time.Now()
	}
	a.smoothCycles += cycles
	elapsed := time.Since(a.smoothStartedAt)
	if elapsed < 500*time.Millisecond {
		return
	}

	throughput := float64(a.smoothCycles) / elapsed.Seconds()
	specCyclesPerSample := float64(apuDmgClock) / float64(a.sampleRate)
	// 5 % safety margin. Empirically the 500 ms benchmark over-estimates
	// steady-state throughput by 3-4 % (game throughput varies by scene),
	// so a thin margin lets the ring drain to empty over time. 5 % covers
	// the variance and guarantees the producer always exceeds the
	// consumer rate. Cost: ~5-7 % pitch drop on a slow host (~1 semitone).
	// User is told to omit --audio-smooth if pitch drop is unacceptable.
	const smoothSafetyFactor = 0.95
	measuredRate := int(throughput / specCyclesPerSample * smoothSafetyFactor)

	if measuredRate < a.sampleRate/2 || measuredRate > a.sampleRate*2 {
		fmt.Fprintf(os.Stderr,
			"APU smooth: benchmark produced implausible rate %d Hz (throughput=%.0f), falling back to default %d Hz\n",
			measuredRate, throughput, a.sampleRate)
	} else {
		fmt.Fprintf(os.Stderr,
			"APU smooth: host throughput=%.0f cycles/sec → opening speaker at %d Hz (%.1f%% of spec; ~%.1f%% pitch drop incl. 5%% safety margin)\n",
			throughput, measuredRate,
			float64(measuredRate)/float64(a.sampleRate)*100,
			(1.0-float64(measuredRate)/float64(a.sampleRate))*100)
		a.sampleRate = measuredRate
		a.cyclesPerSampleQ16 = (apuDmgClock << 16) / a.sampleRate
	}

	if err := a.startStreamer(); err != nil {
		internal.Logger.Warnf("APU smooth: speaker init failed (%v); running silent", err)
		a.audioEnabled = false
	}
	a.smoothInitDone = true
}

// Reset returns the APU to a fresh power-on state. Called from
// Motherboard.Reset() (the R hotkey).
func (a *APU) Reset() {
	a.enabled = false
	a.nr50 = 0
	a.nr51 = 0
	a.ch1.reset()
	a.ch2.reset()
	a.ch3.reset()
	a.ch4.reset()
	a.frameSeqStep = 0
	a.frameSeqCounter = 0
	a.sampleClockQ16 = 0
	a.applyPostBootState()
	if a.streamer != nil {
		a.streamer.flush()
	}
}

// Close shuts down audio output cleanly. Idempotent.
func (a *APU) Close() {
	if a.streamer != nil {
		a.streamer.close()
	}
}

// Tick advances APU state by `cycles` CPU clocks. Called by
// Motherboard.Tick() every CPU step.
func (a *APU) Tick(cycles OpCycles) {
	if a.smoothMode && !a.smoothInitDone && a.audioEnabled {
		// Defer the benchmark until the boot ROM has finished. Boot ROM
		// is simpler code that runs ~3% faster than steady-state game code;
		// measuring during boot ROM over-estimates throughput and the audio
		// rate calibration overshoots → ring drains slowly → chop returns
		// after ~30 s.
		if a.Mb == nil || !a.Mb.BootRomEnabled() {
			a.lazyInitSmoothAudio(int(cycles))
		}
	}

	if !a.enabled {
		// APU off: still emit silence samples so the streamer doesn't underrun.
		a.emitSilence(int(cycles))
		return
	}

	steps := int(cycles)

	// Drive each channel's period timer.
	a.ch1.step(steps)
	a.ch2.step(steps)
	a.ch3.step(steps)
	a.ch4.step(steps)

	// Drive the 512 Hz frame sequencer.
	a.frameSeqCounter += steps
	for a.frameSeqCounter >= apuFrameSeqPeriod {
		a.frameSeqCounter -= apuFrameSeqPeriod
		a.stepFrameSequencer()
	}

	// Emit downsampled output samples. Q16 fixed-point accumulator avoids
	// integer-truncation drift; cyclesPerSampleQ16 itself is continuously
	// adapted by maybeCalibrate() to match the host's actual throughput.
	a.sampleClockQ16 += steps << 16
	for a.sampleClockQ16 >= a.cyclesPerSampleQ16 {
		a.sampleClockQ16 -= a.cyclesPerSampleQ16
		a.emitSample()
	}
}

// stepFrameSequencer advances the 8-step, 512 Hz sequencer one step.
//
//	Step | Length | Envelope | Sweep
//	-----|--------|----------|------
//	  0  |   X    |          |
//	  1  |        |          |
//	  2  |   X    |          |  X
//	  3  |        |          |
//	  4  |   X    |          |
//	  5  |        |          |
//	  6  |   X    |          |  X
//	  7  |        |    X     |
func (a *APU) stepFrameSequencer() {
	switch a.frameSeqStep {
	case 0, 4:
		a.ch1.clockLength()
		a.ch2.clockLength()
		a.ch3.clockLength()
		a.ch4.clockLength()
	case 2, 6:
		a.ch1.clockLength()
		a.ch2.clockLength()
		a.ch3.clockLength()
		a.ch4.clockLength()
		a.ch1.clockSweep()
	case 7:
		a.ch1.clockEnvelope()
		a.ch2.clockEnvelope()
		a.ch4.clockEnvelope()
	}
	a.frameSeqStep = (a.frameSeqStep + 1) & 7
}

// emitSample produces one stereo sample and pushes it into the ring buffer.
func (a *APU) emitSample() {
	if !a.audioEnabled || a.streamer == nil {
		return
	}
	s1 := a.ch1.output()
	s2 := a.ch2.output()
	s3 := a.ch3.output()
	s4 := a.ch4.output()

	var l, r float64
	if a.nr51&0x10 != 0 {
		l += s1
	}
	if a.nr51&0x01 != 0 {
		r += s1
	}
	if a.nr51&0x20 != 0 {
		l += s2
	}
	if a.nr51&0x02 != 0 {
		r += s2
	}
	if a.nr51&0x40 != 0 {
		l += s3
	}
	if a.nr51&0x04 != 0 {
		r += s3
	}
	if a.nr51&0x80 != 0 {
		l += s4
	}
	if a.nr51&0x08 != 0 {
		r += s4
	}
	// Average over the 4 channels then apply NR50 master volume (0..7 → /8).
	leftMaster := float64((a.nr50>>4)&0x07) / 7.0
	rightMaster := float64(a.nr50&0x07) / 7.0
	l = (l / 4.0) * leftMaster
	r = (r / 4.0) * rightMaster
	a.streamer.push(l, r)
}

// emitSilence pushes silent samples to keep the ring buffer fed when
// the APU is disabled. Avoids underrun-induced crackle.
func (a *APU) emitSilence(cycles int) {
	if !a.audioEnabled || a.streamer == nil {
		return
	}
	a.sampleClockQ16 += cycles << 16
	for a.sampleClockQ16 >= a.cyclesPerSampleQ16 {
		a.sampleClockQ16 -= a.cyclesPerSampleQ16
		a.streamer.push(0, 0)
	}
}

// Read returns the value at the given APU register address. The caller
// (motherboard_getitem.go) guarantees addr is in [0xFF10, 0xFF3F].
func (a *APU) Read(addr uint16) uint8 {
	idx := addr - 0xFF10
	mask := apuReadMask[idx]

	if addr >= 0xFF30 && addr <= 0xFF3F {
		if a.ch3.enabled {
			if !a.ch3.waveFormJustRead {
				return 0xFF
			}
			return a.ch3.currentSampleByte()
		}
		return a.waveRAM[addr-0xFF30]
	}

	var v byte
	switch addr {
	case 0xFF10:
		v = a.ch1.readReg(0)
	case 0xFF11:
		v = a.ch1.readReg(1)
	case 0xFF12:
		v = a.ch1.readReg(2)
	case 0xFF13:
		v = a.ch1.readReg(3)
	case 0xFF14:
		v = a.ch1.readReg(4)
	case 0xFF16:
		v = a.ch2.readReg(1)
	case 0xFF17:
		v = a.ch2.readReg(2)
	case 0xFF18:
		v = a.ch2.readReg(3)
	case 0xFF19:
		v = a.ch2.readReg(4)
	case 0xFF1A:
		v = a.ch3.readReg(0)
	case 0xFF1B:
		v = a.ch3.readReg(1)
	case 0xFF1C:
		v = a.ch3.readReg(2)
	case 0xFF1D:
		v = a.ch3.readReg(3)
	case 0xFF1E:
		v = a.ch3.readReg(4)
	case 0xFF20:
		v = a.ch4.readReg(1)
	case 0xFF21:
		v = a.ch4.readReg(2)
	case 0xFF22:
		v = a.ch4.readReg(3)
	case 0xFF23:
		v = a.ch4.readReg(4)
	case 0xFF24:
		v = a.nr50
	case 0xFF25:
		v = a.nr51
	case 0xFF26:
		v = a.readNR52()
	}
	return v | mask
}

// readNR52 returns NR52: bit 7 = APU enable, bits 0-3 = per-channel active flags.
func (a *APU) readNR52() byte {
	var v byte
	if a.enabled {
		v |= 0x80
	}
	if a.ch1.enabled {
		v |= 1 << apuCh1Bit
	}
	if a.ch2.enabled {
		v |= 1 << apuCh2Bit
	}
	if a.ch3.enabled {
		v |= 1 << apuCh3Bit
	}
	if a.ch4.enabled {
		v |= 1 << apuCh4Bit
	}
	return v
}

// Write applies a CPU write to an APU register. The caller
// (motherboard_setitem.go) guarantees addr is in [0xFF10, 0xFF3F].
func (a *APU) Write(addr uint16, v uint8) {
	// Wave RAM writes are always permitted (even with APU disabled).
	if addr >= 0xFF30 && addr <= 0xFF3F {
		if a.ch3.enabled {
			if a.ch3.waveFormJustRead {
				a.waveRAM[a.ch3.wavePos/2] = v
			}
		} else {
			a.waveRAM[addr-0xFF30] = v
		}
		return
	}

	// NR52 master enable is always writable.
	if addr == 0xFF26 {
		newEnabled := v&0x80 != 0
		if a.enabled && !newEnabled {
			// Disabling: zero NR10..NR51 + disable all channels.
			a.powerOff()
		} else if !a.enabled && newEnabled {
			// Re-enabling: reset frame sequencer step.
			a.enabled = true
			a.frameSeqStep = 0
		}
		return
	}

	// DMG quirk (Blargg dmg_sound test 08 "len ctr during power"):
	// while APU is off, length-load writes to NR11/NR21/NR31/NR41 still
	// take effect. Square+noise: only the low 6 length bits write through
	// (the upper bits, including NR11/NR21 duty, are ignored). Wave: full
	// 8 bits write through.  All other registers stay ignored when off.
	if !a.enabled {
		switch addr {
		case 0xFF11:
			a.ch1.lengthLoad = v & 0x3F
			a.ch1.lengthCounter = uint16(64 - int(a.ch1.lengthLoad))
		case 0xFF16:
			a.ch2.lengthLoad = v & 0x3F
			a.ch2.lengthCounter = uint16(64 - int(a.ch2.lengthLoad))
		case 0xFF1B:
			a.ch3.lengthLoad = v
			a.ch3.lengthCounter = uint16(256 - int(v))
		case 0xFF20:
			a.ch4.lengthLoad = v & 0x3F
			a.ch4.lengthCounter = uint16(64 - int(a.ch4.lengthLoad))
		}
		return
	}

	fs := a.frameSeqStep
	switch addr {
	case 0xFF10:
		a.ch1.writeReg(0, v, fs)
	case 0xFF11:
		a.ch1.writeReg(1, v, fs)
	case 0xFF12:
		a.ch1.writeReg(2, v, fs)
	case 0xFF13:
		a.ch1.writeReg(3, v, fs)
	case 0xFF14:
		a.ch1.writeReg(4, v, fs)
	case 0xFF16:
		a.ch2.writeReg(1, v, fs)
	case 0xFF17:
		a.ch2.writeReg(2, v, fs)
	case 0xFF18:
		a.ch2.writeReg(3, v, fs)
	case 0xFF19:
		a.ch2.writeReg(4, v, fs)
	case 0xFF1A:
		a.ch3.writeReg(0, v, fs)
	case 0xFF1B:
		a.ch3.writeReg(1, v, fs)
	case 0xFF1C:
		a.ch3.writeReg(2, v, fs)
	case 0xFF1D:
		a.ch3.writeReg(3, v, fs)
	case 0xFF1E:
		a.ch3.writeReg(4, v, fs)
	case 0xFF20:
		a.ch4.writeReg(1, v, fs)
	case 0xFF21:
		a.ch4.writeReg(2, v, fs)
	case 0xFF22:
		a.ch4.writeReg(3, v, fs)
	case 0xFF23:
		a.ch4.writeReg(4, v, fs)
	case 0xFF24:
		a.nr50 = v
	case 0xFF25:
		a.nr51 = v
	}
}

// powerOff zeroes all sound registers and disables channels. Triggered
// when NR52 bit 7 is cleared. Wave RAM is preserved (DMG behavior).
func (a *APU) powerOff() {
	a.enabled = false
	a.nr50 = 0
	a.nr51 = 0
	a.ch1.powerOff()
	a.ch2.powerOff()
	a.ch3.powerOff()
	a.ch4.powerOff()
	a.frameSeqStep = 0
}

// readWaveRAMByte exposes wave RAM to the wave channel without going
// through the masked Read() path.
func (a *APU) readWaveRAMByte(idx int) byte {
	return a.waveRAM[idx&0x0F]
}

// ───── save state ──────────────────────────────────────────────────────

// Serialize writes the APU state into a byte buffer. Layout:
//
//	[version:u8=1]
//	[enabled:bool][nr50:u8][nr51:u8]
//	[waveRAM:16]
//	[frameSeqStep:u8][frameSeqCounter:int32]
//	[sampleClock:int32]
//	[ch1 state...][ch2 state...][ch3 state...][ch4 state...]
//
// Format version 1.
func (a *APU) Serialize() *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint8(1)) // version
	binary.Write(buf, binary.LittleEndian, a.enabled)
	binary.Write(buf, binary.LittleEndian, a.nr50)
	binary.Write(buf, binary.LittleEndian, a.nr51)
	binary.Write(buf, binary.LittleEndian, a.waveRAM)
	binary.Write(buf, binary.LittleEndian, a.frameSeqStep)
	binary.Write(buf, binary.LittleEndian, int32(a.frameSeqCounter))
	binary.Write(buf, binary.LittleEndian, int32(a.sampleClockQ16))
	buf.Write(a.ch1.serialize().Bytes())
	buf.Write(a.ch2.serialize().Bytes())
	buf.Write(a.ch3.serialize().Bytes())
	buf.Write(a.ch4.serialize().Bytes())
	return buf
}

// Deserialize restores APU state. Returns an error if the version is unknown.
func (a *APU) Deserialize(data *bytes.Buffer) error {
	var version uint8
	if err := binary.Read(data, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != 1 {
		return errAPUUnknownVersion
	}
	if err := binary.Read(data, binary.LittleEndian, &a.enabled); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &a.nr50); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &a.nr51); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &a.waveRAM); err != nil {
		return err
	}
	if err := binary.Read(data, binary.LittleEndian, &a.frameSeqStep); err != nil {
		return err
	}
	var fsc int32
	if err := binary.Read(data, binary.LittleEndian, &fsc); err != nil {
		return err
	}
	a.frameSeqCounter = int(fsc)
	var sc int32
	if err := binary.Read(data, binary.LittleEndian, &sc); err != nil {
		return err
	}
	a.sampleClockQ16 = int(sc)
	if err := a.ch1.deserialize(data); err != nil {
		return err
	}
	if err := a.ch2.deserialize(data); err != nil {
		return err
	}
	if err := a.ch3.deserialize(data); err != nil {
		return err
	}
	if err := a.ch4.deserialize(data); err != nil {
		return err
	}
	return nil
}
