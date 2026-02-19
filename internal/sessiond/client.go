package sessiond

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/SeungKang/memshonk/internal/connmux"
	"github.com/SeungKang/memshonk/internal/cstlv"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

type ClientConfig struct {
	SocketPath string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer

	OptTerminalResizes <-chan goterm.ResizeEvent
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
		ioErr:    make(chan error),
		stopped:  make(chan struct{}),
	}

	go client.loop(clientCtx)

	go func() {
		err := copyAndAddBackslashRLoop(stdErrOutConn, config.Stderr, config.Stdout)
		client.onIoError(err)
	}()

	go func() {
		err := copyStdinToConnLoop(client, config.Stdin, stdinConn)
		client.onIoError(err)
	}()

	if config.OptTerminalResizes != nil {
		go func() {
			err := sendTerminalResizeEvents(client, config.OptTerminalResizes)
			client.onIoError(err)
		}()
	}

	return client, nil
}

func sendTerminalResizeEvents(client *Client, events <-chan goterm.ResizeEvent) error {
loop:
	select {
	case <-client.Done():
		return nil
	case event, isOpen := <-events:
		if !isOpen {
			return fmt.Errorf("terminal resize events listener exited unexpectedly")
		}

		err := client.sendTerminalResized(event)
		if err != nil {
			return fmt.Errorf("failed to send terminal resize event - %w", err)
		}
	}

	goto loop
}

func copyAndAddBackslashRLoop(conn net.Conn, stdErr io.Writer, stdOut io.Writer) error {
	splitter := newStdSplitterReader(conn)

loop:
	kind, data, err := splitter.next()
	if err != nil {
		return fmt.Errorf("failed to read from stderr/out conn - %w", err)
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

	goto loop
}

func copyStdinToConnLoop(client *Client, stdin io.Reader, stdinConn net.Conn) error {
	b := make([]byte, 1)
	var err error

loop:
	_, err = stdin.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read from stdin - %w", err)
	}

	// 0x03 == control + c
	if b[0] == 0x03 {
		err = client.sendSignal(IntSignalType)
		if err != nil {
			return fmt.Errorf("failed to send signal - %w", err)
		}
	}

	_, err = stdinConn.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write to stdin - %w", err)
	}

	goto loop
}

type Client struct {
	apiConn  net.Conn
	cancelFn func()
	ioErr    chan error
	stopped  chan struct{}
	err      error
}

func (o *Client) Err() error {
	return o.err
}

func (o *Client) Done() <-chan struct{} {
	return o.stopped
}

func (o *Client) Close() error {
	o.cancelFn()

	<-o.stopped

	return o.err
}

func (o *Client) onIoError(err error) {
	select {
	case <-o.stopped:
	case o.ioErr <- err:
	}
}

func (o *Client) loop(ctx context.Context) {
	o.err = o.loopWithError(ctx)

	o.apiConn.SetDeadline(time.Now().Add(time.Second))

	o.apiConn.Write(cstlv.MinimalBytes(0, 0, goodbyeeeMessageType))

	o.apiConn.Close()

	close(o.stopped)
}

func (o *Client) loopWithError(ctx context.Context) error {
	incomingMessages := make(chan cstlv.ReadResult)
	go cstlv.ReadFromConn(ctx, o.apiConn, incomingMessages, 0)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-o.ioErr:
			return fmt.Errorf("io error - %w", err)
		case result := <-incomingMessages:
			if result.Err != nil {
				return fmt.Errorf("api message read error - %w", result.Err)
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

func (o *Client) sendTerminalResized(event goterm.ResizeEvent) error {
	b, err := terminalSizeToBytes(event.NewSize)
	if err != nil {
		return fmt.Errorf("failed to convert terminal resize event to bytes - %w", err)
	}

	msg := cstlv.CSTLV{
		Type: terminalResizeMessageType,
		Val:  b,
	}

	_, err = o.apiConn.Write(msg.AutoBytes())
	if err != nil {
		return fmt.Errorf("failed to write terminal resize event to api conn - %w", err)
	}

	return nil
}
