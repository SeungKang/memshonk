package goterm

import (
	"bytes"
	"fmt"
	"testing"
)

func TestBox(t *testing.T) {
	boxSample := `
┌--------┐
│ hello  │
│ world  │
│ test   │
└--------┘`

	out := bytes.NewBuffer(nil)

	screen := NewScreen(NewVirtualTerminal(VirtualTerminalConfig{
		Input:  bytes.NewBuffer(nil),
		Output: out,
	}))

	box, err := NewBox(10, 5, 0, screen)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Fprint(box, "hello i'm very long string\nworld\ntest")

	if box.String() != boxSample[1:] {
		t.Error("\n" + box.String())
		t.Error("!=")
		t.Error(boxSample)
		t.Error(len(box.String()), len(boxSample))
	}
}

func TestBox_WithUnicode(t *testing.T) {
	boxSample := `
┌--------┐
│ hell☺  │
│ w©rld  │
│ test✓✓ │
└--------┘`

	out := bytes.NewBuffer(nil)

	screen := NewScreen(NewVirtualTerminal(VirtualTerminalConfig{
		Input:  bytes.NewBuffer(nil),
		Output: out,
	}))

	box, err := NewBox(10, 5, 0, screen)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Fprint(box, "hell☺\nw©rld\ntest✓✓")

	if box.String() != boxSample[1:] {
		t.Error("\n" + box.String())
		t.Error("!=")
		t.Error(boxSample)
	}
}
