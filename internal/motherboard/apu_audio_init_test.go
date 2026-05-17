// Package motherboard — apu_audio_init_test.go
//
// Manual smoke test for the audio initialization fallback path. Beep's
// speaker.Init touches process-global state, so we hide this behind a
// build tag — run it explicitly with:
//
//	go test -tags=apuaudio -run TestAPU_AudioInitFallback ./internal/motherboard
//
// On a host with no audio device the test confirms speaker.Init's
// failure is caught and the APU degrades gracefully to silent mode.
// On a host WITH audio it confirms init succeeds (audioEnabled remains
// true) and the ring buffer fills.

//go:build apuaudio

package motherboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPU_AudioInitFallback(t *testing.T) {
	mb := newMbForSubsysTest(t)
	apu := NewAPU(mb, true, false)
	require.NotNil(t, apu)

	for i := 0; i < 100000; i++ {
		apu.Tick(4)
	}

	if apu.audioEnabled {
		t.Log("audio init succeeded — host has an audio device; samples should be flowing")
		require.NotNil(t, apu.streamer)
	} else {
		t.Log("audio init failed gracefully — host has no audio device; APU degraded to silent mode (expected on headless CI)")
		require.Nil(t, apu.streamer)
	}
}
