package hexdump

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	defMaxRowLen    uint16 = 16
	defOffsetPadStr        = "8"
)

type Config struct {
	Src io.Reader
	Dst io.Writer

	OptStyle       Style
	OptTitle       string
	OptRowLen      uint16
	OptStartOffset uint64
	OptOffsetBits  uint8
}

func (o Config) OutputLen(totalInputBytes uint64) (totalBytes int, numNewLines int, _ error) {
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

	if config.OptStyle == nil {
		config.OptStyle = DefaultStyle{
			Colors: NewByteColors(),
		}
	}

	data := make([]byte, maxRowLen)

	optColors, _ := config.OptStyle.HasColors()

	rowArgs := &dumpRowArgs{
		writer:    bufW,
		dataCap:   cap(data),
		padOffCol: offsetPadStr,
		adjustOff: config.OptStartOffset,
		style:     config.OptStyle,
		optColors: optColors,
		sectionSp: config.OptStyle.SectionSpacing(),
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
	totalLen  uint64
	data      []byte
	dataCap   int
	padOffCol string
	adjustOff uint64
	style     Style
	optColors Colors
	hexSecEnd int
	sectionSp string
}

func dumpRow(args *dumpRowArgs) error {
	var s string

	dataLen := len(args.data)

	// offset section
	if args.totalLen > uint64(args.dataCap) {
		s = fmt.Sprintf("\n0x%0"+args.padOffCol+"x"+args.sectionSp,
			(args.totalLen-uint64(dataLen))+args.adjustOff)
	} else {
		s = fmt.Sprintf("0x%0"+args.padOffCol+"x"+args.sectionSp,
			args.adjustOff)
	}

	// hex characters section
	hexSection, count := args.style.HexSection(args.data, dataLen, 64)
	if args.hexSecEnd == 0 {
		args.hexSecEnd = count
	}

	if count < args.hexSecEnd {
		hexSection += strings.Repeat(" ", args.hexSecEnd-count)
	}

	s += hexSection

	s += "|"

	// human-readable section
	for i := 0; i < args.dataCap; i++ {
		if i < dataLen {
			b := byte('.')

			if args.data[i] >= 0x21 && args.data[i] <= 0x7e {
				// Is ASCII (except space).
				b = args.data[i]
			}

			if args.optColors == nil {
				s += string(b)
			} else {
				s += args.optColors.Value(string(b), args.data[i])
			}
		} else {
			s += " "
		}
	}

	s += "|"

	_, err := fmt.Fprint(args.writer, s)

	return err
}
