# Audio Test Status — Blargg `dmg_sound` + `cgb_sound`

**Suite source:** [Blargg dmg_sound](https://gbdev.gg8.se/files/roms/blargg-gb-tests/dmg_sound.zip) (mirror: [retrio/gb-test-roms](https://github.com/retrio/gb-test-roms))
**ROMs in repo:** [default_rom/blarrg/dmg_sound/](../default_rom/blarrg/dmg_sound/)
**Runner:** `just test-rom-audio`

## Audio runtime requirements

gobc uses [`gopxl/beep/v2`](https://github.com/gopxl/beep) → [`ebitengine/oto/v3`](https://github.com/ebitengine/oto), which on Linux talks to **libasound** (ALSA) directly. The emulator probes the host before initialising audio; if no plausible sink is found it logs one info line and runs silent (no ALSA error spam).

| Host | Status | What to install |
|---|---|---|
| **Bare-metal Linux** with PulseAudio or PipeWire-pulse running | Works out of the box | nothing |
| **Bare-metal Linux** with ALSA-only | Works out of the box | nothing |
| **WSL2 with WSLg** (`$PULSE_SERVER=unix:/mnt/wslg/PulseServer`) | Needs an ALSA → pulse bridge | `sudo apt install libasound2-plugins` + the `~/.asoundrc` snippet below |
| **Headless CI / container** (no `/dev/snd`, no `PULSE_SERVER`) | Silent, no crash, no spam | nothing — preflight skips speaker.Init |
| **macOS / Windows** | Works out of the box (oto uses CoreAudio / WASAPI) | nothing |

### WSL2 setup (one-time)

The ALSA spam (`cannot find card '0'` / `Unknown PCM default`) means **libasound has no working PCM**. WSL2 ships without `libasound2-plugins`, so even though WSLg's PulseAudio is running (socket at `/mnt/wslg/PulseServer`), ALSA can't reach it.

```bash
# 1. Install the ALSA→Pulse bridge + verification utilities
sudo apt update && sudo apt install -y libasound2-plugins pulseaudio-utils

# 2. Tell ALSA to use pulse as the default PCM.
#    The `!` prefix is required (means "override the system default").
#    `server` line is optional — libasound's pulse plugin honors $PULSE_SERVER.
cat > ~/.asoundrc <<'EOF'
pcm.!default { type pulse }
ctl.!default { type pulse }
EOF

# 3. Verify in order — each step is independent:
pactl info                                                 # WSLg pulse reachable?
paplay /usr/share/sounds/alsa/Front_Center.wav             # pulse plays?
speaker-test -c 2 -t sine -f 440 -l 1                      # ALSA→pulse bridge?
./bin/gobc <rom>                                           # gobc audible
```

**Gotchas:**

| Symptom | Cause |
|---|---|
| `pactl info` hangs or "Connection refused" | WSLg pulse not running. Run `wsl --update` from PowerShell on the Windows side. |
| `pactl info` works, `speaker-test` errors | `libasound2-plugins` not installed, OR `~/.asoundrc` syntax is the legacy `pcm.default pulse` (no braces) form. Use the snippet above verbatim. |
| All three work, gobc still silent | Run `./bin/gobc` (without `--no-gui` and without `--no-audio`); also confirm `LOG_LEVEL=info` doesn't show "APU: no audio device detected". |

### Disabling audio explicitly

```bash
./bin/gobc --no-audio <rom>
```

`--no-gui` also implies no-audio (CI / headless test ROMs never attempt audio init).

### Tuning audio for slow CPUs (WSL, low-end Linux)

If audio is choppy or "CD-scratching", your CPU is producing samples slower than the audio device consumes them. Confirm with the built-in diagnostic:

```bash
APU_DEBUG=1 ./bin/gobc <rom> 2>&1 | grep "APU 5s"
```

A line like:
```
APU 5s: pushed=157000 (+157000) pulled=160000 (+160000) dropped=0 (+0) underruns=2500 (+2500) ring=3800/16384
```
means producer is at ~31,400/sec but consumer wants 32,000/sec (the default sample rate) — a 1.9% deficit that manifests as ~500 underruns/sec of audible chop.

**Fix:** match the consumer rate to your actual production rate with `--audio-rate N`:

```bash
./bin/gobc --audio-rate 31400 <rom>
```

Pick N from the `pushed` rate divided by 5 (samples/sec). Audio pitch is shifted (N / 32000), but at ≥30 kHz the shift is imperceptible. The framerate target itself is corrected per Pan Docs (59.7275 Hz instead of 60 Hz), and frame pacing uses sleep-then-spin to avoid `time.Sleep`'s 1-4 ms granularity overhead on Linux/WSL.

## Pass/Fail Matrix

| # | Test | Status | Notes |
|---|---|:-:|---|
| 01 | registers | ✅ PASS | Register read masks, NR52 disable behavior, Wave RAM preserved across power-off |
| 02 | len ctr | ✅ PASS | 256 Hz length-counter decrement, zero-load = max |
| 03 | trigger | ✅ PASS | Length-clock-on-trigger "obscure behavior" — extra clock when length-enable transitions in FS first half; reload-to-max-1 on trigger-with-zero-length |
| 04 | sweep | ✅ PASS | NR10 frequency sweep core logic |
| 05 | sweep details | ✅ PASS | Period=0→8, negate mode, exiting negate disables channel |
| 06 | overflow on trigger | ✅ PASS | Sweep overflow at >2047 disables channel |
| 07 | len sweep period sync | ✅ PASS | Frame-sequencer phase synchronization |
| 08 | len ctr during power | ✅ PASS | NR11/NR21/NR31/NR41 length writes are accepted while APU is powered off on DMG; length counters preserved across power-off |
| 09 | wave read while on | ✅ PASS | DMG `wave_form_just_read` model — wave-RAM reads return 0xFF unless the channel just finished sampling within the last APU tick chunk |
| 10 | wave trigger while on | ✅ PASS | Retrigger within the 2-T-cycle pre-fetch window copies the next sample byte into waveRAM[0] (or a 4-byte block) — the DMG "wave corruption on retrigger" bug |
| 11 | regs after power | ✅ PASS | NR41 length-load and the per-channel length counters survive APU power-off (DMG quirk) |
| 12 | wave write while on | ✅ PASS | Wave-RAM writes only land while the channel is mid-fetch (same `wave_form_just_read` gate as test 09) |

**Current score: 12/12 passing (DMG).**

All 12 tests are guarded by CI (see `integration-rom-tests` job in `.github/workflows/go.yml`).

## CGB Sound — `cgb_sound`

The same 12-ROM suite recompiled with `REQUIRE_CGB`. The DMG and CGB
expected CRCs differ for tests 08-12: CGB resets length counters on
power-off, allows normal wave-RAM access during channel 3 playback, and
does not exhibit the wave-trigger corruption bug. All 12 pass.

The APU runs the DMG-specific quirks (length preservation across
NR52 power-off, wave-RAM 0xFF gating, wave retrigger corruption) only
when `motherboard.Cgb == false`; in CGB mode the same code paths take
the modern behaviour. The split is documented in `internal/motherboard/apu.go`
(`powerOff`) and `apu_wave.go` (`trigger`).

## How verification works

dmg_sound ROMs do not use the serial port — they write results to **cartridge SRAM at `$A000`** per the Blargg shell convention:

| Address | Meaning |
|---|---|
| `$A000` | `0x80` while running; final result code (`0x00` = pass) when done |
| `$A001-$A003` | Signature `DE B0 61` — confirms cart RAM holds real test output |
| `$A004+` | Zero-terminated ASCII test log (e.g. `"01-registers\n\n\nPassed\n"`) |

gobc auto-saves cart SRAM to a `.sav` file on shutdown. The `just test-rom-audio` recipe runs each ROM `--no-gui`, lets it self-terminate (Blargg's final `jr $-2` triggers gobc's stuck-CPU detector), then reads `$A000-$A003` from the `.sav` to determine pass/fail.

## Hardware quirks implemented

| Test | Implementation | Reference |
|---|---|---|
| 03 | `apu_square.go` / `apu_wave.go` / `apu_noise.go` `writeNRx4` — captures `frameSeqStep`, applies the "first half" extra-clock rule, and the trigger-reload-to-max-1 edge case | SameBoy `Core/apu.c` lines 1885-1934 (square), 2015-2038 (wave), 2134-2157 (noise) |
| 08, 11 | `apu.go` `Write()` allows NR11/NR21/NR31/NR41 length-load writes while APU is off; each channel's `powerOff()` preserves `lengthCounter` and `lengthLoad` | SameBoy `Core/apu.c` lines 1719-1743 (NR52 power-off pulse_length snapshot/restore) |
| 09, 12 | `apu_wave.go` tracks `waveFormJustRead` (true only when the last APU tick chunk ended on a fetch cycle). `apu.go` `Read()`/`Write()` for $FF30-$FF3F return/ignore unless `waveFormJustRead` is set | SameBoy `Core/apu.c` lines 984-1003 (set/reset wave_form_just_read), 1129 (read returns 0xFF when false) |
| 10 | `apu_wave.go` `trigger()` runs the corruption copy when `periodTimer == 2` (our M-cycle-granularity equivalent of SameBoy's `sample_countdown == 0`); offset formula `(wavePos+1)/2` matches SameBoy | SameBoy `Core/apu.c` lines 1978-2003 |
