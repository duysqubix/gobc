# motherboard package

CPU + memory + timer + interrupts + LCD/PPU. The emulator's beating heart. Largest package by far — 14,598 LOC across 22 files; `opcodes.go` alone is 6,219 lines.

## STRUCTURE
```
motherboard/
├── motherboard.go            # Motherboard struct + NewMotherboard factory + Tick()
├── motherboard_getitem.go    # memory read dispatch (ROM → VRAM → ERAM → WRAM → OAM → IO → HRAM)
├── motherboard_setitem.go    # memory write dispatch
├── memory.go                 # WRAM / HRAM / VRAM / OAM / IO backing stores
├── cpu.go                    # CPU struct, Tick(), ExecuteInstruction, registers, flags, halt/stop state
├── opcodes.go                # OPCODES map: ALL ~500 opcodes (main 0x00-0xFF + CB-shifted 0x100-0x1FF)
├── timer.go                  # DIV / TIMA / TMA / TAC; raises Timer IRQ on TIMA overflow
├── interrupts.go             # IE / IF masks; priority VBlank > LCD > Timer > Serial > Joypad
├── lcd.go                    # PPU / LCD scanline rendering (583 LOC)
├── palettes.go               # CGB palette RAM + DMG palette cycling (F3 keybind)
└── root.go                   # OpCode, OpCycles, OpCodeMap types + package constants (incl. CB_SHIFT)
```

## CB-PREFIXED OPCODE ENCODING
Single `OPCODES` map holds everything. Lookup math:
- Raw `0x00`-`0xFF` → use the byte as the map key directly.
- After a `0xCB` prefix byte → call `OpCode(next).Shift()` which adds `CB_SHIFT` (0x100). So `RLC B` is `OPCODES[0x100]`, `BIT 7,A` is `OPCODES[0x17F]`, `SET 7,A` is `OPCODES[0x1FF]`.
- `executeOpcode(opcode, mb, value)` is the dispatcher. Illegal opcodes (see `ILLEGAL_OPCODES`) mark `cpu.IsStuck = true` and return 0 cycles.

## HARDWARE QUIRKS (the impl honors these)
- **F register lower nibble = 0**. `POP AF` masks the popped byte with `0xF0`. `Registers.SetAF` does NOT mask — direct callers can poison F.
- **`LD HL,SP+r8` (0xF8) / `ADD SP,r8` (0xE8)**: H and C flags computed on LOW-BYTE addition (not 16-bit). Common bug spot in other emulators.
- **`RLCA/RRCA/RLA/RRA`** (0x07/0x0F/0x17/0x1F): Z flag ALWAYS 0. Matches hardware; textbook erroneously says `Z = result==0`.
- **`HALT` (0x76) / `STOP` (0x10)**: do NOT advance PC.
- **`EI` (0xFB)**: delayed-enable. Sets `Interrupts.InterruptsEnabling = true`; master enable flips after the NEXT instruction.
- **`DI` (0xF3)**: immediate disable.
- **Illegal opcodes** (`0xD3, 0xDB, 0xDD, 0xE3, 0xE4, 0xEB, 0xEC, 0xED, 0xF4, 0xFC, 0xFD`): mark CPU stuck, dump state to stdout, return 0.
- **`DAA` (0x27)**: BCD adjust gated on N, H, C flags — see test cases in `opcodes_ctrl_test.go` for the truth table.

## TEST INFRASTRUCTURE
- **`newTestCPU(t)` in `cpu_test.go`** — shared helper returning `(*CPU, *Motherboard)`. REUSE; do NOT redeclare across test files.
- **`newMbForSubsysTest(t)` in `subsys_helpers_test.go`** — for timer / interrupts / memory tests. Namespaced `subsys*` so it doesn't collide with `newTestCPU`.
- **`internal.Logger.ExitFunc = func(int) {}`** is set in `init()` in `subsys_helpers_test.go` so `Logger.Fatalf` (called by cartridge loader on bad ROMs) doesn't kill the test binary. **Load-bearing — do not remove.**
- **`withSilencedStdout(fn)`** swaps `os.Stdout` to `/dev/null` around `NewMotherboard` — that call prints the cart header table unconditionally and would crowd `go test -v` output.

## TEST FILES BY DOMAIN (each owns a disjoint opcode class)
- `cpu_test.go` — registers, flags, stack, ALU on A (ADD/SUB/AND/OR/XOR/CP/ADC/SBC/INC/DEC)
- `opcodes_test.go` — opcode-table sanity + smoke-fire every handler
- `opcodes_ld8_test.go` — 8-bit LD family (~80 opcodes)
- `opcodes_ld16_test.go` — 16-bit LD + PUSH/POP rr + ADD HL,rr + ADD SP,r8 + LD HL,SP+r8 + INC/DEC rr
- `opcodes_ctrl_test.go` — JR/JP cond + CALL/RET cond + RST + RLCA/RRCA/RLA/RRA + DAA + CCF + HALT + EI/DI
- `opcodes_cb_test.go` — ALL 256 CB-prefixed
- `timer_test.go` / `interrupts_test.go` / `memory_test.go` — subsystem tests

## ANTI-PATTERNS
- **DO NOT call `cpu.Registers.SetAF(0xFFFF)` and expect F's lower nibble = 0** — only `POP AF` masks. `SetAF` writes through. Hardware quirk.
- **DO NOT add a test file that re-declares `newTestCPU` or `newMbForSubsysTest`** — Go rejects duplicate functions at the package scope.
- **DO NOT mutate `OPCODES` at runtime** — `executeOpcode` indexes directly; the race detector flags any write.
- **DO NOT trust textbook flag rules for `ADD SP,r8` / `LD HL,SP+r8`** — read the actual handler before asserting H/C.
- **DO NOT add direct `os.Stdout` printing in production code** — `cartridge.Dump` already does this, and tests must wrap with `withSilencedStdout`. New println noise leaks into `go test -v`.

## HARDWARE REFERENCE — query Pan Docs via Context7
Before adding / fixing an opcode, PPU mode, timer behavior, or interrupt-routing detail, query Pan Docs (see root AGENTS.md). Example queries that work well for this package:
- `"DAA instruction — exact BCD adjustment rules for N=0 and N=1 with all H/C combinations"`
- `"OAM DMA — cycle count and accessibility restrictions during transfer"`
- `"HALT bug — when does PC fail to increment after HALT?"`
- `"PPU mode 3 timing — how do scroll, window, and sprites affect mode-3 length?"`
- `"Timer TIMA overflow — exact cycle delay before reload from TMA and IRQ raise"`
- `"EI instruction — single-instruction delay before IME becomes active"`

Verify the handler in `opcodes.go` against the doc result, then add a regression test that pins the documented behavior.
