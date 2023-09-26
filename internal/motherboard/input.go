package motherboard

import "github.com/duysqubix/gobc/internal"

const (
	P10 uint8 = iota
	P11
	P12
	P13
)

type Key uint8

const (
	RightArrowPress Key = iota
	RightArrowRelease
	LeftArrowPress
	LeftArrowRelease
	UpArrowPress
	UpArrowRelease
	DownArrowPress
	DownArrowRelease
	APress
	ARelease
	BPress
	BRelease
	StartPress
	StartRelease
	SelectPress
	SelectRelease
)

type Input struct {
	directional uint8
	standard    uint8
}

func NewInput() *Input {
	return &Input{
		directional: 0x0F,
		standard:    0x0F,
	}
}

func (i *Input) KeyEvent(key Key) uint8 {
	prevDirectional := i.directional
	prevStandard := i.standard

	switch key {
	case RightArrowPress:
		internal.ResetBit(&i.directional, P10)
	case LeftArrowPress:
		internal.ResetBit(&i.directional, P11)
	case UpArrowPress:
		internal.ResetBit(&i.directional, P12)
	case DownArrowPress:
		internal.ResetBit(&i.directional, P13)

	case APress:
		internal.ResetBit(&i.standard, P10)
	case BPress:
		internal.ResetBit(&i.standard, P11)
	case SelectPress:
		internal.ResetBit(&i.standard, P12)
	case StartPress:
		internal.ResetBit(&i.standard, P13)

	case RightArrowRelease:
		internal.SetBit(&i.directional, P10)
	case LeftArrowRelease:
		internal.SetBit(&i.directional, P11)
	case UpArrowRelease:
		internal.SetBit(&i.directional, P12)
	case DownArrowRelease:
		internal.SetBit(&i.directional, P13)

	case ARelease:
		internal.SetBit(&i.standard, P10)
	case BRelease:
		internal.SetBit(&i.standard, P11)
	case SelectRelease:
		internal.SetBit(&i.standard, P12)
	case StartRelease:
		internal.SetBit(&i.standard, P13)
	default:
		logger.Fatalf("Unknown key event: %v", key)
	}

	return ((prevDirectional ^ i.directional) & i.directional) | ((prevStandard ^ i.standard) & i.standard)
}

func (i *Input) Pull(joystickbyte uint8) uint8 {
	P14 := (joystickbyte >> 4) & 0x01
	P15 := (joystickbyte >> 5) & 0x01
	// # Bit 7 - Not used (No$GMB)
	// # Bit 6 - Not used (No$GMB)
	// # Bit 5 - P15 out port
	// # Bit 4 - P14 out port
	// # Bit 3 - P13 in port
	// # Bit 2 - P12 in port
	// # Bit 1 - P11 in port
	// # Bit 0 - P10 in port

	joystickByte := 0xFF & (joystickbyte | 0b11001111)
	if P14 != 0 && P15 != 0 {
		return 0
	} else if P14 == 0 && P15 == 0 {
		return 0
	} else if P14 == 0 {
		joystickByte &= i.directional
	} else if P15 == 0 {
		joystickByte &= i.standard
	}
	return joystickByte
}
