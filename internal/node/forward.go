// @anchors
//   code: SRFWD
//   ref: SRVRE

package node

import (
	"io"
	"net"
	"strconv"
	"sync"

	"golang.org/x/crypto/ssh"
)

// forwardKey carries whether a connection may forward sockets (SRVRE-B13).
const forwardKey = "polvo-forward"

// The payloads of the forwarding messages (RFC 4254 §7, and OpenSSH's PROTOCOL for the Unix sockets).
type directTCP struct {
	Host     string
	Port     uint32
	OrigHost string
	OrigPort uint32
}

type directUnix struct {
	Path      string
	Reserved0 string
	Reserved1 uint32
}

type tcpForward struct {
	Addr string
	Port uint32
}

type forwardedTCP struct {
	Addr     string
	Port     uint32
	OrigAddr string
	OrigPort uint32
}

type unixForward struct{ Path string }

type forwardedUnix struct {
	Path     string
	Reserved string
}

// forwards are the listeners a connection asked the node for (SRVRE-B13), closed with it.
type forwards struct {
	conn ssh.Conn
	may  bool

	mu sync.Mutex
	ls map[string]net.Listener
}

func newForwards(conn ssh.Conn, may bool) *forwards {
	return &forwards{conn: conn, may: may, ls: map[string]net.Listener{}}
}

// closeAll ends every listener of the connection.
func (f *forwards) closeAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, l := range f.ls {
		l.Close()
		delete(f.ls, k)
	}
}

// requests answers the connection's global requests: the forwards from the node to the client
// (SRVRE-B13); the others, and all without the permission, are refused.
func (f *forwards) requests(reqs <-chan *ssh.Request) {
	for r := range reqs {
		ok, reply := false, []byte(nil)
		if f.may {
			switch r.Type {
			case "tcpip-forward":
				ok, reply = f.listenTCP(r.Payload)
			case "cancel-tcpip-forward":
				var p tcpForward
				ok = ssh.Unmarshal(r.Payload, &p) == nil && f.cancel("tcp:"+net.JoinHostPort(p.Addr, strconv.Itoa(int(p.Port))))
			case "streamlocal-forward@openssh.com":
				ok = f.listenUnix(r.Payload)
			case "cancel-streamlocal-forward@openssh.com":
				var p unixForward
				ok = ssh.Unmarshal(r.Payload, &p) == nil && f.cancel("unix:"+p.Path)
			}
		}
		if r.WantReply {
			r.Reply(ok, reply)
		}
	}
}

func (f *forwards) listenTCP(payload []byte) (bool, []byte) {
	var p tcpForward
	if ssh.Unmarshal(payload, &p) != nil {
		return false, nil
	}
	l, err := net.Listen("tcp", net.JoinHostPort(p.Addr, strconv.Itoa(int(p.Port))))
	if err != nil {
		return false, nil
	}
	port := uint32(l.Addr().(*net.TCPAddr).Port)
	f.keep("tcp:"+net.JoinHostPort(p.Addr, strconv.Itoa(int(port))), l)
	go f.accept(l, func(c net.Conn) (string, []byte) {
		orig, _ := c.RemoteAddr().(*net.TCPAddr)
		var oa string
		var op uint32
		if orig != nil {
			oa, op = orig.IP.String(), uint32(orig.Port)
		}
		return "forwarded-tcpip", ssh.Marshal(forwardedTCP{Addr: p.Addr, Port: port, OrigAddr: oa, OrigPort: op})
	})
	var answer []byte
	if p.Port == 0 {
		// the port the node chose
		answer = ssh.Marshal(struct{ Port uint32 }{port})
	}
	return true, answer
}

func (f *forwards) listenUnix(payload []byte) bool {
	var p unixForward
	if ssh.Unmarshal(payload, &p) != nil {
		return false
	}
	l, err := net.Listen("unix", p.Path)
	if err != nil {
		return false
	}
	f.keep("unix:"+p.Path, l)
	go f.accept(l, func(net.Conn) (string, []byte) {
		return "forwarded-streamlocal@openssh.com", ssh.Marshal(forwardedUnix{Path: p.Path})
	})
	return true
}

func (f *forwards) keep(key string, l net.Listener) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ls[key] = l
}

func (f *forwards) cancel(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.ls[key]
	if ok {
		l.Close()
		delete(f.ls, key)
	}
	return ok
}

// accept takes each connection to the listener to the client, in a channel of the kind open gives.
func (f *forwards) accept(l net.Listener, open func(net.Conn) (string, []byte)) {
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		go func() {
			kind, payload := open(c)
			ch, reqs, err := f.conn.OpenChannel(kind, payload)
			if err != nil {
				c.Close()
				return
			}
			go ssh.DiscardRequests(reqs)
			pipe(ch, c)
		}()
	}
}

// direct serves a channel to a socket of the node's machine (SRVRE-B13): the client's side of a forward
// from the client to the node.
func (f *forwards) direct(nc ssh.NewChannel) {
	if !f.may {
		// SRVRE-B08
		nc.Reject(ssh.Prohibited, "node: this key may not forward sockets")
		return
	}
	var network, addr string
	switch nc.ChannelType() {
	case "direct-tcpip":
		var p directTCP
		if ssh.Unmarshal(nc.ExtraData(), &p) != nil {
			nc.Reject(ssh.ConnectionFailed, "node: bad direct-tcpip request")
			return
		}
		network, addr = "tcp", net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port)))
	default:
		var p directUnix
		if ssh.Unmarshal(nc.ExtraData(), &p) != nil {
			nc.Reject(ssh.ConnectionFailed, "node: bad direct-streamlocal request")
			return
		}
		network, addr = "unix", p.Path
	}
	c, err := net.Dial(network, addr)
	if err != nil {
		// SRVRE-E04
		nc.Reject(ssh.ConnectionFailed, "node: "+err.Error())
		return
	}
	ch, reqs, err := nc.Accept()
	if err != nil {
		c.Close()
		return
	}
	go ssh.DiscardRequests(reqs)
	pipe(ch, c)
}

// pipe copies between the channel and the socket until both sides end.
func pipe(ch ssh.Channel, c net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(ch, c)
		ch.CloseWrite()
	}()
	go func() {
		defer wg.Done()
		io.Copy(c, ch)
		if cw, ok := c.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite()
		} else {
			c.Close()
		}
	}()
	wg.Wait()
	ch.Close()
	c.Close()
}
