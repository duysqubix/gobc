// Package motherboard — apu_streamer.go
//
// beep wiring: implements beep.Streamer over a lock-protected stereo
// ring buffer. The APU pushes samples (one stereo pair per output rate
// tick); beep's audio goroutine drains them via Stream().
//
// References:
//   - github.com/gopxl/beep/v2 — Streamer interface, speaker.Init/Play
//   - HFO4/gameboy.live — pattern of "one Streamer per APU" (we simplified to one Streamer with internal mixing)

package motherboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/duysqubix/gobc/internal"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// apuStreamer is the bridge from APU-generated samples into beep.
//
// Mutex-protected SPSC ring. Producer (APU.Tick on the emulator's
// main goroutine) calls push() ~44100×/sec; consumer (beep's audio
// goroutine) calls Stream() in blocks of ~512-2048 samples. The
// consumer takes the lock ONCE per block, the producer once per
// sample — under low contention this is faster than a buffered Go
// channel (channels also serialize on an internal mutex and pay
// per-element scheduling cost).
//
// Diagnostic counters (pushed / pulled / dropped / underruns) are
// atomic uint64s so they can be sampled from the diagnostics
// goroutine without blocking the audio path. Set APU_DEBUG=1 to
// enable the periodic log.
type apuStreamer struct {
	mu       sync.Mutex
	bufferL  []float64
	bufferR  []float64
	cap      int
	readIdx  int
	writeIdx int
	count    int

	pushed    atomic.Uint64
	pulled    atomic.Uint64
	dropped   atomic.Uint64
	underruns atomic.Uint64

	closed   atomic.Bool
	diagOnce sync.Once
}

func newAPUStreamer(capacity int) *apuStreamer {
	s := &apuStreamer{
		bufferL: make([]float64, capacity),
		bufferR: make([]float64, capacity),
		cap:     capacity,
	}
	s.prefillSilence(capacity / 2)
	s.maybeStartDiagnostics()
	return s
}

// prefillSilence seeds the ring so the speaker's first drain doesn't
// underrun while waiting for the emulator's first ~16 ms of CPU work
// to produce the first real APU samples. Without this every startup
// (and every reset) produces an immediate audible click.
func (s *apuStreamer) prefillSilence(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < n && s.count < s.cap; i++ {
		s.bufferL[s.writeIdx] = 0
		s.bufferR[s.writeIdx] = 0
		s.writeIdx = (s.writeIdx + 1) % s.cap
		s.count++
	}
}

// maybeStartDiagnostics fires a goroutine that prints push/pull/drop/
// underrun stats every 5 s when APU_DEBUG=1 is set. Writes directly to
// stderr (bypasses logrus) so the user always sees it regardless of
// LOG_LEVEL (gobc's default is ErrorLevel, which suppresses Info).
func (s *apuStreamer) maybeStartDiagnostics() {
	if os.Getenv("APU_DEBUG") != "1" {
		return
	}
	s.diagOnce.Do(func() {
		go func() {
			tick := time.NewTicker(5 * time.Second)
			defer tick.Stop()
			var lastPush, lastPull, lastDrop, lastUR uint64
			for range tick.C {
				if s.closed.Load() {
					return
				}
				p := s.pushed.Load()
				pl := s.pulled.Load()
				d := s.dropped.Load()
				u := s.underruns.Load()
				fmt.Fprintf(os.Stderr,
					"APU 5s: pushed=%d (+%d) pulled=%d (+%d) dropped=%d (+%d) underruns=%d (+%d) ring=%d/%d\n",
					p, p-lastPush, pl, pl-lastPull, d, d-lastDrop, u, u-lastUR, s.length(), s.cap,
				)
				lastPush, lastPull, lastDrop, lastUR = p, pl, d, u
			}
		}()
	})
}

func (s *apuStreamer) length() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

// AvailableSamples returns the current number of samples queued. Used
// by the main loop's adaptive frame limiter (PyBoy pattern) to decide
// whether to sleep — when the queue is full enough, sleep to maintain
// it at the target depth; when it's getting empty, run free so the
// emulator can refill the queue before the audio device underruns.
func (s *apuStreamer) AvailableSamples() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

// push enqueues a stereo sample. When the ring is full we don't
// discard — we AVERAGE the new sample into the most recently written
// slot. A burst of N "drops" then collapses into a single low-pass-
// filtered sample at that position instead of an N-sample audio gap
// (the latter is the "CD skip" sound). The original drop counter is
// still incremented for diagnostics.
func (s *apuStreamer) push(l, r float64) {
	s.mu.Lock()
	if s.count >= s.cap {
		prevIdx := (s.writeIdx - 1 + s.cap) % s.cap
		s.bufferL[prevIdx] = (s.bufferL[prevIdx] + l) * 0.5
		s.bufferR[prevIdx] = (s.bufferR[prevIdx] + r) * 0.5
		s.mu.Unlock()
		s.dropped.Add(1)
		return
	}
	s.bufferL[s.writeIdx] = l
	s.bufferR[s.writeIdx] = r
	s.writeIdx = (s.writeIdx + 1) % s.cap
	s.count++
	s.mu.Unlock()
	s.pushed.Add(1)
}

