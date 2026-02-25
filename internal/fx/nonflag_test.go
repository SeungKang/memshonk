package fx

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIntNf(t *testing.T) {
	fs := NewFlagSet("test")
	var n int
	fs.IntNf(&n, ArgConfig{Name: "count", Description: "count"})

	err := fs.Parse([]string{"42"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 42 {
		t.Errorf("expected n to be 42, got %d", n)
	}
}

func TestInt64Nf(t *testing.T) {
	fs := NewFlagSet("test")
	var n int64
	fs.Int64Nf(&n, ArgConfig{Name: "size", Description: "size"})

	err := fs.Parse([]string{"9223372036854775807"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 9223372036854775807 {
		t.Errorf("expected max int64, got %d", n)
	}
}

func TestUintNf(t *testing.T) {
	fs := NewFlagSet("test")
	var n uint
	fs.UintNf(&n, ArgConfig{Name: "port", Description: "port"})

	err := fs.Parse([]string{"8080"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 8080 {
		t.Errorf("expected n to be 8080, got %d", n)
	}
}

func TestUint64Nf(t *testing.T) {
	fs := NewFlagSet("test")
	var n uint64
	fs.Uint64Nf(&n, ArgConfig{Name: "bytes", Description: "bytes"})

	err := fs.Parse([]string{"18446744073709551615"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if n != 18446744073709551615 {
		t.Errorf("expected max uint64, got %d", n)
	}
}

func TestStringNf(t *testing.T) {
	fs := NewFlagSet("test")
	var s string
	fs.StringNf(&s, ArgConfig{Name: "name", Description: "name"})

	err := fs.Parse([]string{"hello"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s != "hello" {
		t.Errorf("expected s to be 'hello', got %q", s)
	}
}

func TestFloat64Nf(t *testing.T) {
	fs := NewFlagSet("test")
	var f float64
	fs.Float64Nf(&f, ArgConfig{Name: "rate", Description: "rate"})

	err := fs.Parse([]string{"3.14159"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if f != 3.14159 {
		t.Errorf("expected f to be 3.14159, got %f", f)
	}
}

func TestDurationNf(t *testing.T) {
	fs := NewFlagSet("test")
	var d time.Duration
	fs.DurationNf(&d, ArgConfig{Name: "timeout", Description: "timeout"})

	err := fs.Parse([]string{"5s"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if d != 5*time.Second {
		t.Errorf("expected d to be 5s, got %v", d)
	}
}

func TestMultipleNonflags(t *testing.T) {
	fs := NewFlagSet("test")
	var src, dst string
	fs.StringNf(&src, ArgConfig{Name: "source", Description: "source"})
	fs.StringNf(&dst, ArgConfig{Name: "dest", Description: "destination"})

	err := fs.Parse([]string{"file1.txt", "file2.txt"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if src != "file1.txt" {
		t.Errorf("expected src to be 'file1.txt', got %q", src)
	}
	if dst != "file2.txt" {
		t.Errorf("expected dst to be 'file2.txt', got %q", dst)
	}
}

func TestNonflagWithFlags(t *testing.T) {
	fs := NewFlagSet("test")
	var verbose bool
	var file string
	fs.BoolFlag(&verbose, false, ArgConfig{Name: "verbose", Description: "v"})
	fs.StringNf(&file, ArgConfig{Name: "file", Description: "file"})

	err := fs.Parse([]string{"-verbose", "myfile.txt"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !verbose {
		t.Error("expected verbose to be true")
	}
	if file != "myfile.txt" {
		t.Errorf("expected file to be 'myfile.txt', got %q", file)
	}
}

func TestNonflagRequired(t *testing.T) {
	fs := NewFlagSet("test")
	var s string
	fs.StringNf(&s, ArgConfig{
		Name:        "file",
		Description: "file",
		Required:    true,
	})

	err := fs.Parse([]string{})
	if err == nil {
		t.Fatal("expected error for missing required nonflag")
	}
	if !strings.Contains(err.Error(), "required argument not provided") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNonflagOptional(t *testing.T) {
	fs := NewFlagSet("test")
	var s string
	fs.StringNf(&s, ArgConfig{
		Name:        "file",
		Description: "file",
		Required:    false,
	})

	err := fs.Parse([]string{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s != "" {
		t.Errorf("expected s to be empty, got %q", s)
	}
}

func TestNonflagInvalidValue(t *testing.T) {
	fs := NewFlagSet("test")
	var n int
	fs.IntNf(&n, ArgConfig{Name: "count", Description: "count"})

	err := fs.Parse([]string{"not-a-number"})
	if err == nil {
		t.Fatal("expected error for invalid int value")
	}
	if !strings.Contains(err.Error(), "invalid value") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestIntSliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []int
	fs.IntSliceNf(&vals, ArgConfig{Name: "numbers", Description: "numbers"})

	err := fs.Parse([]string{"1", "2", "3", "4", "5"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestInt64SliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []int64
	fs.Int64SliceNf(&vals, ArgConfig{Name: "sizes", Description: "sizes"})

	err := fs.Parse([]string{"1000000000000", "2000000000000"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []int64{1000000000000, 2000000000000}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestUintSliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []uint
	fs.UintSliceNf(&vals, ArgConfig{Name: "ports", Description: "ports"})

	err := fs.Parse([]string{"80", "443", "8080"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []uint{80, 443, 8080}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestUint64SliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []uint64
	fs.Uint64SliceNf(&vals, ArgConfig{Name: "bytes", Description: "bytes"})

	err := fs.Parse([]string{"100", "200", "300"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []uint64{100, 200, 300}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestStringSliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []string
	fs.StringSliceNf(&vals, ArgConfig{Name: "files", Description: "files"})

	err := fs.Parse([]string{"a.txt", "b.txt", "c.txt"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []string{"a.txt", "b.txt", "c.txt"}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestFloat64SliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []float64
	fs.Float64SliceNf(&vals, ArgConfig{Name: "rates", Description: "rates"})

	err := fs.Parse([]string{"1.5", "2.5", "3.5"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []float64{1.5, 2.5, 3.5}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestDurationSliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []time.Duration
	fs.DurationSliceNf(&vals, ArgConfig{Name: "timeouts", Description: "t"})

	err := fs.Parse([]string{"1s", "2m", "3h"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []time.Duration{time.Second, 2 * time.Minute, 3 * time.Hour}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestSliceNfConsumesRemaining(t *testing.T) {
	fs := NewFlagSet("test")
	var cmd string
	var args []string
	fs.StringNf(&cmd, ArgConfig{Name: "command", Description: "command"})
	fs.StringSliceNf(&args, ArgConfig{Name: "args", Description: "arguments"})

	err := fs.Parse([]string{"echo", "hello", "world"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if cmd != "echo" {
		t.Errorf("expected cmd to be 'echo', got %q", cmd)
	}
	expected := []string{"hello", "world"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v, got %v", expected, args)
	}
}

func TestSliceNfEmpty(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []string
	fs.StringSliceNf(&vals, ArgConfig{Name: "files", Description: "files"})

	err := fs.Parse([]string{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("expected empty slice, got %v", vals)
	}
}

func TestSliceNfInvalidValue(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []int
	fs.IntSliceNf(&vals, ArgConfig{Name: "numbers", Description: "numbers"})

	err := fs.Parse([]string{"1", "two", "3"})
	if err == nil {
		t.Fatal("expected error for invalid int value")
	}
	if !strings.Contains(err.Error(), "invalid value") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFlagsAndSliceNf(t *testing.T) {
	fs := NewFlagSet("test")
	var verbose bool
	var count int
	var files []string
	fs.BoolFlag(&verbose, false, ArgConfig{Name: "verbose", Description: "v"})
	fs.IntFlag(&count, 1, ArgConfig{Name: "count", Description: "count"})
	fs.StringSliceNf(&files, ArgConfig{Name: "files", Description: "files"})

	err := fs.Parse([]string{"-verbose", "-count", "5", "a.txt", "b.txt"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !verbose {
		t.Error("expected verbose to be true")
	}
	if count != 5 {
		t.Errorf("expected count to be 5, got %d", count)
	}
	expected := []string{"a.txt", "b.txt"}
	if !reflect.DeepEqual(files, expected) {
		t.Errorf("expected %v, got %v", expected, files)
	}
}

func TestMixedNonflags(t *testing.T) {
	fs := NewFlagSet("test")
	var src string
	var port int
	var dst string
	fs.StringNf(&src, ArgConfig{Name: "source", Description: "source"})
	fs.IntNf(&port, ArgConfig{Name: "port", Description: "port"})
	fs.StringNf(&dst, ArgConfig{Name: "dest", Description: "dest"})

	err := fs.Parse([]string{"localhost", "8080", "remote"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if src != "localhost" {
		t.Errorf("expected src to be 'localhost', got %q", src)
	}
	if port != 8080 {
		t.Errorf("expected port to be 8080, got %d", port)
	}
	if dst != "remote" {
		t.Errorf("expected dst to be 'remote', got %q", dst)
	}
}
