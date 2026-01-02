package connmux

import (
	"bytes"
	"container/list"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/cstlv"
)

var (
	errDeadlineExceeded = errors.New("deadline exceeded")
)

// New instantiates a Mux on the provided net.Conn that can
// dial outgoing connections and accept incoming connections.
//
// ctx is used to cancel the Mux handshake according to the
// caller's desired cancellation criteria.
func New(ctx context.Context, conn net.Conn) (*Mux, error) {
	mux := newUnstartedMux(conn)

	err := startMux(ctx, mux)
	if err != nil {
		return nil, err
	}

	return mux, nil
}

// NewDialOnly instantiates a Mux on the provided net.Conn that can
// dial outgoing connections, but will refuse incoming connections.
//
// ctx is used to cancel the Mux handshake according to the
// caller's desired cancellation criteria.
func NewDialOnly(ctx context.Context, conn net.Conn) (*Mux, error) {
	mux := newUnstartedMux(conn)
	mux.noAccept = true

	err := startMux(ctx, mux)
	if err != nil {
		return nil, err
	}

	return mux, nil
}

func startMux(ctx context.Context, mux *Mux) error {
	go mux.loop()

	select {
	case <-ctx.Done():
		_ = mux.Close()
		return ctx.Err()
	case <-time.After(5 * time.Second):
		_ = mux.Close()
		return errors.New("timed-out waiting for mux handshake to complete")
	case <-mux.hsComplete:
		return nil
	}
}

func newUnstartedMux(conn net.Conn) *Mux {
	return &Mux{
		underlying: conn,
		idsToConns: make(map[uint32]*muxChild),
		dialing:    make(map[uint32]*dialCB),
		hsComplete: make(chan struct{}),
		accept:     make(chan *muxChild, 10),
		onDial:     make(chan *dialCB),
		onWrite:    make(chan *writeCB),
		onClose:    make(chan *muxChild),
		close:      make(chan struct{}),
		closed:     make(chan struct{}),
	}
}

// Mux allows a single net.Conn to be split into several connections
// by multiplexing the underling net.Conn.
//
// Refer to the documentation for the AcceptContext and DialContext
// for more information.
type Mux struct {
	underlying net.Conn
	ourNum     uint8
	idsToConns map[uint32]*muxChild
	dialing    map[uint32]*dialCB
	hsDone     bool
	hsComplete chan struct{}
	nextID     uint32
	noAccept   bool
	accept     chan *muxChild
	onDial     chan *dialCB
	onWrite    chan *writeCB
	onClose    chan *muxChild
	closeOnce  sync.Once
	close      chan struct{}
	closed     chan struct{}
	err        error
}

// AcceptContext accepts the next incoming net.Conn from the peer Mux.
//
// The provided Context must be non-nil. If the context expires before
// a connection is accepted, an error is returned. Once successfully
// connected, any expiration of the context will not affect the connection.
//
// The resulting net.Conn's LocalAddr and RemoteAddr methods will
// return a net.Addr that contains the originally-specified network
// string and address strings. The address string will contain the
// address and underlying connection ID like so:
//
//	<addr-string>:<connection-id>
func (o *Mux) AcceptContext(ctx context.Context) (net.Conn, error) {
	if o.noAccept {
		return nil, errors.New("accepting connections is disabled")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-o.closed:
		return nil, o.err
	case conn := <-o.accept:
		return conn, nil
	}
}

// DialContext dials an outgoing connection for the given network
// and address.
//
// The provided Context must be non-nil. If the context expires
// before the connection is complete, an error is returned. Once
// successfully connected, any expiration of the context will not
// affect the connection.
//
// The network and address strings can be used by the peer
// Mux to derive the desired Go code to connect to. Refer to
// the documentation for AcceptContext for more information.
func (o *Mux) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	cb := &dialCB{
		net:   network,
		addr:  address,
		ready: make(chan struct{}),
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-o.closed:
		return nil, o.err
	case o.onDial <- cb:
		<-cb.ready
		return cb.conn, cb.err
	}
}

