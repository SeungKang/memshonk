package sessiond

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/connmux"
	"github.com/SeungKang/memshonk/internal/cstlv"
)

type ClientConfig struct {
	SocketPath string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func SetupClient(setupCtx context.Context, config ClientConfig) (*Client, error) {
	dialer := net.Dialer{}

	conn, err := dialer.DialContext(setupCtx, "unix", config.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to dial unix socket - %w", err)
	}

	cm, err := connmux.New(setupCtx, conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	apiConn, err := cm.DialContext(setupCtx, "", "")
	if err != nil {
		_ = cm.Close()
		return nil, err
	}

	stdinConn, err := cm.DialContext(setupCtx, "", "")
	if err != nil {
		_ = cm.Close()
		return nil, err
	}

	stdErrOutConn, err := cm.DialContext(setupCtx, "", "")
	if err != nil {
		_ = cm.Close()
		return nil, err
	}

	clientCtx, stopFn := context.WithCancel(context.Background())

	client := &Client{
		apiConn:  apiConn,
		cancelFn: stopFn,
		done:     make(chan struct{}),
	}

	go client.loopWithError(clientCtx)

	go func() {
		err := copyAndAddBackslashRLoop(stdErrOutConn, config.Stderr, config.Stdout)
		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	go func() {
		b := make([]byte, 1)
		var err error

		for {
			_, err = config.Stdin.Read(b)
			if err != nil {
				err = fmt.Errorf("failed to read from stdin - %w", err)
				break
			}

			// 0x03 == control + c
			if b[0] == 0x03 {
				err = client.sendSignal(IntSignalType)
				if err != nil {
					err = fmt.Errorf("failed to send signal - %w", err)
					break
				}
			}

			_, err = stdinConn.Write(b)
			if err != nil {
				err = fmt.Errorf("failed to write to stdin - %w", err)
				break
			}
		}

		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	return client, nil
}

func copyAndAddBackslashRLoop(conn net.Conn, stdErr io.Writer, stdOut io.Writer) error {
	splitter := newStdSplitterReader(conn)

next:
	kind, data, err := splitter.next()
	if err != nil {
		return err
	}

	out := stdOut
	if kind == 0x01 {
		out = stdErr
	}

	for _, b := range data {
		if b == '\n' {
			_, err = out.Write([]byte{'\r'})
			if err != nil {
				return err
			}
		}

		_, err = out.Write([]byte{b})
		if err != nil {
			return err
		}
	}

	goto next
}

type Client struct {
	apiConn  net.Conn
	cancelFn func()
	once     sync.Once
	done     chan struct{}
	err      error
}

func (o *Client) Err() error {
	return o.err
}

func (o *Client) Done() <-chan struct{} {
	return o.done
}

func (o *Client) Close() error {
	o.cancelFn()

	return o.apiConn.Close()
}

func (o *Client) loopWithError(ctx context.Context) error {
	defer o.once.Do(func() {
		o.apiConn.SetDeadline(time.Now().Add(time.Second))

		o.apiConn.Write(cstlv.MinimalBytes(0, 0, goodbyeeeMessageType))

		close(o.done)
	})

	incomingMessages := make(chan cstlv.ReadResult)
	go cstlv.ReadFromConn(ctx, o.apiConn, incomingMessages, 0)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-incomingMessages:
			if result.Err != nil {
				return result.Err
			}

			switch result.Msg.Type {
			case sessionExitedClientMessage:
				return nil
			default:
				// ignore
			}
		}
	}
}

func (o *Client) sendSignal(signalType uint8) error {
	msg := cstlv.CSTLV{
		Type: signalMessageType,
		Val:  []byte{signalType},
	}

	_, err := o.apiConn.Write(msg.AutoBytes())
	return err
}
