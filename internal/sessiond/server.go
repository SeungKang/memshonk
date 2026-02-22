package sessiond

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/connmux"
	"github.com/SeungKang/memshonk/internal/cstlv"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

// Various messages sent to the server by the client.
const (
	unknowMessageType uint16 = iota
	signalMessageType
	terminalResizeMessageType
	goodbyeeeMessageType
)

// Various messages sent to the client by the server.
const (
	unknownClientMessage uint16 = iota
	sessionExitedClientMessage
)

type ServerConfig struct {
	SharedState apicompat.SharedState
	NewShellFn  func(apicompat.Session) (Shell, error)
}

type Shell interface {
	Run(context.Context) error

	Close() error
}

func NewServer(ctx context.Context, config ServerConfig) (*Server, error) {
	socketPath := config.SharedState.Project.WorkspaceConfig().SocketFilePath

	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create server socket - %w", err)
	}

	server := &Server{
		config:     config,
		listener:   listener,
		socketPath: socketPath,
	}

	server.config.SharedState.Sessions = server

	go func() {
		err := server.loopWithError(ctx)
		if err != nil {
			log.Printf("server loop error - %v", err)
		}
	}()

	return server, nil
}

type Server struct {
	config     ServerConfig
	listener   net.Listener
	socketPath string
	rwMu       sync.RWMutex
	randStr    *randomStringer
	sessions   map[string]sessionWrapper
}

type sessionWrapper struct {
	session    *Session
	clientConn *fromClient
}

func (o *Server) Close() error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	_ = o.listener.Close()
	_ = os.Remove(o.socketPath)

	for _, wrapper := range o.sessions {
		_ = wrapper.session.Close()
	}

	return nil
}

func (o *Server) loopWithError(ctx context.Context) error {
	for {
		conn, err := o.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			return err
		}

		err = o.acceptClient(ctx, conn)
		if err != nil {
			_ = conn.Close()

			log.Printf("failed to accept client - %v", err)
		}
	}
}

func (o *Server) acceptClient(ctx context.Context, conn net.Conn) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	setupCtx, cancelFn := context.WithTimeout(ctx, time.Second)
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

	// We use a single net.Conn to transport both stderr and
	// stdout data in order to preserve the order in which
	// it was generated in.
	//
	// We originally used two net.Conns (one for stderr and
	// one for stdout), but that resulted in the data being
	// parsed out of order since it required two go routines
	// to read from both net.Conns. It may have been possible
	// to continue using that design, but I think using a single
	// net.Conn ends up being simpler.
	stdErrAndOutConn, err := cm.AcceptContext(setupCtx)
	if err != nil {
		_ = cm.Close()
		return err
	}

	stdout := stdSplitterWriter{conn: stdErrAndOutConn}

	_, err = o.newSession(ctx, SessionConfig{
		IO: apicompat.SessionIO{
			Stdin:        stdinConn,
			Stdout:       stdout,
			Stderr:       stdSplitterWriter{conn: stdErrAndOutConn, isStderr: true},
			BuiltinUsage: stdout,
			OptTerminal: goterm.NewVirtualTerminal(goterm.VirtualTerminalConfig{
				Input:  stdinConn,
				Output: stdSplitterWriter{conn: stdErrAndOutConn},
			}),
		},
		OptID:         apiConn.RemoteAddr().String(),
		ClientConn:    cm,
		clientApiConn: apiConn,
	})
	if err != nil {
		_ = cm.Close()
		return fmt.Errorf("failed to create new session - %w", err)
	}

	return nil
}

type SessionConfig struct {
	IO         apicompat.SessionIO
	IsDefault  bool
	ClientConn io.Closer
	OptID      string

	clientApiConn net.Conn
}