type dialCB struct {
	net   string
	addr  string
	ready chan struct{}
	conn  *muxChild
	err   error
}

// Close closes the Mux and any remaining open connections. Any calls to
// AcceptContext or DialContext are canceled.
func (o *Mux) Close() error {
	o.closeOnce.Do(func() {
		close(o.close)
		<-o.closed
	})

	return o.err
}

func (o *Mux) loop() {
	defer func() {
		var finalErr error
		if o.err != nil {
			finalErr = o.err
		} else {
			finalErr = errors.New("conn mux has been closed")
		}

		for id, dialing := range o.dialing {
			delete(o.dialing, id)

			dialing.err = finalErr

			close(dialing.ready)
		}

		for id, conn := range o.idsToConns {
			delete(o.idsToConns, id)

			_ = o.closeChild(conn, finalErr, true)
		}

		_ = o.underlying.Close()

		close(o.closed)
	}()

	readResults := make(chan *connMuxMessage)

	go readConnMuxMessage(o.closed, o.underlying, readResults)

	err := o.sendHandshake()
	if err != nil {
		o.err = fmt.Errorf("failed to send initial handshake - %w", err)
		return
	}

	for {
		select {
		case <-o.close:
			o.err = errors.New("mux closed")
			return
		case dial := <-o.onDial:
			err = o.dial(dial)
			if err != nil {
				o.err = fmt.Errorf("failed to send dial - %w", err)
				return
			}
		case message := <-readResults:
			err = o.handleMessage(message)
			if err != nil {
				o.err = fmt.Errorf("failed to handle mux message - %w", err)
				return
			}
		case write := <-o.onWrite:
			o.writeChildPayload(write)
		case child := <-o.onClose:
			err = o.onConnClosed(child, errors.New("peer closed connection"), true)
			if err != nil {
				o.err = fmt.Errorf("failed to close child conn - %w", err)
				return
			}
		}
	}
}

func (o *Mux) sendHandshake() error {
	b := make([]byte, 1)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}

	o.ourNum = b[0]

	err = o.writeMessage(&connMuxMessage{
		Type:    handshakeConnMuxMessage,
		Payload: b,
	})
	if err != nil {
		return fmt.Errorf("failed to send handshake message - %w", err)
	}

	return nil
}

func (o *Mux) dial(cb *dialCB) error {
	id := o.nextID

	err := o.writeMessage(&connMuxMessage{
		Type:    dialConnMuxMessage,
		SAddr:   id,
		Payload: dialPayload(cb.net, cb.addr),
	})
	if err != nil {
		cb.err = err
		close(cb.ready)
		return err
	}

	o.nextID = o.nextID + 2

	o.dialing[id] = cb

	return nil
}

func dialPayload(network string, address string) []byte {
	b := make([]byte, len(network)+len(address)+1)

	n := copy(b, network)
	b[n] = 0x00

	copy(b[n+1:], address)

	return b
}

func splitDialPayload(b []byte) (network string, addr string) {
	if len(b) == 0 {
		return "", ""
	}

	before, after, found := bytes.Cut(b, []byte{0x00})
	if found {
		return string(before), string(after)
	}

	return string(b), ""
}

