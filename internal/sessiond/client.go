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

func NewClient(ctx context.Context, config ClientConfig) (*Client, error) {
	setupCtx, cancelFn := context.WithTimeout(ctx, time.Second)
	defer cancelFn()

	cm, err := connmux.New(setupCtx, config.ServerConn)
	if err != nil {
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

	stdoutConn, err := cm.DialContext(setupCtx, "", "")
	if err != nil {
		_ = cm.Close()
		return nil, err
	}

	stderrConn, err := cm.DialContext(setupCtx, "", "")
	if err != nil {
		_ = cm.Close()
		return nil, err
	}

	var stopFn func()
	ctx, stopFn = context.WithCancel(ctx)

	client := &Client{
		apiConn:  apiConn,
		cancelFn: stopFn,
		done:     make(chan struct{}),
	}

	go client.loopWithError(ctx)

	go func() {
		err := copyAndAddBackslashRLoop(stdoutConn, config.Stdout)
		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	go func() {
		err := copyAndAddBackslashRLoop(stderrConn, config.Stderr)
		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	go func() {
		// TODO should we just ignore the error if stdin is closed
		_, err := io.Copy(stdinConn, config.Stdin)
		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	return client, nil
}

func copyAndAddBackslashRLoop(conn net.Conn, out io.Writer) error {
	b := make([]byte, 1)

	for {
		_, err := conn.Read(b)
		if err != nil {
			return fmt.Errorf("failed to read from server - %w", err)
		}

		if b[0] == '\n' {
			out.Write([]byte{'\r'})
		}

		_, err = out.Write(b)
		if err != nil {
			return err
		}
	}
}

type Client struct {
	apiConn  net.Conn
	cancelFn func()
	once     sync.Once
	done     chan struct{}
	err      error
}

type ClientConfig struct {
	ServerConn net.Conn
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
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
			case signalMessageType:
				// TODO
			case terminalResizeMessageType:
				// TODO
			default:
				// ignore
			}
		}
	}
}