func (o *Server) newSession(ctx context.Context, config SessionConfig) (*Session, error) {
	var id string

	if config.OptID == "" {
		if o.randStr == nil {
			o.randStr = newRandomStringer()
		}

		for i := 0; i < 100; i++ {
			possibleId := o.randStr.String()

			_, hasIt := o.sessions[possibleId]
			if !hasIt {
				id = possibleId

				break
			}
		}

		if id == "" {
			var buf bytes.Buffer

			b := make([]byte, 4)

			_, err := rand.Read(b)
			if err != nil {
				panic(err)
			}

			_, err = hex.NewEncoder(&buf).Write(b)
			if err != nil {
				panic(err)
			}

			id = buf.String()
		}
	} else {
		_, hasIt := o.sessions[config.OptID]
		if hasIt {
			return nil, fmt.Errorf("session id already in use (%q)",
				config.OptID)
		}

		id = config.OptID
	}

	switch {
	case id == "":
		return nil, errors.New("session id string is empty")
	case id == "default":
		if config.IO.Stdin != os.Stdin {
			return nil, errors.New("remote client requested a reserved session id")
		}
	case strings.ContainsAny(id, "/\\"):
		return nil, errors.New("session id contains path separator character(s)")
	}

	sessionCtx, cancelSessionFn := context.WithCancel(ctx)

	session := &Session{
		info: apicompat.SessionInfo{
			ID:        id,
			StartedAt: time.Now(),
		},
		isDefault: config.IsDefault,
		shared:    o.config.SharedState,
		io:        config.IO,
		cmdExec:   &apicompat.CommandExecutor{},
		ctx:       sessionCtx,
		cancelFn:  cancelSessionFn,
		apiConn:   config.clientApiConn,
	}

	sh, err := o.config.NewShellFn(session)
	if err != nil {
		_ = session.Close()

		return nil, err
	}

	session.shell = sh

	go func() {
		// TODO maybe log when the shell exits
		sh.Run(sessionCtx)

		o.RemoveSession(id)
	}()

	if o.sessions == nil {
		o.sessions = make(map[string]sessionWrapper)
	}

	o.sessions[id] = sessionWrapper{
		session:    session,
		clientConn: newFromClient(sessionCtx, config.clientApiConn, session),
	}

	return session, nil
}

func (o *Server) Sessions() []apicompat.Session {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	sessions := make([]apicompat.Session, 0, len(o.sessions))

	for _, wrapper := range o.sessions {
		sessions = append(sessions, wrapper.session)
	}

	sort.SliceStable(sessions, func(i int, j int) bool {
		return sessions[i].Info().ID < sessions[j].Info().ID
	})

	return sessions
}

func (o *Server) GetSession(id string) (apicompat.Session, bool) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	wrapper, hasIt := o.sessions[id]

	if hasIt {
		return wrapper.session, true
	}

	return nil, false
}

func (o *Server) RemoveSession(id string) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	wrapper, hasIt := o.sessions[id]
	if hasIt {
		if wrapper.clientConn != nil {
			timeout := time.After(time.Second)

			wrapper.clientConn.apiConn.SetDeadline(
				time.Now().Add(time.Second))

			wrapper.clientConn.apiConn.Write(
				cstlv.MinimalBytes(0, 0, sessionExitedClientMessage))

			select {
			case <-timeout:
			case <-wrapper.clientConn.session.Done():
			}
		}

		_ = wrapper.session.Close()

		delete(o.sessions, id)
	}
}

func newFromClient(ctx context.Context, apiConn net.Conn, session *Session) *fromClient {
	var cancelFn func()
	ctx, cancelFn = context.WithCancel(ctx)

	fromClient := &fromClient{
		apiConn:  apiConn,
		session:  session,
		done:     ctx.Done(),
		cancelFn: cancelFn,
	}

	go fromClient.loopWithError(ctx)

	return fromClient
}

type fromClient struct {
	apiConn  net.Conn
	session  *Session
	done     <-chan struct{}
	cancelFn func()
}

func (o *fromClient) Close() error {
	o.cancelFn()

	return o.apiConn.Close()
}

func (o *fromClient) loopWithError(ctx context.Context) error {
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
				o.handleSignalMessage(result.Msg)
			case terminalResizeMessageType:
				if o.session.io.OptTerminal != nil {
					o.handTerminalResizeMessage(result.Msg, o.session.io.OptTerminal)
				}
			case goodbyeeeMessageType:
				_ = o.session.Close()

				return nil
			default:
				// ignore
			}
		}
	}
}

func (o *fromClient) handleSignalMessage(msg *cstlv.CSTLV) {
	if len(msg.Val) == 0 {
		return
	}

	o.session.OnSignal(msg.Val[0])
}

func (o *fromClient) handTerminalResizeMessage(msg *cstlv.CSTLV, terminal *goterm.VirtualTerminal) {
	newSize, err := terminalSizeFromBytes(msg.Val)
	if err != nil {
		return
	}

	terminal.SetSize(newSize)
}

func terminalSizeToBytes(size goterm.Size) ([]byte, error) {
	var buf bytes.Buffer

	err := binary.Write(&buf, binary.BigEndian, tmpTermSize{
		Cols: int64(size.Cols),
		Rows: int64(size.Rows),
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func terminalSizeFromBytes(msg []byte) (goterm.Size, error) {
	var event tmpTermSize

	err := binary.Read(bytes.NewReader(msg), binary.BigEndian, &event)
	if err != nil {
		return goterm.Size{}, err
	}

	return goterm.Size{
		Cols: int(event.Cols),
		Rows: int(event.Rows),
	}, nil
}

// tmpTermSize is needed because goterm.Size does uses non-fixed sized
// integers in its fields.
//
// TODO: Remove once we replace those fields with fixed-size types.
type tmpTermSize struct {
	Cols int64
	Rows int64
}