func (o *Mux) handleMessage(message *connMuxMessage) error {
	if message.err != nil {
		return message.err
	}

	switch message.Type {
	case errConnMuxMessage:
		return fmt.Errorf("received mux-level peer error - %s", string(message.Payload))
	case errChildConnMuxMessage:
		recErr := fmt.Errorf("peer error - %s", string(message.Payload))

		conn, hasIt := o.idsToConns[message.SAddr]
		if hasIt {
			return o.closeChild(conn, recErr, true)
		}

		cb, hasIt := o.dialing[message.SAddr]
		if hasIt {
			cb.err = recErr
			close(cb.ready)
			return nil
		}

		return nil
	case handshakeConnMuxMessage:
		if o.hsDone {
			return nil
		}

		if len(message.Payload) == 0 {
			_ = o.writeMessage(&connMuxMessage{
				Type:    errConnMuxMessage,
				Payload: []byte("handshake payload is empty"),
			})

			return errors.New("peer's handshake payload is empty")
		}

		switch {
		case message.Payload[0] == o.ourNum:
			err := o.sendHandshake()
			if err != nil {
				return fmt.Errorf("failed to send follow-up handshake - %w", err)
			}

			return nil
		default:
			if o.ourNum < message.Payload[0] {
				o.nextID = 1
				// 1
				// 1+2 = 3
				// 3+2 = 5
				// 5+2 = 7
				// 7+2 = 9
				// 9+2 = 11
				// versus:
				// 0
				// 0+2 = 2
				// 2+2 = 4
				// 4+2 = 6
				// 6+2 = 8
				// 8+2 = 10
				// 10+2 = 12
			}

			return o.writeMessage(&connMuxMessage{
				Type: handshakeAckConnMuxMessage,
			})
		}
	case handshakeAckConnMuxMessage:
		if !o.hsDone {
			close(o.hsComplete)
			o.hsDone = true
		}
		return nil
	case dialConnMuxMessage:
		return o.acceptDial(message)
	case dialAckConnMuxMessage:
		cb, hasIt := o.dialing[message.SAddr]
		if !hasIt {
			return o.writeMessage(&connMuxMessage{
				Type:  errChildConnMuxMessage,
				SAddr: message.SAddr,
				Payload: []byte(fmt.Sprintf("received dial ack for non-existent socket addr: '%d'",
					message.SAddr)),
			})
		}

		delete(o.dialing, message.SAddr)

		conn := o.newChild(message.SAddr, cb.net, cb.addr)
		cb.conn = conn
		o.idsToConns[message.SAddr] = conn

		close(cb.ready)
		return nil
	case closeConnMuxMessage, payloadConnMuxMessage:
		conn, hasIt := o.idsToConns[message.SAddr]
		if !hasIt {
			return o.writeMessage(&connMuxMessage{
				Type:    errChildConnMuxMessage,
				SAddr:   message.SAddr,
				Payload: []byte(fmt.Sprintf("received data for unknown socket addr: '%d'", message.SAddr)),
			})
		}

		if message.Type == closeConnMuxMessage {
			return o.closeChild(conn, errors.New(string(message.Payload)), false)
		}

		conn.addToReadBuf(message.Payload)

		return nil
	default:
		return fmt.Errorf("unknown message type: %d", message.Type)
	}
}

func (o *Mux) acceptDial(message *connMuxMessage) error {
	if o.noAccept {
		return o.writeMessage(&connMuxMessage{
			Type:    errChildConnMuxMessage,
			SAddr:   message.SAddr,
			Payload: []byte(fmt.Sprintf("peer does not permit incomming connections")),
		})
	}

	network, addr := splitDialPayload(message.Payload)

	childConn := o.newChild(message.SAddr, network, addr)

	acceptTimeout := time.NewTimer(time.Second)
	defer acceptTimeout.Stop()

	select {
	case <-acceptTimeout.C:
		return o.writeMessage(&connMuxMessage{
			Type:    errChildConnMuxMessage,
			SAddr:   message.SAddr,
			Payload: []byte("timed-out waiting for connection to be accepted"),
		})
	case o.accept <- childConn:
		o.idsToConns[childConn.sAddr] = childConn

		return o.writeMessage(&connMuxMessage{
			Type:  dialAckConnMuxMessage,
			SAddr: message.SAddr,
		})
	}
}

func (o *Mux) newChild(sAddr uint32, network string, addr string) *muxChild {
	netAddr := &customAddr{
		network: network,
		str:     fmt.Sprintf("%s:%d", addr, sAddr),
	}

	return &muxChild{
		sAddr:  sAddr,
		lAddr:  netAddr,
		rAddr:  netAddr,
		reads:  make(chan connMuxReadReady, 1),
		readB:  list.New(),
		rdlCh:  make(chan struct{}),
		write:  o.onWrite,
		wdlCh:  make(chan struct{}),
		mux:    o,
		closed: make(chan struct{}),
	}
}

