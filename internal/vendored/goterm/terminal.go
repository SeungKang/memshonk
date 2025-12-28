// Provides basic bulding blocks for advanced console UI
//
// Coordinate system:
//
//	1/1---X---->
//	 |
//	 Y
//	 |
//	 v
//
// Documentation for ANSI codes: http://en.wikipedia.org/wiki/ANSI_escape_code#Colors
//
// Inspired by: http://www.darkcoding.net/software/pretty-command-line-console-output-on-unix-in-python-and-go-lang/
package goterm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// Reset all custom styles
const RESET = "\033[0m"

// Reset to default color
const RESET_COLOR = "\033[32m"

// Return cursor to start of line and clean it
const RESET_LINE = "\r\033[K"

// List of possible colors
const (
	BLACK = iota
	RED
	GREEN
	YELLOW
	BLUE
	MAGENTA
	CYAN
	WHITE
)

// Set percent flag: num | PCT
//
// Check percent flag: num & PCT
//
// Reset percent flag: num & 0xFF
const shift = uint(^uint(0)>>63) << 4
const PCT = 0x8000 << shift

func NewStdioTerminal() (*StdioTerminal, error) {
	return &StdioTerminal{}, nil
}

type StdioTerminal struct {
}

func (o StdioTerminal) Input() io.Reader {
	return os.Stdin
}

func (o StdioTerminal) Output() io.Writer {
	return os.Stdout
}

func (o StdioTerminal) Size() (Size, error) {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return Size{}, err
	}

	return Size{
		Cols: width,
		Rows: height,
	}, nil
}

type VirtualTerminalConfig struct {
	Input   io.Reader
	Output  io.Writer
	OptSize Size
}

func NewVirtualTerminal(config VirtualTerminalConfig) *VirtualTerminal {
	if config.OptSize.Cols == 0 && config.OptSize.Rows == 0 {
		config.OptSize = DefaultVirtualTerminalSize()
	}

	return &VirtualTerminal{config: config}
}

type VirtualTerminal struct {
	rwMu   sync.RWMutex
	config VirtualTerminalConfig
}

func (o *VirtualTerminal) Input() io.Reader {
	return o.config.Input
}

func (o *VirtualTerminal) Output() io.Writer {
	return o.config.Output
}

func (o *VirtualTerminal) Size() (Size, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.config.OptSize, nil
}

func (o *VirtualTerminal) SetSize(size Size) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	o.config.OptSize = size
}

type Terminal interface {
	Input() io.Reader

	Output() io.Writer

	Size() (Size, error)
}

func DefaultVirtualTerminalSize() Size {
	return Size{Cols: 80, Rows: 24}
}

type Size struct {
	Cols int // Width.
	Rows int // Height.
}

func NewStdioScreen() (*Screen, error) {
	terminal, err := NewStdioTerminal()
	if err != nil {
		return nil, err
	}

	return NewScreen(terminal), nil
}

func NewScreen(terminal Terminal) *Screen {
	return &Screen{
		terminal: terminal,
		output:   bufio.NewWriter(terminal.Output()),
		screen:   bytes.NewBuffer(nil),
	}
}

type Screen struct {
	terminal Terminal
	output   *bufio.Writer
	screen   *bytes.Buffer
}

// GetXY gets relative or absolute coordinates
// To get relative, set PCT flag to number:
//
//	// Get 10% of total width to `x` and 20 to y
//	x, y = tm.GetXY(10|tm.PCT, 20)
func (o *Screen) GetXY(x int, y int) (int, int, error) {
	if y == -1 {
		y = o.CurrentHeight() + 1
	}

	size, err := o.terminal.Size()
	if err != nil {
		return 0, 0, err
	}

	if x&PCT != 0 {
		x = int((x & 0xFF) * size.Cols / 100)
	}

	if y&PCT != 0 {
		y = int((y & 0xFF) * size.Rows / 100)
	}

	return x, y, nil
}

type sf func(int, string) string

// Apply given transformation func for each line in string
func applyTransform(str string, transform sf) (out string) {
	out = ""

	for idx, line := range strings.Split(str, "\n") {
		out += transform(idx, line)
	}

	return
}

