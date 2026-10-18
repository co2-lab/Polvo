// @anchors
//   code: DSTST
//   ref: DSCVR

package discovery

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// testGroup is the mDNS group on the loopback at a free port, apart from the machine's mDNS.
func testGroup(t *testing.T) Options {
	t.Helper()
	lo, err := net.InterfaceByName("lo")
	if err != nil {
		t.Skip("no loopback interface")
	}
	free, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	port := free.LocalAddr().(*net.UDPAddr).Port
	free.Close()
	return Options{Group: &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: port}, Interface: lo}
}

func announce(t *testing.T, s Service, o Options) {
	t.Helper()
	a, err := Announce(s, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
}

// ask sends the question from conn to the group and reads the answer on reply.
func ask(t *testing.T, conn, reply *net.UDPConn, o Options) dnsmessage.Message {
	t.Helper()
	if _, err := conn.WriteToUDP(question(), o.Group); err != nil {
		t.Fatal(err)
	}
	reply.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 9000)
	for {
		n, _, err := reply.ReadFromUDP(buf)
		if err != nil {
			t.Fatalf("no answer: %v", err)
		}
		var m dnsmessage.Message
		if m.Unpack(buf[:n]) == nil && m.Header.Response {
			return m
		}
	}
}

func unicast(t *testing.T) *net.UDPConn {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func describe(m dnsmessage.Message) string {
	var out []string
	for _, r := range append(m.Answers, m.Additionals...) {
		switch b := r.Body.(type) {
		case *dnsmessage.PTRResource:
			out = append(out, "PTR "+b.PTR.String())
		case *dnsmessage.SRVResource:
			out = append(out, fmt.Sprintf("SRV %s %d", b.Target, b.Port))
		case *dnsmessage.TXTResource:
			out = append(out, "TXT "+strings.Join(b.TXT, " "))
		case *dnsmessage.AResource:
			out = append(out, "A "+net.IP(b.A[:]).String())
		}
	}
	sort.Strings(out)
	return strings.Join(out, "; ")
}

var oficina = Service{Instance: "oficina", Host: "oficina", Port: 2222, Fingerprint: "SHA256:abc", IPs: []net.IP{net.IPv4(192, 0, 2, 7)}}

func TestDiscovery_announce(t *testing.T) {
	t.Run("DSCVR-B01: Anunciar", func(t *testing.T) {
		o := testGroup(t)
		announce(t, oficina, o)
		c := unicast(t)
		got := describe(ask(t, c, c, o))
		want := "A 192.0.2.7; PTR oficina._polvo._tcp.local.; SRV oficina.local. 2222; TXT fp=SHA256:abc name=oficina"
		if got != want {
			t.Fatalf("want %s, got %s", want, got)
		}
	})
	t.Run("DSCVR-B03: A quem se responde", func(t *testing.T) {
		o := testGroup(t)
		announce(t, oficina, o)
		member, err := net.ListenMulticastUDP("udp4", o.Interface, o.Group)
		if err != nil {
			t.Fatal(err)
		}
		defer member.Close()
		// from the group's port, the answer comes to the group
		inGroup := ask(t, member, member, o)
		// from another port, to that port
		c := unicast(t)
		direct := ask(t, c, c, o)
		if len(inGroup.Answers) == 0 || len(direct.Answers) == 0 {
			t.Fatalf("want both answered, got %v and %v", inGroup, direct)
		}
	})
	t.Run("DSCVR-E01: Um grupo que não se abre", func(t *testing.T) {
		o := testGroup(t)
		o.Group = &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: o.Group.Port}
		if _, err := Announce(oficina, o); err == nil {
			t.Fatal("want the error of a group that is not multicast")
		}
	})
	t.Run("DSCVR-E02: Um pacote que não é DNS", func(t *testing.T) {
		o := testGroup(t)
		announce(t, oficina, o)
		c := unicast(t)
		c.WriteToUDP([]byte("lixo"), o.Group)
		if m := ask(t, c, c, o); len(m.Answers) == 0 {
			t.Fatalf("want the answer, got %v", m)
		}
	})
}

func TestDiscovery_browse(t *testing.T) {
	t.Run("DSCVR-B02: Procurar", func(t *testing.T) {
		o := testGroup(t)
		announce(t, Service{Instance: "a", Host: "a", Port: 1, Fingerprint: "SHA256:a", IPs: []net.IP{net.IPv4(127, 0, 0, 1)}}, o)
		announce(t, Service{Instance: "b", Host: "b", Port: 2, Fingerprint: "SHA256:b", IPs: []net.IP{net.IPv4(127, 0, 0, 1)}}, o)
		found, err := Browse(context.Background(), 500*time.Millisecond, o)
		if err != nil {
			t.Fatal(err)
		}
		sort.Slice(found, func(i, j int) bool { return found[i].Instance < found[j].Instance })
		got := fmt.Sprint(found)
		if len(found) != 2 || found[0] != (Found{Instance: "a", Name: "a", Addr: "127.0.0.1:1", Fingerprint: "SHA256:a"}) || found[1].Addr != "127.0.0.1:2" {
			t.Fatalf("got %s", got)
		}
	})
	t.Run("DSCVR-B04: O endereço de um nó sem A", func(t *testing.T) {
		o := testGroup(t)
		announce(t, Service{Instance: "sem", Host: "sem", Port: 7, Fingerprint: "SHA256:s"}, o)
		found, err := Browse(context.Background(), 500*time.Millisecond, o)
		if err != nil || len(found) != 1 {
			t.Fatalf("got %v %v", found, err)
		}
		// the answer came from one of this machine's addresses, which the kernel chose
		host, port, _ := net.SplitHostPort(found[0].Addr)
		mine := false
		addrs, _ := net.InterfaceAddrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.String() == host {
				mine = true
			}
		}
		if !mine || port != "7" {
			t.Fatalf("want an address of this machine with port 7, got %s", found[0].Addr)
		}
	})
}
