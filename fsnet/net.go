package fsnet

import (
	"context"
	"io/fs"
	"path/filepath"
	"time"

	bypass2 "github.com/go-gost/core/bypass"
	"github.com/shynome/go-fsnet"
	devnet "github.com/shynome/go-fsnet/dev/net"
)

type Net struct {
	fs.FS
	bp   bypass2.Bypass
	rule string
}

func New(rule string) fs.FS {
	return &Net{
		FS:   fsnet.New("/dev/"),
		rule: rule,
		bp:   ParseBypass(rule),
	}
}

var _ fs.FS = (*Net)(nil)

func (n *Net) Open(name string) (fs.File, error) {
	switch name {
	case ".":
		return n.FS.Open(name)
	}
	path := filepath.Join("/dev/", name)
	addr, err := devnet.ParseAddr(path)
	if err != nil {
		return nil, err
	}
	addr.Address()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	saddr := addr.Address()
	pass := n.bp.Contains(ctx, addr.NetType, saddr)
	if !pass {
		return nil, fs.ErrNotExist
	}
	return n.FS.Open(name)
}