func (o *Mux) writeChildPayload(cb *writeCB) {
	defer close(cb.ready)

	conn, hasIt := o.idsToConns[cb.sAddr]
	if !hasIt {
		cb.err = fmt.Errorf("unknown socket addr: '%d'", cb.sAddr)
		return
	}

	err := o.writeMessage(&connMuxMessage{
		Type:    payloadConnMuxMessage,
		SAddr:   conn.sAddr,
		Payload: cb.b,
	})
	if err != nil {
		cb.err = err
		return
	}

	cb.n = len(cb.b)
}

func (o *Mux) writeMessage(message *connMuxMessage) error {
	msg := cstlv.CSTLV{
		Chan: message.SAddr,
		Type: uint16(message.Type),
		Val:  message.Payload,
	}

	_, err := o.underlying.Write(msg.AutoBytes())
	if err != nil {
		return fmt.Errorf("failed to write mux message - %w", err)
	}

	return nil
}

func (o *Mux) closeChild(child *muxChild, err error, sendMsg bool) error {
	child.close(err)

	return o.onConnClosed(child, err, sendMsg)
}

func (o *Mux) onConnClosed(child *muxChild, err error, sendMsg bool) error {
	delete(o.idsToConns, child.sAddr)

	if sendMsg {
		return o.writeMessage(&connMuxMessage{
			Type:    closeConnMuxMessage,
			SAddr:   child.sAddr,
			Payload: []byte(err.Error()),
		})
	}

	return nil
}

func (o *Mux) childClosed(child *muxChild) {
	select {
	case <-o.closed:
	case o.onClose <- child:
	}
}

type muxChild struct {
	sAddr  uint32
	lAddr  *customAddr
	rAddr  *customAddr
	reads  chan connMuxReadReady
	readM  sync.Mutex
	readB  *list.List
	write  chan<- *writeCB
	rdlMu  sync.RWMutex
	rdlRst func()
	rdlCh  chan struct{}
	wdlMu  sync.RWMutex
	wdlRst func()
	wdlCh  chan struct{}
	once   sync.Once
	mux    *Mux
	closed chan struct{}
	err    error
}

func (o *muxChild) addToReadBuf(b []byte) {
	o.readM.Lock()
	defer o.readM.Unlock()

	o.readB.PushBack(b)

	select {
	case o.reads <- connMuxReadReady{}:
	default:
	}
}

type connMuxReadReady struct{}

func (o *muxChild) Read(b []byte) (int, error) {
	select {
	case <-o.closed:
		return 0, o.err
	case <-o.rdlCh:
		return 0, fmt.Errorf("read: %w", errDeadlineExceeded)
	case <-o.reads:
		o.readM.Lock()
		defer o.readM.Unlock()

		element := o.readB.Front()
		if element == nil {
			return 0, nil
		}

		val := element.Value.([]byte)

		n := copy(b, val)
		if n == len(val) {
			o.readB.Remove(element)
		} else {
			element.Value = val[n:]
		}

		if o.readB.Len() > 0 {
			select {
			case o.reads <- connMuxReadReady{}:
			default:
			}
		}

		return n, nil
	}
}

func (o *muxChild) Write(b []byte) (n int, err error) {
	cb := &writeCB{
		sAddr: o.sAddr,
		b:     b,
		ready: make(chan struct{}),
	}

	select {
	case <-o.closed:
		return 0, o.err
	case <-o.wdlCh:
		return 0, fmt.Errorf("write: %w", errDeadlineExceeded)
	case o.write <- cb:
		// Keep going.
	}

	select {
	case <-o.wdlCh:
		return 0, fmt.Errorf("write: %w", errDeadlineExceeded)
	case <-cb.ready:
		return cb.n, cb.err
	}
}

