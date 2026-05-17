// Package motherboard — apu_stderr_other.go
//
// No-op silenceStderr for non-Unix platforms (Windows). Audio drivers
// on those platforms don't dump diagnostic spew to stderr the way
// libasound does, so the redirect is unnecessary.

//go:build !unix

package motherboard

func silenceStderr() (restore func()) {
	return func() {}
}