// Stream drains the ring into out. On underrun (producer can't keep
// up) it uses zero-order hold — replays the last real sample — rather
// than padding with silence. For sustained underruns this sounds like
// a tiny pitch flutter instead of audible clicks/scratching. One Lock
// per call (not per sample) minimizes contention with the per-sample
// producer.
func (s *apuStreamer) Stream(out [][2]float64) (int, bool) {
	s.mu.Lock()
	var underran uint64
	var lastL, lastR float64
	for i := range out {
		if s.count > 0 {
			lastL = s.bufferL[s.readIdx]
			lastR = s.bufferR[s.readIdx]
			s.readIdx = (s.readIdx + 1) % s.cap
			s.count--
		} else {
			underran++
		}
		out[i][0] = lastL
		out[i][1] = lastR
	}
	pulled := uint64(len(out)) - underran
	s.mu.Unlock()
	s.pulled.Add(pulled)
	if underran > 0 {
		s.underruns.Add(underran)
	}
	return len(out), true
}

// Err is beep.Streamer.Err: never errors (infinite stream).
func (s *apuStreamer) Err() error { return nil }

// flush drops any buffered samples and re-primes silence. Called on
// APU reset to avoid stale audio bleeding into the new session.
func (s *apuStreamer) flush() {
	s.mu.Lock()
	s.readIdx, s.writeIdx, s.count = 0, 0, 0
	s.mu.Unlock()
	s.prefillSilence(s.cap / 2)
}

// close stops audio playback. Idempotent.
func (s *apuStreamer) close() {
	s.closed.Store(true)
	speaker.Clear()
}

// startStreamer brings the beep speaker up and connects the APU ring
// buffer to it. Idempotent across emulator resets (beep's speaker is a
// process-global singleton that rejects double-Init).
//
// All failure modes degrade gracefully to silent operation:
//
//   - No audio device present (audioAvailable() == false) → skip the
//     speaker.Init call entirely, log an info line, leave the emulator
//     running silent. This is the common case on headless servers and
//     on WSL without libasound2-plugins installed.
//   - speaker.Init returns an error → log warning, run silent.
//
// libasound itself writes config-parse diagnostics directly to fd 2
// before returning the error code. We wrap speaker.Init in
// silenceStderr() so end users don't see the ALSA spew when their
// system has a partial audio config.
func (a *APU) startStreamer() error {
	if !audioAvailable() {
		internal.Logger.Info("APU: no audio device detected; running silent. " +
			"On WSL install `libasound2-plugins` and add an `~/.asoundrc` pulse PCM; " +
			"on bare Linux install pulseaudio or pipewire-pulse.")
		a.audioEnabled = false
		return nil
	}

	sr := beep.SampleRate(a.sampleRate)
	a.streamer = newAPUStreamer(apuRingBufferCap)

	if !speakerInitialized {
		restore := silenceStderr()
		err := speaker.Init(sr, sr.N(time.Second/apuBufferDuration))
		restore()
		if err != nil {
			internal.Logger.Warnf("APU: speaker init failed (%v); running silent.", err)
			a.audioEnabled = false
			a.streamer = nil
			return nil
		}
		speakerInitialized = true
	}
	speaker.Clear()
	speaker.Play(a.streamer)
	return nil
}

// audioAvailable reports whether the host has SOME plausible audio
// sink we can target. Returns true when any of:
//
//   - /dev/snd exists and contains entries (real ALSA cards)
//   - $PULSE_SERVER points at a PulseAudio socket (covers WSLg via
//     /mnt/wslg/PulseServer)
//   - $XDG_RUNTIME_DIR/pulse/native exists (standard Linux PulseAudio)
//
// Returning true does NOT guarantee audio will work — libasound may
// still fail to open the PCM — but returning false reliably avoids
// the ALSA error spam on hosts with no audio path at all.
func audioAvailable() bool {
	if entries, err := os.ReadDir("/dev/snd"); err == nil && len(entries) > 0 {
		return true
	}
	if os.Getenv("PULSE_SERVER") != "" {
		return true
	}
	if r := os.Getenv("XDG_RUNTIME_DIR"); r != "" {
		if _, err := os.Stat(filepath.Join(r, "pulse", "native")); err == nil {
			return true
		}
	}
	return false
}

// Process-wide flag because beep's speaker is a global singleton.
var speakerInitialized bool
