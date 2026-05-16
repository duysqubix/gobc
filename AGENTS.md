# gobc

**Generated:** 2026-05-15
**Commit:** af12f56
**Branch:** master

## OVERVIEW
Go Game Boy / Game Boy Color emulator. Built on **pixel/v2** (GLFW/OpenGL) for GUI and **urfave/cli/v2** for the CLI. Cycle accuracy verified against Blargg + Mooneye test ROMs in CI.

## STRUCTURE
```
gobc/
├── cmd/
│   ├── gobc/             # main binary; urfave/cli/v2 app with `run` + `cartdump` subcommands
│   └── cartdump/         # standalone cartridge-dump binary (same logic as `gobc cartdump`)
├── internal/
│   ├── motherboard/      # CPU + opcodes (6219 LOC) + memory + timer + interrupts + PPU
│   ├── cartridge/        # header parser + ROM_ONLY / MBC1 / MBC3+RTC / MBC5
│   ├── windows/          # Pixel/GLFW GUI: 1 main + 5 viewer windows (VRAM/Memory/Cart/CPU/IO)
│   ├── bootrom/          # DMG + CGB boot ROMs as hex blobs
│   └── root.go           # shared utilities: Logger, constants, bit-ops, state save/load
├── default_rom/          # test ROMs: blarrg/, mooneye_test_suite/
├── .githooks/            # repo-tracked git hooks (installed via `just install-hooks`)
├── .github/workflows/    # CI: unit tests + coverage + ROM integration
└── justfile              # task runner (run `just` to list recipes)
```

## WHERE TO LOOK
| Task | Location |
|---|---|
| Add a new opcode | `internal/motherboard/opcodes.go` (CB-prefixed at +`CB_SHIFT` = 0x100 offset) |
| Implement a new MBC type | `internal/cartridge/mbcN.go` + register in `CartridgeTypeMap` in `cartridge.go` |
| Add a new debug viewer window | `internal/windows/<name>view.go`, then open in `openDebugWindows()` in `cmd/gobc/gobc.go` |
| Change keybindings | `internal/windows/mainview_input.go` AND `keybindingsHelp` in `cmd/gobc/gobc.go` (must stay in sync) |
| Tweak CLI flags | `cmd/gobc/gobc.go` (`runFlags` shared by root + `run`; cartdump has its own) |
| Add a build/test task | `justfile` |
| Add a ROM integration test in CI | `.github/workflows/go.yml` `integration-rom-tests` job |

## CONVENTIONS
- **Task runner**: `just`, not `make`. Run `just` (no args) to list recipes.
- **Pre-commit hook**: gofmt + vet + staticcheck on staged `.go` files via `.githooks/pre-commit`. Enable via `just install-hooks`.
- **Tests**: stdlib `testing` + `testify/assert` + `testify/require`. No gomock/moq/ginkgo.
- **Coverage gate**: `COVER_MIN=N just test-cover-check` fails if total < N%.
- **Logger**: every package uses `internal.Logger` (logrus singleton). `internal.Logger.ExitFunc` is overridden in tests so `Logger.Fatalf` doesn't kill the test binary.
- **CLI**: `gobc ROM_File` is the shorthand for `gobc run ROM_File` (backward compat). All flags must appear BEFORE the positional ROM arg.

## ANTI-PATTERNS (THIS PROJECT)
- **NEVER bump pixel/v2 ≥ v2.3.0** — upstream commit `36af8f43` reverted the text-anchor fix; debug viewer windows render no text. Pinned to **v2.2.1**.
- **NEVER add `var logger = internal.Logger` in new packages** — staticcheck flags the local alias as unused (U1000). Use `internal.Logger` directly.
- **DO NOT call `cli.ShowAppHelpAndExit(ctx, 1)`** — informational help should exit 0 so shell tooling (`just`, CI scripts) don't misreport.
- **DO NOT call `os.Exit` directly** — bypasses the test `ExitFunc` hook. Use `internal.Logger.Fatalf` instead.
- **DO NOT commit with `--no-verify`** — bypasses pre-commit gate. Fix issues instead.

## UNIQUE STYLES
- **CB-prefixed opcode encoding**: All opcodes live in the SAME `OPCODES` map. Main 0x00-0xFF use their raw key; CB-prefixed use `OpCode(byte).Shift()` which adds `CB_SHIFT` (0x100). So `RLC B` lives at key `0x100`, `BIT 7,A` at `0x17F`.
- **Hardware quirks honored**:
  - F register lower nibble forced to 0 on `POP AF` (Game Boy hardware)
  - MBC1 bank-0-becomes-1 substitution (write 0 to ROM bank reg → bank 1)
  - MBC3 RTC latch sequence: write `0x00` → `0x01` to `0x6000-0x7FFF`
  - `LD HL,SP+r8` / `ADD SP,r8` H/C flags computed on LOW BYTE addition (NOT 16-bit) — common mis-implementation
  - `RLCA/RRCA/RLA/RRA`: Z flag always 0 (matches hardware, NOT textbook)
- **Shared package-level globals in `internal/windows/`** bridge to `cmd/gobc` via the only public accessors: `windows.IsDebugInfo()` / `windows.SetDebugInfo(bool)`.

## COMMANDS
```bash
just bootstrap            # install Go (if missing) + gopls + staticcheck + dlv + goimports
just install-hooks        # enable repo-tracked pre-commit hook
just                      # list all recipes
just run <ROM_File>       # compile + run
just build                # vet + race-test + compile
just build-release        # stripped (-trimpath -s -w) release build (~24% smaller)
just build-debug          # dlv-friendly debug build (-gcflags=all=-N -l)
just test-cover-html      # generate coverage.html
just lint                 # vet + staticcheck
COVER_MIN=70 just test-cover-check   # CI-style coverage gate
```

## HARDWARE REFERENCE (for AI agents)
The canonical Game Boy hardware reference is **Pan Docs** ([gbdev/pandocs](https://github.com/gbdev/pandocs)). Available via the Context7 MCP as library ID `/gbdev/pandocs` (729 indexed snippets, High source reputation). **Query this BEFORE assuming hardware behavior** — beats textbook references and third-party opcode tables, which are commonly wrong on quirks (DAA, ADD SP,r8 flag math, OAM bug, HALT bug, PPU mode timing, MBC1 banking, RTC latching).

```
mcp_Context7_query-docs(
  libraryId="/gbdev/pandocs",
  query="<specific hardware question — e.g. 'OAM DMA timing and access restrictions during transfer'>"
)
```

Use it when:
- Adding/fixing a CPU opcode (especially flag semantics)
- Implementing a new MBC type
- Touching PPU / LCD STAT / scanline timing
- Hitting an unexpected ROM behavior — check Pan Docs FIRST before patching emulation

## NOTES
- The `gobc` binary at the repo root (~13.5M) is committed for convenience; `just build` regenerates it.
- ROM integration tests in CI boot `gobc --no-gui` against Blargg ROMs; the binary writes test pass/fail to serial out which CI greps for "Passed".
- Test ROMs live under `default_rom/blarrg/` + `default_rom/mooneye_test_suite/`. Crystal/Pokemon ROMs at repo root are NOT for CI; they're developer convenience for manual smoke tests.
- `internal.VERSION = "1.3"` is the canonical version string surfaced by `gobc --version`. Bump it on releases.