type writeCB struct {
	sAddr uint32
	b     []byte
	ready chan struct{}
	n     int
	err   error
}

func (o *muxChild) Close() error {
	o.close(errors.New("connection closed"))
	go o.mux.childClosed(o)
	return nil
}

func (o *muxChild) close(err error) {
	o.once.Do(func() {
		o.err = err
		close(o.closed)
	})
}

func (o *muxChild) LocalAddr() net.Addr {
	return o.lAddr
}

func (o *muxChild) RemoteAddr() net.Addr {
	return o.rAddr
}

func (o *muxChild) SetDeadline(t time.Time) error {
	o.rdlMu.Lock()
	defer o.rdlMu.Unlock()

	o.wdlMu.Lock()
	defer o.wdlMu.Unlock()

	err := o.setDeadline(t, &o.rdlRst, o.rdlCh)
	if err != nil {
		return fmt.Errorf("failed to set read deadline - %w", err)
	}

	err = o.setDeadline(t, &o.wdlRst, o.wdlCh)
	if err != nil {
		return fmt.Errorf("failed to set write deadline - %w", err)
	}

	return nil
}

func (o *muxChild) SetReadDeadline(t time.Time) error {
	o.rdlMu.Lock()
	defer o.rdlMu.Unlock()

	return o.setDeadline(t, &o.rdlRst, o.rdlCh)
}

func (o *muxChild) SetWriteDeadline(t time.Time) error {
	o.wdlMu.Lock()
	defer o.wdlMu.Unlock()

	return o.setDeadline(t, &o.wdlRst, o.wdlCh)
}

func (o *muxChild) setDeadline(t time.Time, cancelDlFnPtr *func(), deadlineCh chan<- struct{}) error {
	if *cancelDlFnPtr != nil {
		cancelFn := *cancelDlFnPtr

		cancelFn()

		*cancelDlFnPtr = nil
	}

	if t.Equal(time.Time{}) {
		return nil
	}

	ctx := context.Background()

	canceledCtx, cancelFn := context.WithCancel(ctx)
	*cancelDlFnPtr = cancelFn

	deadlineCtx, cancelDeadline := context.WithDeadline(ctx, t)

	go func() {
		defer cancelDeadline()

		select {
		case <-o.closed:
			return
		case <-canceledCtx.Done():
			return
		case <-deadlineCtx.Done():
			// Deadline exceeded.
		}

		for {
			select {
			case <-o.closed:
				return
			case <-canceledCtx.Done():
				return
			case deadlineCh <- struct{}{}:
				// Keep going.
			}
		}
	}()

	return nil
}

type connMuxMessage struct {
	Type    connMuxMessageType
	SAddr   uint32
	Payload []byte
	err     error
}

const (
	unknownConnMuxMessage connMuxMessageType = iota
	errConnMuxMessage
	errChildConnMuxMessage
	handshakeConnMuxMessage
	handshakeAckConnMuxMessage
	dialConnMuxMessage
	dialAckConnMuxMessage
	payloadConnMuxMessage
	closeConnMuxMessage
)

type connMuxMessageType uint16

func readConnMuxMessage(done <-chan struct{}, reader io.Reader, c chan<- *connMuxMessage) {
	parser := cstlv.NewBufferedParserFrom(reader)

	for parser.Next() {
		select {
		case <-done:
			return
		case c <- &connMuxMessage{
			Type:    connMuxMessageType(parser.Message().Type),
			SAddr:   parser.Message().Chan,
			Payload: parser.Message().Val,
		}:
		}
	}

	finalErr := parser.Err()
	if finalErr == nil {
		finalErr = io.EOF
	}

	select {
	case <-done:
	case c <- &connMuxMessage{err: finalErr}:
	}
}

type customAddr struct {
	network string
	str     string
}

func (o *customAddr) Network() string {
	return o.network
}

func (o *customAddr) String() string {
	return o.str
}
