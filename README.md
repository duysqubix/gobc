<p align="center">
<img src="docs/gobc_logo.png" width="480">
</p>

<p align="center">
<a href="https://github.com/duysqubix/gobc/actions/workflows/go.yml"><img src="https://github.com/duysqubix/gobc/actions/workflows/go.yml/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/duysqubix/gobc"><img src="https://codecov.io/gh/duysqubix/gobc/branch/master/graph/badge.svg" alt="Coverage"></a>
<a href="https://goreportcard.com/report/github.com/duysqubix/gobc"><img src="https://goreportcard.com/badge/github.com/duysqubix/gobc" alt="Go Report Card"></a>
<a href="https://pkg.go.dev/github.com/duysqubix/gobc"><img src="https://pkg.go.dev/badge/github.com/duysqubix/gobc.svg" alt="Go Reference"></a>
</p>

__If you have any questions, or just want to chat, [join us on Discord](https://discord.gg/EVCX5X3A)__


Standing on the shoulders of giants, this is a fun project to create, _yet another_, GameBoy emulator. 
Checkout [TESTS.md](https://github.com/duysqubix/gobc/blob/master/TESTS.md) for the ever growing tests GoBC passes.

<!---
Generate GIF with the layout and captions
-->
<table>
  <tbody>
      <tr>
      <td align="center">
        <img src="docs/crystal.gif" width="200">
      </td>
      <td align="center">
        <img src="docs/blue.gif" width="200">
      </td>
            <td align="center">
        <img src="docs/tetris.gif" width="200">
      </td>
    </tr>
      <tr>
      <td align="center">
        <img src="docs/dr_mario.png" width="200">
      </td>
          <td align="center">
        <img src="docs/bgb.gif" width="200">
      </td>
          <td align="center">
        <img src="docs/loz.gif" width="200">
      </td>
    </tr>
    <tr>
    </tr>
  </tbody>
</table>

Supported Features
==================
- [x] CPU
- [ ] Sound
- [x] Interrupts
- [x] Joypad
- [ ] Serial
- [ ] Cartridges
    - [x] Cartridge RAM_ONLY
    - [x] Cartridge MBC1
    - [ ] Cartridge MBC2
    - [x] Cartridge MBC3
      - [x] RTC   
    - [x] Cartridge MBC5
    - [ ] Cartridge HuC1
    - [ ] Cartridge HuC3
    - [ ] Cartridge MMM01
    - [ ] Cartridge Pocket Camera
    - [ ] Cartridge Bandai TAMA5
    - [ ] Cartridge Hudson HuC-1
    - [ ] Cartridge Hudson HuC-3
- [x] Graphics 
- [x] Timers
- [x] CGB Mode
- [x] Save States
- [x] Load States
- [ ] Debugger
  - [x] VRAM
  - [x] Disassembler
  - [x] IO
  - [x] TileMaps
  - [x] Tile Data
- [ ] Shaders

Future Features
==================
List is in no particular order and is subject to change. Take a stab at any of them if you feel like it.

- [ ] Sound
- [ ] Use real Cartridges (via Arduino)
- [ ] Implement all MBCs

Known Issues
==================
- [ ] Fix Links Awakening DMG (Menu is broken)
- [ ] Pokemon Crystal - Battle Scenes Not animating
- [ ] Pokemon Crystal - Character is split upon battle entry



Installing GoBC
============
You need the following dependencies installed:
- [Go](https://golang.org/doc/install) (1.26+)
- [OpenGL](https://github.com/gopxl/pixel#requirements)

First-time setup
```bash
# Install just itself (one of):
cargo install just         # or: brew install just / apt install just

just bootstrap             # installs Go (if missing) + gopls + staticcheck + dlv + goimports
just install-hooks         # wires up the repo-tracked pre-commit hook (gofmt + vet + staticcheck)
just                       # list all recipes
```

Building from source
```bash
just build              # vet + test + compile -> bin/gobc + bin/cartdump
just compile            # compile only (no tests) for fast iteration
just build-release      # stripped + trimpath release build
# or directly:
go build -o bin/gobc ./cmd/gobc
go build -o bin/cartdump ./cmd/cartdump
```

> The project uses [`just`](https://github.com/casey/just) as its task runner. `just bootstrap` overrides: `GO_VERSION=1.27.0 just bootstrap` (pick a specific Go release), `GO_INSTALL_DIR=$HOME/.local just bootstrap` (install without sudo).

Or install the latest binary
```bash
go install github.com/duysqubix/gobc/cmd/gobc@latest
```

Usage
========

`gobc` exposes two subcommands plus a backward-compatible shorthand:

| Command | Purpose |
|---|---|
| `gobc run ROM_File [options]` | Boot the emulator and run a ROM. |
| `gobc cartdump ROM_File [options]` | Dump cartridge header metadata (and optional disassembly). |
| `gobc ROM_File [options]` | Shorthand for `gobc run ROM_File`. |

Run `gobc --help` for the full keybinding reference and `gobc <command> --help` for
per-command flags.

```bash
# run a ROM
gobc run roms/cpu_instrs.gb
gobc run roms/cpu_instrs.gb --debug --breakpoints 0x100,0x200,0x300
gobc run roms/pokemon.gb --force-cgb        # force CGB mode on a DMG ROM
gobc run roms/blargg.gb --no-gui            # headless (CI / test ROMs)
LOG_LEVEL=debug gobc run roms/zelda.gb      # raise log verbosity

# inspect a cartridge
gobc cartdump roms/pokemon.gb                                     # write cartdump.txt
gobc cartdump --raw roms/pokemon.gb                               # print raw header to stdout
gobc cartdump --instruction-set --include-nop -o dump.txt roms/pokemon.gb
```
LOG_LEVEL=info gobc roms/cpu_instrs.gb # Set Log level to info
```

Key Bindings
============
Main Window
-----------
| Key | Description |
| --- | ----------- |
| `F1` | Toggle Grid |
| `F2` | Toggle Debug Information |
| `F3` | Change Pallete (DMG Only) |
| `F4` | Save Cartridge SRAM |
| `F5` | Save State |
| `F6` | Load State |
| `A` | Button B |
| `S` | Button A |
| `Enter` | Start |
| `Shift` | Select |
| `Up` | D-Pad Up |
| `Down` | D-Pad Down |
| `Left` | D-Pad Left |
| `Right` | D-Pad Right |
| `DEBUG ON - Space Bar` | Game Pause |
| `DEBUG ON - N` | Step N Cycles |
| `DEBUG ON - M` | Increase Cycles Per Frame 10x |
| `DEBUG ON - B` | Decrease Cycles Per Frame 10x |
| `DEBUG ON - F` | Step Frame |

VRAM Window
-----------
| Key | Description |
| --- | ----------- |
| `T` | Toggle Tile Addressing Mode|
| `B` | Toggle TileMap Addressing Mode|
| `G` | Toggle Grid |
| `V` | Toggle VRAM bank 0/1 |

Memory Window
-----------
| Key | Description |
| --- | ----------- |
| `Right` | Page Down |
| `Left` | Page Up |
| `Up` | Scroll Up |
| `Down` | Scroll Down |
| `Mouse Wheel Up` | Scroll Up |
| `Mouse Wheel Down` | Scroll Down |

Cart Window
-----------
| Key | Description |
| --- | ----------- |
| `Right` | Page Down |
| `Left` | Page Up |
| `Up` | Scroll Up |
| `Down` | Scroll Down |
| `Mouse Wheel Up` | Scroll Up |
| `Mouse Wheel Down` | Scroll Down |


Testing
=======

Unit tests run with the standard `go test` toolchain plus the race detector. ROM-based
integration tests (Blargg, Mooneye) live under `default_rom/` and are executed in CI by
booting `gobc --no-gui` against each ROM.

```bash
# unit tests
just test                # go test -race ./...
just test-cover          # writes coverage.out + prints total
just test-cover-html     # generates coverage.html
COVER_MIN=70 just test-cover-check   # fails if coverage < 70%

# benchmarks
just bench

# ROM integration tests (mirrors CI)
./bin/gobc --no-gui default_rom/blarrg/cpu_instrs/cpu_instrs.gb
./bin/gobc --no-gui default_rom/blarrg/instr_timing/instr_timing.gb
```

CI uploads coverage to [Codecov](https://codecov.io/gh/duysqubix/gobc) on every push;
see `codecov.yml` for thresholds and ignored paths.


Contributors
============

Thanks to all the people who have contributed to the project and we welcome anyone to 
join and help out!

Original Developers
-------------------

 * Duan Uys- [duysqubix](https://github.com/duysqubix)



Contribute
==========
Any contribution is appreciated. The currently known problems are tracked in the Issues tab. Feel free to take a swing at any one of them.

If you want to implement something which is not on the list, feel free to do so anyway. If you want to merge it into our repo, then just send a pull request and we will have a look at it.

Resources and References
========

[Pan Docs](https://gbdev.io/pandocs/About.html)

[GBEDG](https://hacktix.github.io/GBEDG/)

[Instruction Set](https://www.pastraiser.com/cpu/gameboy/gameboy_opcodes.html)

[PyBoy](https://github.com/Baekalfen/PyBoy/tree/master) <-- Huge Inspiration

[Test Roms Archive](https://gbdev.gg8.se/wiki/articles/Test_ROMs)
