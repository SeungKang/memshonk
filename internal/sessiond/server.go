package sessiond

import (
	"bytes"
	"context"
	"crypto/rand"
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
	"github.com/SeungKang/memshonk/internal/grsh"
	"github.com/SeungKang/memshonk/internal/shvars"
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

func NewServer(ctx context.Context, sharedState apicompat.SharedState) (*Server, error) {
	socketPath := sharedState.Project.WorkspaceConfig().SocketFilePath

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
		sharedState: sharedState,
		listener:    listener,
		socketPath:  socketPath,
	}

	server.sharedState.Sessions = server

	go func() {
		err := server.loopWithError(ctx)
		if err != nil {
			log.Printf("server loop error - %v", err)
		}
	}()

	return server, nil
}

type Server struct {
	sharedState apicompat.SharedState
	listener    net.Listener
	socketPath  string
	rwMu        sync.RWMutex
	randStr     *randomStringer
	sessions    map[string]sessionWrapper
}

type sessionWrapper struct {
	session         *Session
	optRemoteClient *fromClient
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

	_, err = o.newSession(ctx, SessionConfig{
		IO: apicompat.SessionIO{
			Stdin:  stdinConn,
			Stdout: stdoutConn,
			Stderr: stderrConn,
			OptTerminal: goterm.NewVirtualTerminal(goterm.VirtualTerminalConfig{
				Input:  stdinConn,
				Output: stdoutConn,
			}),
		},
		OptID:            apiConn.RemoteAddr().String(),
		OptCloseConn:     cm,
		optClientApiConn: apiConn,
	})
	if err != nil {
		_ = cm.Close()
		return fmt.Errorf("failed to create new session - %w", err)
	}

	return nil
}

type SessionConfig struct {
	IO           apicompat.SessionIO
	IsDefault    bool
	OptCloseConn io.Closer
	OptID        string

	optClientApiConn net.Conn
}

func (o *Server) NewSession(ctx context.Context, config SessionConfig) (*Session, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	return o.newSession(ctx, config)
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

	stopper := newSessionStopper(sessionCtx, cancelSessionFn, config.OptCloseConn)

	session := &Session{
		info: apicompat.SessionInfo{
			ID:        id,
			StartedAt: time.Now(),
		},
		isDefault: config.IsDefault,
		shared:    o.sharedState,
		vars: &SessionVariables{
			proj: o.sharedState.Project,
			vars: &shvars.Variables{},
		},
		io:      config.IO,
		stopper: stopper,
	}

	sh, err := grsh.NewShell(sessionCtx, session)
	if err != nil {
		_ = stopper.Close()
		return nil, err
	}

	go func() {
		// TODO maybe log when the shell exits
		sh.Run()

		o.RemoveSession(id)
	}()

	if o.sessions == nil {
		o.sessions = make(map[string]sessionWrapper)
	}

	var optClientApiConn *fromClient
	if config.optClientApiConn != nil {
		optClientApiConn = newFromClient(sessionCtx, config.optClientApiConn, session)
	}

	o.sessions[id] = sessionWrapper{
		session:         session,
		optRemoteClient: optClientApiConn,
	}

	return session, nil
}

func newSessionStopper(ctx context.Context, cancelFn func(), optConn io.Closer) *sessionStopper {
	return &sessionStopper{
		ctx:       ctx,
		cancelFn:  cancelFn,
		optCloser: optConn,
	}
}

type sessionStopper struct {
	ctx       context.Context
	ocne      sync.Once
	cancelFn  func()
	optCloser io.Closer
}

func (o *sessionStopper) Close() error {
	var err error

	o.ocne.Do(func() {
		o.cancelFn()

		if o.optCloser != nil {
			err = o.optCloser.Close()
		}
	})

	return err
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
		if wrapper.optRemoteClient != nil {
			timeout := time.After(time.Second)

			wrapper.optRemoteClient.apiConn.SetDeadline(
				time.Now().Add(time.Second))

			wrapper.optRemoteClient.apiConn.Write(
				cstlv.MinimalBytes(0, 0, sessionExitedClientMessage))

			select {
			case <-timeout:
			case <-wrapper.optRemoteClient.session.Done():
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
				// TODO
			case goodbyeeeMessageType:
				o.session.stopper.Close()

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
