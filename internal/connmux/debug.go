package connmux

import (
	"crypto/sha256"
	"hash"
	"log"
	"net"
	"time"
)

func NewHashConn(conn net.Conn, logger *log.Logger) net.Conn {
	return &hashConn{
		r: sha256.New(),
		w: sha256.New(),
		c: conn,
		l: logger,
	}
}

type hashConn struct {
	r hash.Hash
	w hash.Hash
	c net.Conn
	l *log.Logger
}

func (o *hashConn) Read(b []byte) (n int, err error) {
	n, err = o.c.Read(b)

	o.r.Write(b[:n])
	o.l.Printf("[read] n: %d | h: %x | v: %x", n, o.r.Sum(nil), b[:n])

	return n, err
}

func (o *hashConn) Write(b []byte) (n int, err error) {
	n, err = o.c.Write(b)

	o.w.Write(b[:n])
	o.l.Printf("[writ] n: %d | h: %x | v: %x", n, o.w.Sum(nil), b[:n])

	return n, err
}

func (o *hashConn) Close() error {
	return o.c.Close()
}

func (o *hashConn) LocalAddr() net.Addr {
	return o.c.LocalAddr()
}

func (o *hashConn) RemoteAddr() net.Addr {
	return o.c.RemoteAddr()
}

func (o *hashConn) SetDeadline(t time.Time) error {
	return o.c.SetDeadline(t)
}

func (o *hashConn) SetReadDeadline(t time.Time) error {
	return o.c.SetReadDeadline(t)
}

func (o *hashConn) SetWriteDeadline(t time.Time) error {
	return o.c.SetWriteDeadline(t)
}
