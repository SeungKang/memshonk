package hexdump

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const (
	defMaxRowLen    uint16 = 16
	defOffsetPadStr        = "8"
)

type Config struct {
	Src    io.Reader
	Dst    io.Writer
	Colors Colors

	OptTitle       string
	OptRowLen      uint16
	OptStartOffset uint64
	OptOffsetBits  uint8
}

func (o Config) OutputLen(totalInputBytes uint64) (int, int, error) {
	if o.OptRowLen == 0 {
		o.OptRowLen = defMaxRowLen
	}

	o.Src = bytes.NewReader(make([]byte, totalInputBytes))

	var out bytes.Buffer
	o.Dst = &out

	err := Dump(context.Background(), o)
	if err != nil {
		return 0, 0, err
	}

	return out.Len(), bytes.Count(out.Bytes(), []byte{'\n'}), nil
}

func Dump(ctx context.Context, config Config) error {
	bufR := bufio.NewReader(config.Src)
	bufW := bufio.NewWriter(config.Dst)
	defer bufW.Flush()

	maxRowLen := config.OptRowLen
	if maxRowLen == 0 {
		maxRowLen = defMaxRowLen
	}

	var offsetPadStr string
	if config.OptOffsetBits == 0 {
		offsetPadStr = defOffsetPadStr
	} else {
		// 32 == 8, 64 == 16.
		offsetPadStr = strconv.FormatUint(uint64(config.OptOffsetBits/4), 10)
	}

	data := make([]byte, maxRowLen)

	rowArgs := dumpRowArgs{
		writer:    bufW,
		dataCap:   cap(data),
		colors:    config.Colors,
		padOffCol: offsetPadStr,
		adjustOff: config.OptStartOffset,
	}

	if config.OptTitle != "" {
		_, err := bufW.WriteString(config.OptTitle + "\n")
		if err != nil {
			return err
		}
	}

	for {
		n, err := bufR.Read(data)

		if n > 0 {
			rowArgs.totalLen += uint64(n)
			rowArgs.data = data[0:n]

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
	colors    Colors
	totalLen  uint64
	data      []byte
	dataCap   int
	padOffCol string
	adjustOff uint64
}

func dumpRow(args dumpRowArgs) error {
	var s string

	dataLen := len(args.data)

	// offset section
	if args.totalLen > uint64(args.dataCap) {
		s = fmt.Sprintf("\n%0"+args.padOffCol+"x   ",
			(args.totalLen-uint64(dataLen))+args.adjustOff)
	} else {
		s = fmt.Sprintf("%0"+args.padOffCol+"x   ",
			args.adjustOff)
	}

	// hex characters section
	for i := 0; i < args.dataCap; i++ {
		if i < dataLen {
			s += args.colors.HexChar(args.data[i]) + " "
		} else {
			s += "   "
		}

		// this puts an extra space between the chunks in the hex column
		if (i+1)%4 == 0 {
			s += " "
		}
	}

	s += " |"

	// human-readable section
	for i := 0; i < args.dataCap; i++ {
		if i < dataLen {
			b := byte('.')

			if args.data[i] >= 0x21 && args.data[i] <= 0x7e {
				// Is ASCII (except space).
				b = args.data[i]
			}

			s += args.colors.Value(string(b), args.data[i])
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
