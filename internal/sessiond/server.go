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
	"github.com/SeungKang/memshonk/internal/grsh"
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
	clients    map[net.Conn]*app.Session
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

	if o.clients == nil {
		o.clients = make(map[net.Conn]*app.Session)
	}

	var optSessionId string

	session, err := o.app.NewSession(app.SessionConfig{
		IO: app.SessionIO{
			Stdin:  conn,
			Stdout: conn,
			Stderr: conn,
		},
		OptID: optSessionId,
	})
	if err != nil {
		err = fmt.Errorf("failed to create new session - %w", err)

		return err
	}

	// TODO use actual context
	sh, err := grsh.NewShell(context.Background(), session)
	if err != nil {
		return err
	}

	go func() {
		// TODO maybe log when the shell exits
		sh.Run()

		o.RemoveSession(conn)
	}()

	o.clients[conn] = session

	return nil
}

func (o *Server) RemoveSession(conn net.Conn) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	defer func() {
		go conn.Close()
	}()

	_, hasIt := o.clients[conn]
	if hasIt {
		delete(o.clients, conn)
	}
}
