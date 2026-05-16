# cartridge package

Game Boy ROM header parser + Memory Bank Controller (MBC) implementations. 10 files, 3,245 LOC.

## STRUCTURE
```
cartridge/
├── cartridge.go    # NewCartridge factory, Cartridge struct, CartridgeTypeMap, header parsing, Dump/RawHeaderDump/DumpInstructionSet
├── root.go         # constants (HEADER_START_ADDR, TITLE_*, CBG_FLAG_ADDR, …), header types
├── rom_only.go     # type 0x00 — no banking, fixed 32 KiB ROM
├── mbc1.go         # types 0x01-0x03 — 5-bit ROM bank + 2-bit upper-RAM-or-ROM via mode select
├── mbc3.go         # types 0x0F-0x13 — 7-bit ROM bank + RTC (real-time clock)
├── mbc5.go         # types 0x19-0x1E — 9-bit ROM bank, NO bank-0-becomes-1 quirk
└── rtc.go          # MBC3's RTC: SEC/MIN/HOUR/DL/DH registers + latch state machine
```

## ENTRY POINT
`NewCartridge(path *pathlib.Path) *Cartridge` — opens the file via afero, parses byte `0x147` to pick the MBC, dispatches through `CartridgeTypeMap`, validates header checksum + ROM size against file size.

Unsupported types call `internal.Logger.Fatalf("unsupported cartridge type")` which `os.Exit(1)`s. Tests neutralize this via `internal.Logger.ExitFunc = func(int) {}`.

## HARDWARE QUIRKS (verified in tests)
- **MBC1 bank-0-becomes-1**: write `0x00` to `0x2000-0x3FFF` → bank 1 selected (NOT 0). To address bank `0x21`: write upper bits `0b01` to `0x4000-0x5FFF` FIRST, then write `0x01` to the lower reg → effective bank `(1<<5)|1 = 0x21`.
- **MBC1 mode select** (`0x6000-0x7FFF`): mode 0 = upper bits are RAM bank; mode 1 = upper bits extend ROM bank.
- **MBC3 RTC latch** (`0x6000-0x7FFF`): write `0x00` → `0x01` in sequence latches current real-time into the RTC registers. Any other sequence resets the latch state.
- **RTC bit masks**: SEC/MIN are 6-bit (0-59), HOUR is 5-bit (0-23), DL (day low) is 8-bit, DH (day high) bit 0 = day MSB, bit 6 = halt, bit 7 = day-carry.
- **MBC5**: 9-bit ROM bank = low 8 bits at `0x2000-0x2FFF` + bit 8 at `0x3000-0x3FFF`. NO bank-0 quirk — writing 0 selects bank 0.
- **RAM enable**: every MBC requires `0x0A` written to `0x0000-0x1FFF` before RAM at `0xA000-0xBFFF` is accessible. Any other value disables (reads return `0xFF`, writes dropped).

## CONVENTIONS
- All MBCs implement the same shape: `Read(addr uint16) byte`, `Write(addr uint16, value byte)`. Banking state lives on the MBC struct, accessed via the Cartridge.
- Header constants live in `root.go` — reuse them instead of hard-coding offsets.
- SRAM save/load goes through `cartridge.SaveSRAM` / `cartridge.LoadSRAM` (filename derived from ROM path).

## ANTI-PATTERNS
- **DO NOT add a new cartridge type without registering it in `CartridgeTypeMap`** — unregistered byte values at `0x147` trigger `Logger.Fatalf` and kill the process.
- **DO NOT bypass `Read`/`Write` and poke `RomBanks` / `RamBanks` arrays directly outside MBC code** — banking state will go out of sync with the bus.
- **DO NOT instantiate Cartridge from a `*os.File`** — the constructor takes a `*pathlib.Path` (afero-backed). Tests synthesize a minimal in-memory ROM via `os.CreateTemp` + `pathlib.NewPath(name, pathlib.PathWithAfero(afero.NewOsFs()))`.

## TEST CONVENTIONS
- `cartridge_test.go` — header parsing (title, CGB flag, SGB flag, MBC type detection, ROM/RAM size from bytes 0x148/0x149, checksums).
- `mbc_test.go` — bank-switching behavior for all 4 MBC implementations + the bank-0 / RAM-enable / latch quirks.
- `rtc_test.go` — RTC register masks + latch state machine.
- All tests use a `makeFakeROM` helper that computes a valid header checksum — `NewCartridge` Fatalfs on invalid checksum, so synthetic ROMs MUST honor the math (see Pan Docs for the checksum formula).

## HARDWARE REFERENCE — query Pan Docs via Context7
Before adding a new MBC type or fixing banking behavior, query Pan Docs (see root AGENTS.md). Example queries that work well for this package:
- `"MBC1 register map — banking mode select 0x6000-0x7FFF, simple vs advanced mode"`
- `"MBC3 RTC latching sequence and which registers are exposed at 0xA000-0xBFFF"`
- `"MBC5 ROM bank register — 9-bit bank number across 0x2000-0x2FFF and 0x3000-0x3FFF"`
- `"Cartridge header — byte 0x147 cartridge type values and what hardware each implies"`
- `"Cartridge header — byte 0x148 ROM size codes 0x00..0x08 and resulting bank count"`
- `"Cartridge header checksum — exact formula for bytes 0x134..0x14C verifying against byte 0x14D"`
- `"MBC2 — built-in 512×4-bit RAM and how addressing differs from external SRAM"` (when implementing MBC2)
- `"HuC1 / HuC3 / TAMA5 / Pocket Camera — banking semantics"` (when implementing exotic MBCs)
