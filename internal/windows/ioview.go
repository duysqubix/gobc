package windows

import (
	"fmt"
	"strconv"

	"github.com/duysqubix/gobc/internal"
	"github.com/duysqubix/gobc/internal/motherboard"
	"github.com/duysqubix/pixel2"
	"github.com/duysqubix/pixel2/pixelgl"
	"github.com/duysqubix/pixel2/text"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

const (
	ioScreenWidth  = 1200
	ioScreenHeight = 670
	ioScale        = 1
	ioTrueWidth    = float64(ioScreenWidth * ioScale)
	ioTrueHeight   = float64(ioScreenHeight * ioScale)
)

var (
	ioDefaultFont *basicfont.Face
	ioConsoleTxt  *text.Text
	ioTableWriter *tablewriter.Table
)

func init() {

	// set up font
	ioDefaultFont = basicfont.Face7x13

}

type IoViewWindow struct {
	hw      *GoBoyColor
	YOffset float64
	Window  *pixelgl.Window
}

func NewIoViewWindow(gobc *GoBoyColor) *IoViewWindow {
	/// create memory window
	memWin, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:       fmt.Sprintf("gobc v%s | IO View", internal.VERSION),
		Bounds:      pixel.R(0, 0, ioScreenWidth, ioTrueHeight),
		VSync:       true,
		AlwaysOnTop: true,
		Resizable:   true,
	})

	if err != nil {
		logger.Panicf("Failed to create window: %s", err)
	}
	return &IoViewWindow{
		Window:  memWin,
		YOffset: 0,
		hw:      gobc,
	}
}

func (mw *IoViewWindow) Win() *pixelgl.Window {
	return mw.Window
}

func (mw *IoViewWindow) SetUp() {
	mw.Window.SetBounds(pixel.R(0, 0, ioTrueWidth, ioTrueHeight))
	ioConsoleTxt = text.New(
		pixel.V(10, 5),
		text.NewAtlas(ioDefaultFont, text.ASCII),
		// text.NewAtlas(inconsolata.Regular8x16, text.ASCII),
	)
	ioConsoleTxt.Color = colornames.Cyan

	ioTableWriter = tablewriter.NewWriter(ioConsoleTxt)
	ioTableWriter.SetAutoWrapText(false)
	ioTableWriter.SetAlignment(tablewriter.ALIGN_LEFT)
	ioTableWriter.SetRowSeparator("-")
	ioTableWriter.SetBorder(false)
}

func (mw *IoViewWindow) Finalize() {
	mw.Window.Update()
}

func (mw *IoViewWindow) Update() error {

	return nil
}

func formatBitValue(value uint8) string {
	bin := strconv.FormatUint(uint64(value), 2)
	for len(bin) < 8 {
		bin = "0" + bin
	}
	return "(" + bin[:4] + " " + bin[4:] + ")"
}

