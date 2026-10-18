//go:build cgo && polvo_p2p

// @anchors
//   code: TRCGO
//   ref: TRNSP

package transport

/*
#cgo LDFLAGS: -lpolvo_p2p -lm -ldl -lpthread
#include <stdint.h>
#include <stdlib.h>
int64_t polvo_p2p_open(const uint8_t *seed, const char *relay, char **err);
char *polvo_p2p_addr(int64_t h);
char *polvo_p2p_id(int64_t h);
int64_t polvo_p2p_accept(int64_t h, char **remote, char **err);
int64_t polvo_p2p_connect(int64_t h, const char *addr, char **err);
int64_t polvo_p2p_read(int64_t s, uint8_t *buf, size_t len);
int64_t polvo_p2p_write(int64_t s, const uint8_t *buf, size_t len);
void polvo_p2p_close_write(int64_t s);
void polvo_p2p_close(int64_t s);
void polvo_p2p_shutdown(int64_t h);
void polvo_p2p_free(char *s);
*/
import "C"

import (
	"errors"
	"io"
	"net"
	"sync"
	"unsafe"
)

// take gives the Go string of text the C API gave, and frees it.
func take(s *C.char) string {
	if s == nil {
		return ""
	}
	defer C.polvo_p2p_free(s)
	return C.GoString(s)
}

type cPeer struct {
	h    C.int64_t
	id   string
	once sync.Once
	l    *listener
}

func open(seed []byte, relay string) (Peer, error) {
	cr := C.CString(relay)
	defer C.free(unsafe.Pointer(cr))
	var cerr *C.char
	h := C.polvo_p2p_open((*C.uint8_t)(unsafe.Pointer(&seed[0])), cr, &cerr)
	if h < 0 {
		// TRNSP-E01
		return nil, errors.New(take(cerr))
	}
	p := &cPeer{h: h, id: take(C.polvo_p2p_id(h))}
	p.l = &listener{p: p}
	return p, nil
}

func (p *cPeer) ID() string             { return p.id }
func (p *cPeer) Addr() string           { return take(C.polvo_p2p_addr(p.h)) }
func (p *cPeer) Listener() net.Listener { return p.l }

func (p *cPeer) Dial(addr string) (*Conn, error) {
	ca := C.CString(addr)
	defer C.free(unsafe.Pointer(ca))
	var cerr *C.char
	s := C.polvo_p2p_connect(p.h, ca, &cerr)
	if s < 0 {
		// TRNSP-E01
		return nil, errors.New(take(cerr))
	}
	return &Conn{stream: &cStream{s: s}, local: Addr(p.id), remote: remoteOf(addr)}, nil
}

func (p *cPeer) Close() error {
	p.once.Do(func() { C.polvo_p2p_shutdown(p.h) })
	return nil
}

// listener gives the connections the endpoint accepts.
type listener struct{ p *cPeer }

// Accept waits for the next connection; net.ErrClosed once the endpoint is closed.
func (l *listener) Accept() (net.Conn, error) {
	var remote, cerr *C.char
	s := C.polvo_p2p_accept(l.p.h, &remote, &cerr)
	switch {
	case s == 0:
		return nil, net.ErrClosed
	case s < 0:
		return nil, errors.New(take(cerr))
	}
	return &Conn{stream: &cStream{s: s}, local: Addr(l.p.id), remote: Addr(take(remote))}, nil
}

func (l *listener) Close() error   { return l.p.Close() }
func (l *listener) Addr() net.Addr { return Addr(l.p.id) }

type cStream struct {
	s    C.int64_t
	once sync.Once
}

func (c *cStream) read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := C.polvo_p2p_read(c.s, (*C.uint8_t)(unsafe.Pointer(&p[0])), C.size_t(len(p)))
	switch {
	case n == 0:
		return 0, io.EOF
	case n < 0:
		return 0, net.ErrClosed
	}
	return int(n), nil
}

func (c *cStream) write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := C.polvo_p2p_write(c.s, (*C.uint8_t)(unsafe.Pointer(&p[0])), C.size_t(len(p)))
	if n < 0 {
		return 0, net.ErrClosed
	}
	return int(n), nil
}

func (c *cStream) closeWrite() { C.polvo_p2p_close_write(c.s) }
func (c *cStream) close()      { c.once.Do(func() { C.polvo_p2p_close(c.s) }) }
