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

	OptRowLen    uint16
	OptStartOff  uint64
	OptOffColPad uint8
}

func Dump(ctx context.Context, config Config) error {
	bufR := bufio.NewReader(config.Src)
	bufW := bufio.NewWriter(config.Dst)
	defer bufW.Flush()

	maxRowLen := config.OptRowLen
	if maxRowLen == 0 {
		maxRowLen = 16
	}

	var offsetPadStr string
	if config.OptOffColPad == 0 {
		offsetPadStr = "8"
	} else {
		offsetPadStr = strconv.FormatUint(uint64(config.OptOffColPad), 10)
	}

	rowArgs := dumpRowArgs{
		writer:    bufW,
		row:       make([]byte, maxRowLen),
		maxRowLen: maxRowLen,
		colors:    config.Colors,
		padOffCol: offsetPadStr,
		adjustOff: config.OptStartOff,
	}

	for {
		n, err := bufR.Read(rowArgs.row)

		if n > 0 {
			rowArgs.totalLen += uint64(n)
			rowArgs.rowLen = uint16(n)

			dErr := dumpRow(rowArgs)
			if dErr != nil {
				return dErr
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}
	}
}

type dumpRowArgs struct {
	writer    io.Writer
	maxRowLen uint16
	colors    Colors
	totalLen  uint64
	row       []byte
	rowLen    uint16
	padOffCol string
	adjustOff uint64
}

func dumpRow(args dumpRowArgs) error {
	// 1. 16
	//    total = 16
	//    rowLen = 16
	// 2. 8
	//    total = 24
	//    rowLen = 8
	var s string

	if args.totalLen > uint64(args.maxRowLen) {
		s = fmt.Sprintf("\n%0"+args.padOffCol+"x   ",
			(args.totalLen-uint64(args.rowLen))+args.adjustOff)
	} else {
		s = fmt.Sprintf("%0"+args.padOffCol+"x   ",
			args.adjustOff)
	}

	for i := uint16(0); i < args.maxRowLen; i++ {
		if i < args.rowLen {
			s += args.colors.HexChar(args.row[i]) + " "
		} else {
			s += "   "
		}

		if (i+1)%4 == 0 {
			s += " "
		}
	}

	s += " |"

	for i := uint16(0); i < args.maxRowLen; i++ {
		if i < args.rowLen {
			b := byte('.')

			if args.row[i] >= 0x21 && args.row[i] <= 0x7e {
				// Is ASCII (except space).
				b = args.row[i]
			}

			s += args.colors.Value(string(b), args.row[i])
		} else {
			s += " "
		}
	}

	s += "|"

	_, err := fmt.Fprint(args.writer, s)

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
