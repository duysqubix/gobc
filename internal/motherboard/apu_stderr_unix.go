// Package motherboard — apu_stderr_unix.go
//
// libasound writes its config-parse diagnostics directly to fd 2 via
// its built-in error handler (see `snd_lib_error_set_handler`). oto
// does not install a Go-side handler, so the only reliable way to
// silence the spam during `speaker.Init` on systems with broken /
// missing ALSA config (most notably WSL without `libasound2-plugins`)
// is to redirect fd 2 to /dev/null around the init call.
//
// silenceStderr returns a restore function callers must `defer`. If
// any step of the redirect fails it returns a no-op restore — never
// errors out, since silencing is best-effort.

//go:build unix

package motherboard

import (
	"os"
	"syscall"
)

func silenceStderr() (restore func()) {
	noop := func() {}

	stderrFd := int(os.Stderr.Fd())
	saved, err := syscall.Dup(stderrFd)
	if err != nil {
		return noop
	}

	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		_ = syscall.Close(saved)
		return noop
	}

	if err := syscall.Dup2(int(devnull.Fd()), stderrFd); err != nil {
		_ = devnull.Close()
		_ = syscall.Close(saved)
		return noop
	}

	return func() {
		_ = syscall.Dup2(saved, stderrFd)
		_ = syscall.Close(saved)
		_ = devnull.Close()
	}
}
