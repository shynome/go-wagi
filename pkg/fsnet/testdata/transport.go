package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/shynome/go-wagi/pkg/fsnet/dev"
)

type DevTCP struct{}

var _ http.RoundTripper = (*DevTCP)(nil)

var tcpdev = &DevTCP{}

var portsMap = map[string]string{
	"https": "443",
	"http":  "80",
}

func (*DevTCP) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	defer err2.Handle(&err)
	var f io.ReadWriteCloser
	var port = req.URL.Port()
	if port == "" {
		port = portsMap[req.URL.Scheme]
	}
	if os.Args[0] == "wasm" {
		return dev.Transport.RoundTrip(req)
	} else {
		raddr := fmt.Sprintf("%s:%s", req.URL.Hostname(), port)
		if req.URL.Scheme == "https" {
			f = try.To1(tls.Dial("tcp", raddr, nil))
		} else {
			f = try.To1(net.Dial("tcp", raddr))
		}
	}
	req.Write(f)
	resp = try.To1(http.ReadResponse(bufio.NewReader(f), req))
	return
}
