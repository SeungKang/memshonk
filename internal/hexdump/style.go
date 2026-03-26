package hexdump

import "encoding/hex"

type Style interface {
	SectionSpacing() string

	HexSection(data []byte, dataLen int, offsetBits uint8) (string, int)

	HasColors() (Colors, bool)
}

type DefaultStyle struct {
	Colors Colors
}

func (o DefaultStyle) SectionSpacing() string {
	return "  "
}

func (o DefaultStyle) HexSection(data []byte, dataLen int, offsetBits uint8) (string, int) {
	var s string
	charCount := 0

	for i := range data {
		if o.Colors == nil {
			s += hex.EncodeToString([]byte{data[i]})
		} else {
			s += o.Colors.HexChar(data[i]) + " "
		}

		charCount += 3

		// this puts an extra space between the chunks in the hex column
		if (i+1)%4 == 0 {
			s += " "

			charCount++
		}
	}

	return s, charCount
}

func (o DefaultStyle) HasColors() (Colors, bool) {
	return o.Colors, o.Colors != nil
}

type HeapStyle struct {
	Colors Colors
}

func (o HeapStyle) SectionSpacing() string {
	return "  "
}

func (o HeapStyle) HexSection(data []byte, dataLen int, offsetBits uint8) (string, int) {
	if dataLen >= 8 {
		SwapEndian(data[0:8])
	}

	if dataLen == 16 {
		SwapEndian(data[8:])
	}

	charCount := 0
	chunkLen := int(offsetBits / 8)

	var s string

	for i := 0; i < dataLen; i++ {
		if o.Colors == nil {
			s += hex.EncodeToString([]byte{data[i]})
		} else {
			s += o.Colors.HexChar(data[i])
		}

		charCount += 2

		// this puts an extra space between the chunks in the hex columns
		if (i+1)%chunkLen == 0 {
			charCount++
			s += " "
		}
	}

	return s, charCount
}

func (o HeapStyle) HasColors() (Colors, bool) {
	return o.Colors, o.Colors != nil
}

func SwapEndian(b []byte) {
	bLen := len(b)
	if bLen%2 != 0 {
		return
	}

	stop := (bLen / 2) - 1

	for i := range b {
		c := b[bLen-1-i]
		b[bLen-1-i] = b[i]
		b[i] = c
		if i == stop {
			return
		}
	}
}
