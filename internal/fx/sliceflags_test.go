package fx

import (
	"reflect"
	"testing"
	"time"
)

func TestIntSliceFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []int
	fs.IntSliceFlag(&vals, ArgConfig{Name: "num", Description: "numbers"})

	err := fs.Parse([]string{"-num", "1", "-num", "2", "-num", "3"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestIntSliceFlagShortAlias(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []int
	fs.IntSliceFlag(&vals, ArgConfig{Name: "num", Description: "numbers"})

	err := fs.Parse([]string{"-n", "10", "-n", "20"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []int{10, 20}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestInt64SliceFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []int64
	fs.Int64SliceFlag(&vals, ArgConfig{Name: "size", Description: "sizes"})

	err := fs.Parse([]string{"-size", "1000000000000", "-size", "2000000000000"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []int64{1000000000000, 2000000000000}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestUintSliceFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []uint
	fs.UintSliceFlag(&vals, ArgConfig{Name: "port", Description: "ports"})

	err := fs.Parse([]string{"-port", "80", "-port", "443", "-port", "8080"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []uint{80, 443, 8080}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestUint64SliceFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []uint64
	fs.Uint64SliceFlag(&vals, ArgConfig{Name: "bytes", Description: "bytes"})

	err := fs.Parse([]string{"-bytes", "18446744073709551615"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []uint64{18446744073709551615}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestStringSliceFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []string
	fs.StringSliceFlag(&vals, ArgConfig{Name: "tag", Description: "tags"})

	err := fs.Parse([]string{"-tag", "foo", "-tag", "bar", "-tag", "baz"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []string{"foo", "bar", "baz"}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestFloat64SliceFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []float64
	fs.Float64SliceFlag(&vals, ArgConfig{Name: "rate", Description: "rates"})

	err := fs.Parse([]string{"-rate", "1.5", "-rate", "2.5", "-rate", "3.5"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []float64{1.5, 2.5, 3.5}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestDurationSliceFlag(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []time.Duration
	fs.DurationSliceFlag(&vals, ArgConfig{
		Name:        "timeout",
		Description: "timeouts",
	})

	err := fs.Parse([]string{"-timeout", "1s", "-timeout", "2m", "-timeout", "3h"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	expected := []time.Duration{
		1 * time.Second,
		2 * time.Minute,
		3 * time.Hour,
	}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("expected %v, got %v", expected, vals)
	}
}

func TestSliceFlagEmpty(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []string
	fs.StringSliceFlag(&vals, ArgConfig{Name: "tag", Description: "tags"})

	err := fs.Parse([]string{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("expected empty slice, got %v", vals)
	}
}

func TestSliceValueString(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() interface{ String() string }
		expected string
	}{
		{
			name: "intSliceValue empty",
			setup: func() interface{ String() string } {
				var s []int
				return &intSliceValue{p: &s}
			},
			expected: "",
		},
		{
			name: "intSliceValue with values",
			setup: func() interface{ String() string } {
				s := []int{1, 2, 3}
				return &intSliceValue{p: &s}
			},
			expected: "1,2,3",
		},
		{
			name: "intSliceValue nil",
			setup: func() interface{ String() string } {
				return &intSliceValue{p: nil}
			},
			expected: "",
		},
		{
			name: "stringSliceValue with values",
			setup: func() interface{ String() string } {
				s := []string{"a", "b", "c"}
				return &stringSliceValue{p: &s}
			},
			expected: "a,b,c",
		},
		{
			name: "float64SliceValue with values",
			setup: func() interface{ String() string } {
				s := []float64{1.5, 2.5}
				return &float64SliceValue{p: &s}
			},
			expected: "1.5,2.5",
		},
		{
			name: "durationSliceValue with values",
			setup: func() interface{ String() string } {
				s := []time.Duration{time.Second, time.Minute}
				return &durationSliceValue{p: &s}
			},
			expected: "1s,1m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.setup()
			got := v.String()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestSliceFlagRequired(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []string
	fs.StringSliceFlag(&vals, ArgConfig{
		Name:        "tag",
		Description: "tags",
		Required:    true,
	})

	err := fs.Parse([]string{})
	if err == nil {
		t.Fatal("expected error for missing required slice flag")
	}
}

func TestSliceFlagRequiredProvided(t *testing.T) {
	fs := NewFlagSet("test")
	var vals []string
	fs.StringSliceFlag(&vals, ArgConfig{
		Name:        "tag",
		Description: "tags",
		Required:    true,
	})

	err := fs.Parse([]string{"-tag", "value"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(vals) != 1 || vals[0] != "value" {
		t.Errorf("expected [value], got %v", vals)
	}
}
