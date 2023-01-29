package dev

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DevTCP struct{}

var tcpdev = &DevTCP{}

var portsMap = map[string]string{
	"https": "443",
	"http":  "80",
}

var Transport http.RoundTripper = tcpdev

func (*DevTCP) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var port = req.URL.Port()
	if port == "" {
		port = portsMap[req.URL.Scheme]
	}
	raddr := fmt.Sprintf("/dev/tcp/%s/%s", req.URL.Hostname(), port)
	if req.URL.Scheme == "https" {
		raddr += "/tls"
	}
	var f io.ReadWriteCloser
	f, err = os.OpenFile(raddr, os.O_RDWR, os.ModePerm)
	if err != nil {
		return
	}
	req.Write(f)
	resp, err = http.ReadResponse(bufio.NewReader(f), req)
	return
}