func (mw *IoViewWindow) Draw() {
	ioConsoleTxt.Clear()
	mw.Window.Clear(colornames.Black)
	ioTableWriter.ClearRows()

	statIO := mw.hw.Mb.Memory.IO[0x41] & 0x3
	var statMode string
	switch statIO {
	case motherboard.STAT_MODE_HBLANK:
		statMode = "H-Blank"
	case motherboard.STAT_MODE_VBLANK:
		statMode = "V-Blank"
	case motherboard.STAT_MODE_OAM:
		statMode = "OAM"
	default:
		statMode = "Transfer"
	}

	ioTableWriter.AppendBulk(
		[][]string{
			{"INTERRUPTS:", "IME:", fmt.Sprintf("%t", mw.hw.Mb.Cpu.Interrupts.InterruptsOn), "", "LCD:"},
			//IE
			{"$FFFF", "IE", fmt.Sprintf("$%02x", mw.hw.Mb.Cpu.Interrupts.IE), formatBitValue(mw.hw.Mb.Cpu.Interrupts.IE), "$FF40", "LCDC", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x40]), formatBitValue(mw.hw.Mb.Memory.IO[0x40])},
			// IF
			{"$FF0F", "IF", fmt.Sprintf("$%02x", mw.hw.Mb.Cpu.Interrupts.IF), formatBitValue(mw.hw.Mb.Cpu.Interrupts.IF), "$FF41", "STAT", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x41]), formatBitValue(mw.hw.Mb.Memory.IO[0x41])},
			append(mw.hw.Mb.Cpu.Interrupts.ReportOn(0), "$FF42", "SCY", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x42]), formatBitValue(mw.hw.Mb.Memory.IO[0x42])),
			append(mw.hw.Mb.Cpu.Interrupts.ReportOn(1), "$FF43", "SCX", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x43]), formatBitValue(mw.hw.Mb.Memory.IO[0x43])),
			append(mw.hw.Mb.Cpu.Interrupts.ReportOn(2), "$FF44", "LY", fmt.Sprintf("%d", mw.hw.Mb.Memory.IO[0x44]), formatBitValue(mw.hw.Mb.Memory.IO[0x44])),
			append(mw.hw.Mb.Cpu.Interrupts.ReportOn(3), "$FF45", "LYC", fmt.Sprintf("%d", mw.hw.Mb.Memory.IO[0x45]), formatBitValue(mw.hw.Mb.Memory.IO[0x45])),
			append(mw.hw.Mb.Cpu.Interrupts.ReportOn(4), "$FF46", "DMA", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x46]), formatBitValue(mw.hw.Mb.Memory.IO[0x46])),
			{"", "", "", "", "$FF47", "BGP", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x47]), formatBitValue(mw.hw.Mb.Memory.IO[0x47])},
			{"GBC:", "", "", "", "$FF48", "OBP0", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x48]), formatBitValue(mw.hw.Mb.Memory.IO[0x48])},
			{"$FF4D", "KEY1", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x4D]), formatBitValue(mw.hw.Mb.Memory.IO[0x4D]), "$FF49", "OBP1", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x49]), formatBitValue(mw.hw.Mb.Memory.IO[0x49])},
			{"$FF70", "SVBK", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x70]), formatBitValue(mw.hw.Mb.Memory.IO[0x70]), "$FF4A", "WY", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x4A]), formatBitValue(mw.hw.Mb.Memory.IO[0x4A])},
			{"", "", "", "", "$FF4B", "WX", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x4B]), formatBitValue(mw.hw.Mb.Memory.IO[0x4B])},
			{"GBC LCD:", "", "", "", "TIMER:"},
			{"$FF68", "BCPS", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x68]), formatBitValue(mw.hw.Mb.Memory.IO[0x68]), "$FF04", "DIV", fmt.Sprintf("$%02x", uint8(mw.hw.Mb.Timer.DIV)), formatBitValue(uint8(mw.hw.Mb.Timer.DIV))},
			{"$FF69", "BCPD", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x69]), formatBitValue(mw.hw.Mb.Memory.IO[0x69]), "$FF05", "TIMA", fmt.Sprintf("$%02x", mw.hw.Mb.Timer.TIMA), formatBitValue(uint8(mw.hw.Mb.Timer.TIMA))},
			{"$FF6A", "OCPS", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x6A]), formatBitValue(mw.hw.Mb.Memory.IO[0x6A]), "$FF06", "TMA", fmt.Sprintf("$%02x", mw.hw.Mb.Timer.TMA), formatBitValue(uint8(mw.hw.Mb.Timer.TMA))},
			{"$FF6B", "OCPD", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x6B]), formatBitValue(mw.hw.Mb.Memory.IO[0x6B]), "$FF07", "TAC", fmt.Sprintf("$%02x", mw.hw.Mb.Timer.TAC), formatBitValue(uint8(mw.hw.Mb.Timer.TAC))},
			{"$FF4F", "VBK", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x4F]), formatBitValue(mw.hw.Mb.Memory.IO[0x4F]), "Input:"},
			{"", "", "", "", "$FF00", "JOYP", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x00]), formatBitValue(mw.hw.Mb.Memory.IO[0x00])},
			{"GBC HDMA:", "", "", "", "SERIAL:"},
			{"$FF51:$FF52", "Source", fmt.Sprintf("$%02x%02x", mw.hw.Mb.Memory.IO[0x51], mw.hw.Mb.Memory.IO[0x52]), "", "$FF01", "SB", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x01]), formatBitValue(mw.hw.Mb.Memory.IO[0x01])},
			{"$FF53:$FF54", "Dest", fmt.Sprintf("$%02x%02x", mw.hw.Mb.Memory.IO[0x53], mw.hw.Mb.Memory.IO[0x54]), "", "$FF02", "SC", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x02]), formatBitValue(mw.hw.Mb.Memory.IO[0x02])},
			{"$FF55", "LEN", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x55]), formatBitValue(mw.hw.Mb.Memory.IO[0x55]), ""},
			{"", "", "", "", "LCDC Flags:"},
			append(append([]string{"GBC INFRARED", "", "", ""}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_ENABLE, "ON", "OFF")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_WINEN, "ON", "OFF")...),
			append(append([]string{"$FF56", "RP", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x56]), formatBitValue(mw.hw.Mb.Memory.IO[0x56])}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_BGMAP, "$8000", "$8800")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_WINMAP, "$9C00", "$9800")...),
			append(append([]string{"", "", "", ""}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_OBJEN, "ON", "OFF")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_BGEN, "ON", "OFF")...),
			append(append([]string{"STAT Flags:", "", "", ""}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_OBJSZ, "8x8", "8x16")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_BGWIN, "$9C00", "$9800")...),
			append(mw.hw.Mb.Lcd.ReportOnSTAT(motherboard.STAT_LYCINT), mw.hw.Mb.Lcd.ReportOnSTAT(motherboard.STAT_OAMINT)...),
			append(mw.hw.Mb.Lcd.ReportOnSTAT(motherboard.STAT_VBLINT), mw.hw.Mb.Lcd.ReportOnSTAT(motherboard.STAT_HBLINT)...),
			append(mw.hw.Mb.Lcd.ReportOnSTAT(motherboard.STAT_LYC), []string{"Mode", statMode}...),
		},
	)
	// append(append([]string{"GBC INFRARED", "", "", ""}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_ENABLE, "ON", "OFF")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_WINMAP, "$9C00", "$9800")...),
	// append(append([]string{"$FF56", "RP", fmt.Sprintf("$%02x", mw.hw.Mb.Memory.IO[0x56]), formatBitValue(mw.hw.Mb.Memory.IO[0x56])}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_WINEN, "ON", "OFF")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_BGMAP, "$8000", "$8800")...),
	// append(append([]string{"", "", "", ""}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_BGWIN, "$9C00", "$9800")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_OBJSZ, "8x8", "8x16")...),
	// append(append([]string{"STAT Flags:", "", "", ""}, mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_OBJEN, "ON", "OFF")...), mw.hw.Mb.Lcd.ReportOnLCDC(motherboard.LCDC_BGEN, "ON", "OFF")...),
	ioTableWriter.Render()
	ioConsoleTxt.Draw(mw.Window, pixel.IM.Scaled(ioConsoleTxt.Orig, 1.5))
}
