package cstlv

import (
	"bufio"
	"context"
	"io"
)

type ReadResult struct {
	Msg *CSTLV
	OCI uint32 // optional channel ID.
	Src io.WriteCloser
	Err error
}

func ReadFromConn(ctx context.Context, conn io.ReadWriteCloser, readResults chan<- ReadResult, optChanID uint32) {
	defer func() {
		_ = conn.Close()
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Split(ScannerSplitFn)

	for scanner.Scan() {
		cp := make([]byte, len(scanner.Bytes()))
		copy(cp, scanner.Bytes())

		msg, err := FromRawBytes(cp)
		if err != nil {
			select {
			case <-ctx.Done():
			case readResults <- ReadResult{Src: conn, OCI: optChanID, Err: err}:
			}

			return
		}

		if msg.Chan == 0 && optChanID > 0 {
			msg.Chan = optChanID
		}

		select {
		case <-ctx.Done():
			return
		case readResults <- ReadResult{Src: conn, OCI: optChanID, Msg: msg}:
		}
	}

	finalErr := scanner.Err()
	if finalErr == nil {
		finalErr = io.EOF
	}

	select {
	case <-ctx.Done():
	case readResults <- ReadResult{Src: conn, OCI: optChanID, Err: finalErr}:
	}
}
