package fsnet_test

import (
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/shynome/err0/try"
	"github.com/shynome/go-fsnet/dev"
	devnet "github.com/shynome/go-fsnet/dev"
	"github.com/shynome/go-wagi/fsnet"
)

var (
	l1Addr string
	l2Addr string
)

func TestMain(m *testing.M) {

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	})

	l1 := try.To1(net.Listen("tcp", "127.0.0.1:0"))
	defer l1.Close()
	l1Addr = l1.Addr().String()
	go http.Serve(l1, h)

	l2 := try.To1(net.Listen("tcp", "127.0.0.1:0"))
	defer l2.Close()
	l2Addr = l2.Addr().String()
	go http.Serve(l2, h)

	m.Run()
}

func TestNet(t *testing.T) {
	t.Run("reject all", func(t *testing.T) {
		rule := "bypass=~0.0.0.0/0"
		setFileOpener(fsnet.New(rule))

		check(t, false, false)
	})

	t.Run("reject one", func(t *testing.T) {
		rule := "bypass=~" + l1Addr
		setFileOpener(fsnet.New(rule))

		check(t, false, true)
	})

	t.Run("reject two", func(t *testing.T) {
		rule := "bypass=~" + l1Addr + "&bypass=~" + l2Addr
		setFileOpener(fsnet.New(rule))

		check(t, false, false)
	})

	t.Run("allow empty", func(t *testing.T) {
		rule := ""
		setFileOpener(fsnet.New(rule))

		check(t, false, false)
	})

	t.Run("allow all", func(t *testing.T) {
		rule := "bypass=0.0.0.0/0"
		setFileOpener(fsnet.New(rule))

		check(t, true, true)
	})

	t.Run("allow one", func(t *testing.T) {
		rule := "bypass=" + l1Addr
		setFileOpener(fsnet.New(rule))

		check(t, true, false)
	})

	t.Run("allow one reject second", func(t *testing.T) {
		rule := "bypass=" + l1Addr + "&bypass=~" + l2Addr
		setFileOpener(fsnet.New(rule))

		check(t, true, false)
	})

	t.Run("reject other but allow second", func(t *testing.T) {
		rule := "bypass=~0.0.0.0/0,::/0,*&bypass=~" + l1Addr + "&bypass=" + l2Addr
		setFileOpener(fsnet.New(rule))

		check(t, false, true)
	})
}

var client = &http.Client{Transport: dev.Transport}

func check(t *testing.T, ok1, ok2 bool) {
	addrs := []string{l1Addr, l2Addr}
	for i, ok := range []bool{ok1, ok2} {
		addr := addrs[i]
		if ok {
			if resp, err := client.Get("http://" + addr); err != nil {
				t.Error(i, err)
				return
			} else {
				defer resp.Body.Close()
				if code := resp.StatusCode; code != http.StatusOK {
					t.Error(i, code)
					return
				}
				body := try.To1(io.ReadAll(resp.Body))
				if body := string(body); body != "ok" {
					t.Error(i, body)
					return
				}
			}
			continue
		} else {
			if resp, err := client.Get("http://" + addr); err == nil {
				t.Error(resp.Status)
				t.Error(i, "the host should be rejected")
				return
			} else {
				t.Log(err)
			}
		}
	}
}

func setFileOpener(fsn fs.FS) {
	devnet.SetFileOpener(func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
		f, err := fsn.Open(strings.TrimPrefix(name, "/dev/"))
		if err != nil {
			return nil, err
		}
		return f.(io.ReadWriteCloser), nil
	})
}
