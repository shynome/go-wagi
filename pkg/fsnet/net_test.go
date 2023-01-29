package fsnet

import (
	"fmt"
	"io"
	"net"
	"net/netip"
	"testing"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func TestNet(t *testing.T) {
	l := try.To1(net.Listen("tcp", "127.0.0.1:0"))
	defer l.Close()
	var w = "hello world"
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				break
			}
			go func(conn net.Conn) {
				defer conn.Close()
				io.WriteString(conn, w)
			}(conn)
		}
	}()
	fsnet := &Net{}

	p := try.To1(netip.ParseAddrPort(l.Addr().String()))
	addr := fmt.Sprintf("/dev/tcp/127.0.0.1/%d", p.Port())
	f := try.To1(fsnet.Open(addr))
	defer f.Close()
	b := try.To1(io.ReadAll(f))
	assert.Equal(string(b), w)
}

func TestParseAddress(t *testing.T) {
	cases := [][]string{
		{"/dev/tcp/0/0", "tcp", "0:0", ""},
		{"/dev/tcp/0/0/tls", "tcp", "0:0", "tls"},
		{"/dev/tcp/[::0]/0", "tcp", "[::0]:0", ""},
		{"/dev/tcp/[::0]/0/tls", "tcp", "[::0]:0", "tls"},
		{"/dev/udp/0/0", "udp", "0:0", ""},
		{"/dev/udp/0/0/tls", "udp", "0:0", "tls"},
		{"/dev/udp/[::0]/0", "udp", "[::0]:0", ""},
		{"/dev/udp/[::0]/0/tls", "udp", "[::0]:0", "tls"},
	}

	for _, v := range cases {
		t.Run("parse addr "+v[0], func(t *testing.T) {
			a := getAddress(v[0])
			assert.Equal(a.network, v[1])
			assert.Equal(fmt.Sprintf("%s:%s", a.host, a.port), v[2])
			assert.Equal(a.tls, v[3] != "")
		})
	}
}
