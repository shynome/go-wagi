package fsnet

import (
	"crypto/tls"
	"fmt"
	"io/fs"
	"net"
	"path/filepath"
	"regexp"
	"time"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type Net struct {
	dir string
}

var _ fs.FS = (*Net)(nil)

func New(dir string) *Net {
	return &Net{dir: dir}
}

func (n *Net) Open(name string) (f fs.File, err error) {
	defer err2.Handle(&err)
	name = filepath.Join(n.dir, name)
	file := getAddress(name)
	if file == nil {
		return n.Default(name)
	}
	address := fmt.Sprintf("%s:%s", file.host, file.port)
	var conn net.Conn
	if file.tls {
		conn = try.To1(tls.Dial(file.network, address, nil))
	} else {
		conn = try.To1(net.Dial(file.network, address))
	}
	return NewFileConn(name, conn), nil
}

func (n *Net) Default(name string) (f fs.File, err error) {
	if name == "/" {
		return Dir("/"), nil
	}
	return nil, fs.ErrNotExist
}

var addrSpitter = regexp.MustCompile(`^\/dev\/(tcp|udp)\/(.+)\/(\d+)(/tls|)$`)

type netAddr struct {
	network string
	host    string
	port    string
	tls     bool
}

func getAddress(name string) *netAddr {
	n := addrSpitter.FindStringSubmatch(name)
	if len(n) != 5 {
		return nil
	}
	return &netAddr{
		network: n[1],
		host:    n[2],
		port:    n[3],
		tls:     n[4] != "",
	}
}

type Dir string

var _ fs.File = (*Dir)(nil)

func (fc Dir) Stat() (fs.FileInfo, error) { return fc, nil }
func (fc Dir) Read(p []byte) (int, error) { return 4096, fs.ErrClosed }
func (fc Dir) Close() error               { return nil }

var _ fs.FileInfo = (*Dir)(nil)

func (info Dir) Name() string       { return string(info) }
func (info Dir) Size() int64        { return 0 }
func (info Dir) Mode() fs.FileMode  { return fs.FileMode(0644) }
func (info Dir) ModTime() time.Time { return time.Now() }
func (info Dir) IsDir() bool        { return true }
func (info Dir) Sys() any           { return info }
