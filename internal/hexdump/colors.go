package hexdump

import (
	"encoding/hex"
	"strconv"
)

type Colors interface {
	// HexChar returns the hex-encoded string of a byte with
	// coloring applied.
	HexChar(b byte) string

	// Value colors the human-readable representation of a byte.
	// The byte itself is passed in case a lookup is required.
	Value(humanReadableRepresentation string, b byte) string
}

func NewByteColors() ByteColors {
	var colors [256]string

	const WHITE_B = "\033[1;37m"

	for i := 0; i < 256; i++ {
		var fg, bg string

		// colors that are very hard to read on a dark background
		barelyVisible := i == 0 || (i >= 16 && i <= 20) || (i >= 232 && i <= 242)

		if barelyVisible {
			fg = WHITE_B + "\033[38;5;" + "255" + "m"
			bg = "\033[48;5;" + strconv.Itoa(int(i)) + "m"

		} else {
			fg = WHITE_B + "\033[38;5;" + strconv.Itoa(int(i)) + "m"
			bg = ""
		}

		colors[i] = bg + fg
	}

	return ByteColors{
		colors: colors,
	}
}

type ByteColors struct {
	colors [256]string
}

func (o ByteColors) HexChar(b byte) string {
	//return fmt.Sprintf("%s%02x%s", colors[b], b, "\033[0m")

	return o.colors[b] + hex.EncodeToString([]byte{b}) + "\033[0m"
}

func (o ByteColors) Value(humanReadableRep string, b byte) string {
	// if b == 25 {
	// 	fmt.Println("\n\nSHIT", strings.ReplaceAll(colors[b], "\\", ">"), s)
	// }
	return o.colors[b] + humanReadableRep + "\033[0m"
}
