package windows

import (
	"math"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/cartridge"
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/gopxl/pixel/v2/pixelgl"
)

func (mw *MainGameWindow) _handleDebugInput() {
	if internalShowDebugInfo {
		if mw.Window.JustPressed(pixelgl.KeySpace) || mw.Window.Repeated(pixelgl.KeySpace) {
			internalGamePaused = !internalGamePaused
			if !internalGamePaused {
				mw.hw.Mb.GuiPause = false
			}
		}

		if (mw.Window.JustPressed(pixelgl.KeyN) || mw.Window.Repeated(pixelgl.KeyN)) && internalGamePaused {
			mw.hw.UpdateInternalGameState(internalDebugCyclePerFrame) // update every tick
		}

		if (mw.Window.JustPressed(pixelgl.KeyM) || mw.Window.Repeated(pixelgl.KeyM)) && internalGamePaused {
			internalDebugCycleScaler++
			internalDebugCyclePerFrame = int(math.Pow10(internalDebugCycleScaler))
		}

		if (mw.Window.JustPressed(pixelgl.KeyB) || mw.Window.Repeated(pixelgl.KeyB)) && internalGamePaused {
			internalDebugCycleScaler--
			if internalDebugCycleScaler < 0 {
				internalDebugCycleScaler = 0
			}
			internalDebugCyclePerFrame = int(math.Pow10(internalDebugCycleScaler))
		}

		if (mw.Window.JustPressed(pixelgl.KeyF) || mw.Window.Repeated(pixelgl.KeyF)) && internalGamePaused {
			mw.hw.UpdateInternalGameState(mw.cyclesFrame) // update every tick
			globalFrames++
		}
	}

	if mw.Window.JustPressed(pixelgl.KeyF1) || mw.Window.Repeated(pixelgl.KeyF1) {
		internalShowGrid = !internalShowGrid
	}

	if mw.Window.JustPressed(pixelgl.KeyF2) || mw.Window.Repeated(pixelgl.KeyF2) {
		internalShowDebugInfo = !internalShowDebugInfo
	}

	if (mw.Window.JustPressed(pixelgl.KeyF3) || mw.Window.Repeated(pixelgl.KeyF3)) && !mw.hw.Mb.Cgb {
		motherboard.ChangePallete()
	}

	if mw.Window.JustPressed(pixelgl.KeyF4) || mw.Window.Repeated(pixelgl.KeyF4) {
		cartridge.SaveSRAM(mw.hw.Mb.Cartridge.GetFilename(), &mw.hw.Mb.Cartridge.RamBanks, mw.hw.Mb.Cartridge.RamBankCount)
	}

	if mw.Window.JustPressed(pixelgl.KeyF5) || mw.Window.Repeated(pixelgl.KeyF5) {
		internal.StateToFile(mw.hw.Mb.Cartridge.GetFilename(), mw.hw.Mb)
	}

	if mw.Window.JustPressed(pixelgl.KeyF6) || mw.Window.Repeated(pixelgl.KeyF6) {
		internal.LoadState(mw.hw.Mb.Cartridge.GetFilename(), mw.hw.Mb)
	}
}

func (mw *MainGameWindow) _handleJoyPadInput() {
	/*
		KeyA = Button B
		KeyS = Button A
		KeyEnter = Start
		KeyBackspace = Select

		KeyUp = Up
		KeyDown = Down
		KeyLeft = Left
		KeyRight = Right
	*/

	if mw.Window.JustPressed(pixelgl.KeyEnter) || mw.Window.Repeated(pixelgl.KeyEnter) {
		mw.hw.Mb.ButtonEvent(motherboard.StartPress)
	}

	if mw.Window.JustReleased(pixelgl.KeyEnter) {
		mw.hw.Mb.ButtonEvent(motherboard.StartRelease)
	}

	if mw.Window.JustPressed(pixelgl.KeyRightShift) || mw.Window.Repeated(pixelgl.KeyRightShift) {
		mw.hw.Mb.ButtonEvent(motherboard.SelectPress)
	}

	if mw.Window.JustReleased(pixelgl.KeyRightShift) {
		mw.hw.Mb.ButtonEvent(motherboard.SelectRelease)
	}

	if mw.Window.JustPressed(pixelgl.KeyLeft) || mw.Window.Repeated(pixelgl.KeyLeft) {
		mw.hw.Mb.ButtonEvent(motherboard.LeftArrowPress)
	}

	if mw.Window.JustReleased(pixelgl.KeyLeft) {
		mw.hw.Mb.ButtonEvent(motherboard.LeftArrowRelease)
	}

	if mw.Window.JustPressed(pixelgl.KeyRight) || mw.Window.Repeated(pixelgl.KeyRight) {
		mw.hw.Mb.ButtonEvent(motherboard.RightArrowPress)
	}

	if mw.Window.JustReleased(pixelgl.KeyRight) {
		mw.hw.Mb.ButtonEvent(motherboard.RightArrowRelease)
	}

	if mw.Window.JustPressed(pixelgl.KeyUp) || mw.Window.Repeated(pixelgl.KeyUp) {
		mw.hw.Mb.ButtonEvent(motherboard.UpArrowPress)
	}

	if mw.Window.JustReleased(pixelgl.KeyUp) {
		mw.hw.Mb.ButtonEvent(motherboard.UpArrowRelease)
	}

	if mw.Window.JustPressed(pixelgl.KeyDown) || mw.Window.Repeated(pixelgl.KeyDown) {
		mw.hw.Mb.ButtonEvent(motherboard.DownArrowPress)
	}

	if mw.Window.JustReleased(pixelgl.KeyDown) {
		mw.hw.Mb.ButtonEvent(motherboard.DownArrowRelease)
	}

	if mw.Window.JustPressed(pixelgl.KeyA) || mw.Window.Repeated(pixelgl.KeyA) {
		mw.hw.Mb.ButtonEvent(motherboard.BPress)
	}

	if mw.Window.JustReleased(pixelgl.KeyA) {
		mw.hw.Mb.ButtonEvent(motherboard.BRelease)
	}

	if mw.Window.JustPressed(pixelgl.KeyS) || mw.Window.Repeated(pixelgl.KeyS) {
		mw.hw.Mb.ButtonEvent(motherboard.APress)
	}

	if mw.Window.JustReleased(pixelgl.KeyS) {
		mw.hw.Mb.ButtonEvent(motherboard.ARelease)
	}

}
