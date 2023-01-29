package fsnet

import (
	"io"
	"io/fs"
	"net"
	"time"
)

type FileConn struct {
	conn net.Conn
	path string
}

var _ fs.File = (*FileConn)(nil)

func NewFileConn(name string, conn net.Conn) *FileConn {
	return &FileConn{
		conn: conn,
		path: name,
	}
}

func (fc *FileConn) Stat() (fs.FileInfo, error) { return fc, nil }
func (fc *FileConn) Read(p []byte) (n int, err error) {
	n, err = fc.conn.Read(p)
	return
}
func (fc *FileConn) Close() error { return fc.conn.Close() }

var _ io.Writer = (*FileConn)(nil)

func (fc *FileConn) Write(p []byte) (n int, err error) {
	n, err = fc.conn.Write(p)
	return
}

var _ fs.FileInfo = (*FileConn)(nil)

func (info *FileConn) Name() string       { return info.path }
func (info *FileConn) Size() int64        { return 4096 }
func (info *FileConn) Mode() fs.FileMode  { return fs.ModeType }
func (info *FileConn) ModTime() time.Time { return time.Now() }
func (info *FileConn) IsDir() bool        { return false }
func (info *FileConn) Sys() any           { return nil }
