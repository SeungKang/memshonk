package sessiond

import (
	"context"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/connmux"
	"github.com/SeungKang/memshonk/internal/cstlv"
)

func NewClient(ctx context.Context, conn net.Conn) (*Client, error) {
	setupCtx, cancelFn := context.WithTimeout(ctx, time.Second)
	defer cancelFn()

	cm, err := connmux.New(setupCtx, conn)
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
		conn:     apiConn,
		cancelFn: stopFn,
		done:     make(chan struct{}),
	}

	go client.loopWithError(ctx)

	go func() {
		_, err := io.Copy(os.Stdout, stdoutConn)
		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	go func() {
		_, err := io.Copy(os.Stderr, stderrConn)
		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	go func() {
		// TODO should we just ignore the error if stdin is closed
		_, err := io.Copy(stdinConn, os.Stdin)
		client.once.Do(func() {
			client.err = err
			close(client.done)
		})
	}()

	return client, nil
}

type Client struct {
	conn     net.Conn
	cancelFn func()
	once     sync.Once
	done     chan struct{}
	err      error
}

func (o *Client) Close() error {
	o.cancelFn()

	return o.conn.Close()
}

func (o *Client) loopWithError(ctx context.Context) error {
	incomingMessages := make(chan cstlv.ReadResult)
	go cstlv.ReadFromConn(ctx, o.conn, incomingMessages, 0)

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