// Clear screen
func (o *Screen) Clear() {
	o.output.WriteString("\033[2J")
}

// Move cursor to given position
func (o *Screen) MoveCursor(x int, y int) {
	fmt.Fprintf(o.screen, "\033[%d;%dH", y, x)
}

// Move cursor up relative the current position
func (o *Screen) MoveCursorUp(bias int) {
	fmt.Fprintf(o.screen, "\033[%dA", bias)
}

// Move cursor down relative the current position
func (o *Screen) MoveCursorDown(bias int) {
	fmt.Fprintf(o.screen, "\033[%dB", bias)
}

// Move cursor forward relative the current position
func (o *Screen) MoveCursorForward(bias int) {
	fmt.Fprintf(o.screen, "\033[%dC", bias)
}

// Move cursor backward relative the current position
func (o *Screen) MoveCursorBackward(bias int) {
	fmt.Fprintf(o.screen, "\033[%dD", bias)
}

// Move string to possition
func (o *Screen) MoveTo(str string, x int, y int) (string, error) {
	var err error

	x, y, err = o.GetXY(x, y)
	if err != nil {
		return "", err
	}

	return applyTransform(str, func(idx int, line string) string {
		return fmt.Sprintf("\033[%d;%dH%s", y+idx, x, line)
	}), nil
}

// Width gets console width
func (o *Screen) Width() (int, error) {
	ws, err := o.terminal.Size()
	if err != nil {
		return 0, err
	}

	return int(ws.Cols), nil
}

// CurrentHeight gets current height. Line count in Screen buffer.
func (o *Screen) CurrentHeight() int {
	return strings.Count(o.screen.String(), "\n")
}

// Flush buffer and ensure that it will not overflow screen
func (o *Screen) Flush() error {
	for idx, str := range strings.SplitAfter(o.screen.String(), "\n") {
		size, err := o.terminal.Size()
		if err != nil {
			return err
		}

		if idx > size.Rows {
			return nil
		}

		o.output.WriteString(str)
	}

	// TODO: Here
	o.output.Flush()

	o.screen.Reset()

	return nil
}

func (o *Screen) Print(a ...interface{}) (n int, err error) {
	return fmt.Fprint(o.screen, a...)
}

func (o *Screen) Println(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(o.screen, a...)
}

func (o *Screen) Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(o.screen, format, a...)
}

func Context(data string, idx, max int) string {
	var start, end int

	if len(data[:idx]) < (max / 2) {
		start = 0
	} else {
		start = idx - max/2
	}

	if len(data)-idx < (max / 2) {
		end = len(data) - 1
	} else {
		end = idx + max/2
	}

	return data[start:end]
}

// ResetLine returns carrier to start of line
func ResetLine(str string) (out string) {
	return applyTransform(str, func(idx int, line string) string {
		return fmt.Sprintf("%s%s", RESET_LINE, line)
	})
}

// Make bold
func Bold(str string) string {
	return applyTransform(str, func(idx int, line string) string {
		return fmt.Sprintf("\033[1m%s\033[0m", line)
	})
}

// Apply given color to string:
//
//	tm.Color("RED STRING", tm.RED)
func Color(str string, color int) string {
	return applyTransform(str, func(idx int, line string) string {
		return fmt.Sprintf("%s%s%s", getColor(color), line, RESET)
	})
}

func Highlight(str, substr string, color int) string {
	hiSubstr := Color(substr, color)
	return strings.Replace(str, substr, hiSubstr, -1)
}

func HighlightRegion(str string, from, to, color int) string {
	return str[:from] + Color(str[from:to], color) + str[to:]
}

// Change background color of string:
//
//	tm.Background("string", tm.RED)
func Background(str string, color int) string {
	return applyTransform(str, func(idx int, line string) string {
		return fmt.Sprintf("%s%s%s", getBgColor(color), line, RESET)
	})
}

func getColor(code int) string {
	return fmt.Sprintf("\033[3%dm", code)
}

func getBgColor(code int) string {
	return fmt.Sprintf("\033[4%dm", code)
}
