package hexdump

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type Config struct {
	Src    io.Reader
	Dst    io.Writer
	Colors Colors
}

func Dump(ctx context.Context, config Config) error {
	bufR := bufio.NewReader(config.Src)
	bufW := bufio.NewWriter(config.Dst)
	defer bufW.Flush()

	offset := int64(-1) // TODO
	row := make([]byte, 16)

	for {
		b, err := bufR.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return dumpRow(bufW, offset, row, config.Colors, false)
			}

			return err
		}

		offset++
		rowOffset := offset % 16
		row[rowOffset] = b

		if offset > 0 && rowOffset == 15 {
			err := dumpRow(bufW, offset, row, config.Colors, true)
			if err != nil {
				return err
			}
		}
	}
}

func dumpRow(w io.Writer, offset int64, row []byte, colors Colors, addNewline bool) error {
	rowLen := int(offset%16 + 1)

	s := fmt.Sprintf("%08x", offset-int64(rowLen)+1) + "   "

	for i := 0; i < 16; i++ {
		if i < rowLen {
			s += colors.HexChar(row[i]) + " "
		} else {
			s += "   "
		}
		if (i+1)%4 == 0 {
			s += " "
		}
	}

	s += " |"

	for i := 0; i < 16; i++ {
		if i < rowLen {
			b := byte('.')

			if row[i] >= 33 && row[i] <= 126 {
				b = row[i]
			}

			s += colors.Value(string(b), row[i])
		} else {
			s += " "
		}
	}

	if addNewline {
		s += "|\n"
	} else {
		s += "|"
	}

	_, err := fmt.Fprint(w, s)

	return err
}

func NewColors() Colors {
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

	return Colors{
		colors: colors,
	}
}

type Colors struct {
	colors [256]string
}

func (o Colors) Value(s string, b byte) string {
	// if b == 25 {
	// 	fmt.Println("\n\nSHIT", strings.ReplaceAll(colors[b], "\\", ">"), s)
	// }
	return o.colors[b] + s + "\033[0m"
}

func (o Colors) HexChar(b byte) string {
	//return fmt.Sprintf("%s%02x%s", colors[b], b, "\033[0m")

	return o.colors[b] + hex.EncodeToString([]byte{b}) + "\033[0m"
}
