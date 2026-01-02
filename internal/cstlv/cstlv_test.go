package cstlv

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	mathrand "math/rand"
	"testing"
)

func TestHappyPath(t *testing.T) {
	payload := []byte("hello world")

	t.Run("ManualBytes", func(t *testing.T) {
		exp := CSTLV{
			Chan: 10,
			Seq:  1000,
			Type: 2,
			Len:  uint32(len(payload)),
			Val:  payload,
		}
		raw, err := exp.ManualBytes()
		if err != nil {
			t.Fatalf("failed to encode - %s", err.Error())
		}
		err = compareExpectedTo(exp, raw)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("AutoBytes", func(t *testing.T) {
		exp := CSTLV{
			Chan: 10,
			Seq:  1000,
			Type: 2,
			Val:  payload,
		}
		err := compareExpectedTo(exp, exp.AutoBytes())
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestScannerSplitFn(t *testing.T) {
	seedRaw := make([]byte, 8)
	_, err := rand.Read(seedRaw)
	if err != nil {
		t.Fatal(err)
	}

	seed := binary.BigEndian.Uint64(seedRaw)
	r := mathrand.New(mathrand.NewSource(int64(seed)))

	buf := bytes.NewBuffer(nil)
	numMessages := r.Intn(1000)
	expected := make([]CSTLV, numMessages)
	for i := 0; i < numMessages; i++ {
		payloadLen := r.Intn(1024)
		payload := make([]byte, payloadLen)
		_, err = rand.Read(payload)
		if err != nil {
			t.Fatal(err)
		}

		exp := CSTLV{
			Chan: uint32(r.Intn(math.MaxUint32)),
			Seq:  uint32(r.Intn(math.MaxUint32)),
			Type: uint16(r.Intn(math.MaxUint16)),
			Len:  uint32(payloadLen),
			Val:  payload,
		}

		expRaw, err := exp.ManualBytes()
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(expRaw)
		expected[i] = exp
	}

	scanner := bufio.NewScanner(buf)
	scanner.Split(ScannerSplitFn)
	i := 0
	for scanner.Scan() {
		err = compareExpectedTo(expected[i], scanner.Bytes())
		if err != nil {
			t.Fatalf("test iteration %d failed - %s", i, err)
		}

		i++
	}
	err = scanner.Err()
	if err != nil {
		t.Fatal(err)
	}

	if i != numMessages {
		t.Fatalf("scanner should have run %d times - it ran %d times", numMessages, i)
	}
}

func compareExpectedTo(exp CSTLV, rawResult []byte) error {
	result, err := FromRawBytes(rawResult)
	if err != nil {
		return fmt.Errorf("failed to parse bytes as cstlv, expected:\n0x%x\n... got:\n0x%x\nerror is: %w",
			exp.AutoBytes(), rawResult, err)
	}

	if result.Chan != exp.Chan {
		return fmt.Errorf("chan should be %d - got %d", exp.Chan, result.Chan)
	}

	if result.Seq != exp.Seq {
		return fmt.Errorf("seq should be %d - got %d", exp.Seq, result.Seq)
	}

	if result.Type != exp.Type {
		return fmt.Errorf("type should be %d - got %d", exp.Type, result.Type)
	}

	if result.Len != exp.Len {
		return fmt.Errorf("len should be %d - got %d", exp.Len, result.Len)
	}

	if !bytes.Equal(result.Val, exp.Val) {
		return fmt.Errorf("data should be:\n0x%x\n... got:\n0x%x",
			exp.Val, result.Val)
	}

	return nil
}
