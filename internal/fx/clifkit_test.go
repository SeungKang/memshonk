package fx

import (
	"net"
	"testing"
	"time"
)

func TestNewFlagSet(t *testing.T) {
	fs := NewFlagSet("test")
	if fs == nil {
		t.Fatal("NewFlagSet returned nil")
	}
	if fs.Actual() == nil {
		t.Fatal("Actual() returned nil")
	}
}

func TestBoolFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var b bool
	fs.BoolFlag(&b, false, FlagConfig{Name: "verbose", Description: "verbose"})

	err := fs.Parse([]string{"-verbose"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !b {
		t.Error("expected b to be true")
	}
}

func TestBoolFlagShortAlias(t *testing.T) {
	fs := NewFlagSet("test")
	var b bool
	fs.BoolFlag(&b, false, FlagConfig{Name: "verbose", Description: "verbose"})

	err := fs.Parse([]string{"-v"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !b {
		t.Error("expected b to be true via short alias")
	}
}

func TestIntFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var n int
	fs.IntFlag(&n, 0, FlagConfig{Name: "count", Description: "count"})

	err := fs.Parse([]string{"-count", "42"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 42 {
		t.Errorf("expected n to be 42, got %d", n)
	}
}

func TestInt64Flag(t *testing.T) {
	fs := NewFlagSet("test")
	var n int64
	fs.Int64Flag(&n, 0, FlagConfig{Name: "size", Description: "size"})

	err := fs.Parse([]string{"-size", "9223372036854775807"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 9223372036854775807 {
		t.Errorf("expected max int64, got %d", n)
	}
}

func TestUintFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var n uint
	fs.UintFlag(&n, 0, FlagConfig{Name: "port", Description: "port"})

	err := fs.Parse([]string{"-port", "8080"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 8080 {
		t.Errorf("expected n to be 8080, got %d", n)
	}
}

func TestUint64Flag(t *testing.T) {
	fs := NewFlagSet("test")
	var n uint64
	fs.Uint64Flag(&n, 0, FlagConfig{Name: "bytes", Description: "bytes"})

	err := fs.Parse([]string{"-bytes", "18446744073709551615"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 18446744073709551615 {
		t.Errorf("expected max uint64, got %d", n)
	}
}

func TestStringFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var s string
	fs.StringFlag(&s, "", FlagConfig{Name: "name", Description: "name"})

	err := fs.Parse([]string{"-name", "test-value"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s != "test-value" {
		t.Errorf("expected s to be 'test-value', got %q", s)
	}
}

func TestFloat64Flag(t *testing.T) {
	fs := NewFlagSet("test")
	var f float64
	fs.Float64Flag(&f, 0, FlagConfig{Name: "rate", Description: "rate"})

	err := fs.Parse([]string{"-rate", "3.14159"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if f != 3.14159 {
		t.Errorf("expected f to be 3.14159, got %f", f)
	}
}

func TestDurationFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var d time.Duration
	fs.DurationFlag(&d, 0, FlagConfig{Name: "timeout", Description: "timeout"})

	err := fs.Parse([]string{"-timeout", "5s"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if d != 5*time.Second {
		t.Errorf("expected d to be 5s, got %v", d)
	}
}

func TestTextFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var ip net.IP
	fs.TextFlag(&ip, net.IPv4(127, 0, 0, 1), FlagConfig{
		Name:        "addr",
		Description: "address",
	})

	err := fs.Parse([]string{"-addr", "192.168.1.1"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := net.IPv4(192, 168, 1, 1)
	if !ip.Equal(expected) {
		t.Errorf("expected ip to be %v, got %v", expected, ip)
	}
}

func TestFuncFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var called bool
	var value string
	fs.FuncFlag(func(s string) error {
		called = true
		value = s
		return nil
	}, FlagConfig{Name: "callback", Description: "callback"})

	err := fs.Parse([]string{"-callback", "hello"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !called {
		t.Error("expected func to be called")
	}
	if value != "hello" {
		t.Errorf("expected value to be 'hello', got %q", value)
	}
}

func TestBoolFuncFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var called bool
	fs.BoolFuncFlag(func(s string) error {
		called = true
		return nil
	}, FlagConfig{Name: "help", Description: "help"})

	err := fs.Parse([]string{"-help"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !called {
		t.Error("expected func to be called")
	}
}

func TestRequiredFlagMissing(t *testing.T) {
	fs := NewFlagSet("test")
	var s string
	fs.StringFlag(&s, "", FlagConfig{
		Name:        "required",
		Description: "required flag",
		Required:    true,
	})

	err := fs.Parse([]string{})
	if err == nil {
		t.Fatal("expected error for missing required flag")
	}
}

func TestRequiredFlagProvided(t *testing.T) {
	fs := NewFlagSet("test")
	var s string
	fs.StringFlag(&s, "", FlagConfig{
		Name:        "required",
		Description: "required flag",
		Required:    true,
	})

	err := fs.Parse([]string{"-required", "value"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s != "value" {
		t.Errorf("expected s to be 'value', got %q", s)
	}
}

func TestShortAliasConflict(t *testing.T) {
	fs := NewFlagSet("test")
	var a, b bool
	fs.BoolFlag(&a, false, FlagConfig{Name: "verbose", Description: "verbose"})
	fs.BoolFlag(&b, false, FlagConfig{Name: "version", Description: "version"})

	// -v should match "verbose" since it was registered first
	err := fs.Parse([]string{"-v"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !a {
		t.Error("expected 'verbose' to be true via -v")
	}
	if b {
		t.Error("expected 'version' to be false")
	}
}

func TestSingleCharFlagNoShortAlias(t *testing.T) {
	fs := NewFlagSet("test")
	var b bool
	fs.BoolFlag(&b, false, FlagConfig{Name: "v", Description: "verbose"})

	// Single char flag should not create another alias
	err := fs.Parse([]string{"-v"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !b {
		t.Error("expected b to be true")
	}
}

func TestDefaultValue(t *testing.T) {
	fs := NewFlagSet("test")
	var n int
	fs.IntFlag(&n, 100, FlagConfig{Name: "count", Description: "count"})

	err := fs.Parse([]string{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 100 {
		t.Errorf("expected default value 100, got %d", n)
	}
}
