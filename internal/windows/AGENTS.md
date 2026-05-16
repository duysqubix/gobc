# windows package

Pixel/GLFW-backed GUI. One main game window + 5 debug viewer windows (VRAM, Memory, Cart, CPU, IO). 9 files, 1,535 LOC.

## STRUCTURE
```
windows/
├── windows.go             # Window interface (SetUp / Update / Draw / Finalize / Win) + ParseBreakpoints utility
├── mainview.go            # MainGameWindow + GoBoyColor (the emulator handle) + IsDebugInfo / SetDebugInfo
├── mainview_input.go      # F-keys + joypad + debug stepping (Space/N/B/M/F)
├── vramview.go            # tile + tilemap visualizer; keys T, B, G, H, V
├── memoryview.go          # WRAM/HRAM hex viewer; arrows + mouse-wheel scroll
├── cartview.go            # cartridge RAM viewer; brackets [ / ] navigate banks
├── cpuview.go             # CPU register display
├── ioview.go              # I/O register table
└── root.go                # updatePicture() helper for tile rendering
```

## SHARED PACKAGE-LEVEL STATE
These are PACKAGE-LEVEL VARIABLES (declared in `mainview.go` top) read/written by multiple windows:
- `internalGamePaused`, `internalShowGrid`, `internalShowDebugInfo`
- `internalDebugCyclePerFrame`, `internalDebugCycleScaler`
- `internalCycleCounter`, `globalCycles`, `globalFrames`
- `internalConsoleTxt` (the text overlay handle)

**Bridge to `cmd/gobc`**: the only public accessors are `IsDebugInfo()` / `SetDebugInfo(v bool)`. `gameLoopGUI` in `cmd/gobc/gobc.go` polls `IsDebugInfo()` each frame and LAZILY creates the 5 viewer windows the first time it flips true. So pressing F2 from a `--debug`-less startup also opens the viewers.

## PIXEL/V2 VERSION PIN
Locked to **v2.2.1**, not v2.3.0. Upstream commit `36af8f43` reverted a text-anchor fix in v2.3.0 → debug viewer windows render NO text. **Do not bump to v2.3.x** until upstream re-fixes it. See the top-level AGENTS.md "ANTI-PATTERNS" note.

## KEYBINDINGS SOURCE OF TRUTH
Key handlers live in `mainview_input.go` and the per-viewer `<name>view.go` files. The **help text** lives in `cmd/gobc/gobc.go` as the `keybindingsHelp` constant. The constant has a `// Must stay in sync with internal/windows/*.go input handlers.` comment as a maintenance hint — **when you add or rename a key binding, update BOTH sides.**

## ANTI-PATTERNS
- **DO NOT add new top-level package vars in this package** unless you have a great reason — the existing shared state is already a kept-for-legacy anti-pattern. New state goes on a window struct.
- **DO NOT call GLFW/OpenGL functions outside the main thread** — pixel/v2 requires main-thread execution, locked via `runtime.LockOSThread()` in `cmd/gobc/gobc.go` `init()`.
- **DO NOT create new windows from inside an input handler** — defer to the next frame of `gameLoopGUI` so GLFW state stays consistent.
- **DO NOT use `pathlib.NewPathAfero(path, fs)`** — deprecated in pathlib v0.19+. Use `pathlib.NewPath(path, pathlib.PathWithAfero(fs))` (staticcheck SA1019 enforces).
- **DO NOT add text rendering without testing in the GUI** — pixel/v2's text package has been the source of two regressions; manual visual QA is mandatory.
