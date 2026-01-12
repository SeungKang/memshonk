package sessiond

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/connmux"
	"github.com/SeungKang/memshonk/internal/cstlv"
	"github.com/SeungKang/memshonk/internal/grsh"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

const (
	unknowMessageType uint16 = iota
	signalMessageType
	terminalResizeMessageType
)

func NewServer(app *app.App) (*Server, error) {
	socketPath := app.Project().WorkspaceConfig().SocketFilePath

	dialCtx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn()

	dialer := net.Dialer{}

	tempConn, _ := dialer.DialContext(dialCtx, "unix", socketPath)
	if tempConn != nil {
		_ = tempConn.Close()
		return nil, fmt.Errorf("a memshonk server is already listening for this project (%q)", socketPath)
	}

	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create server socket - %w", err)
	}

	server := &Server{
		app:        app,
		listener:   listener,
		socketPath: socketPath,
	}

	go func() {
		err := server.loopWithError()
		if err != nil {
			log.Printf("server loop error - %v", err)
		}
	}()

	return server, nil
}

type Server struct {
	app        *app.App
	listener   net.Listener
	socketPath string
	rwMu       sync.RWMutex
	clients    map[net.Conn]*FromClient
}

func (o *Server) Close() error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	_ = o.listener.Close()
	_ = os.Remove(o.socketPath)

	for conn := range o.clients {
		_ = conn.Close()
	}

	return nil
}

func (o *Server) loopWithError() error {
	for {
		conn, err := o.listener.Accept()
		if err != nil {
			return err
		}

		err = o.acceptClient(conn)
		if err != nil {
			_ = conn.Close()

			log.Printf("failed to accept client - %v", err)
		}
	}
}

func (o *Server) acceptClient(conn net.Conn) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	setupCtx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn()

	cm, err := connmux.New(setupCtx, conn)
	if err != nil {
		return err
	}

	apiConn, err := cm.AcceptContext(setupCtx)
	if err != nil {
		_ = cm.Close()
		return err
	}

	stdinConn, err := cm.AcceptContext(setupCtx)
	if err != nil {
		_ = cm.Close()
		return err
	}

	stdoutConn, err := cm.AcceptContext(setupCtx)
	if err != nil {
		_ = cm.Close()
		return err
	}

	stderrConn, err := cm.AcceptContext(setupCtx)
	if err != nil {
		_ = cm.Close()
		return err
	}

	if o.clients == nil {
		o.clients = make(map[net.Conn]*FromClient)
	}

	session, err := o.app.NewSession(app.SessionConfig{
		IO: app.SessionIO{
			Stdin:  stdinConn,
			Stdout: stdoutConn,
			Stderr: stderrConn,
			OptTerminal: goterm.NewVirtualTerminal(goterm.VirtualTerminalConfig{
				Input:  stdinConn,
				Output: stdoutConn,
			}),
		},
		OptID: apiConn.RemoteAddr().String(),
	})
	if err != nil {
		err = fmt.Errorf("failed to create new session - %w", err)

		return err
	}

	// TODO create session context
	ctx := context.Background()

	sh, err := grsh.NewShell(ctx, session)
	if err != nil {
		return err
	}

	go func() {
		// TODO maybe log when the shell exits
		sh.Run()

		o.RemoveSession(conn)
	}()

	o.clients[conn] = NewFromClient(ctx, apiConn, session)

	return nil
}

func (o *Server) RemoveSession(conn net.Conn) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	defer func() {
		go conn.Close()
	}()

	fromClient, hasIt := o.clients[conn]
	if hasIt {
		_ = fromClient.Close()
		delete(o.clients, conn)
	}
}

func NewFromClient(ctx context.Context, conn net.Conn, session *app.Session) *FromClient {
	var cancelFn func()
	ctx, cancelFn = context.WithCancel(ctx)

	fromClient := &FromClient{
		conn:     conn,
		session:  session,
		cancelFn: cancelFn,
	}

	go fromClient.loopWithError(ctx)

	return fromClient
}

type FromClient struct {
	conn     net.Conn
	session  *app.Session
	cancelFn func()
}

func (o *FromClient) Close() error {
	o.cancelFn()

	return o.conn.Close()
}

func (o *FromClient) loopWithError(ctx context.Context) error {
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
