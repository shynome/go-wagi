package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func TestWagi(t *testing.T) {
	args.addr = getRandomAddr()
	go runWagi()

	caddyAddr := getRandomAddr()
	config := fmt.Sprintf(caddyConfigTpl, caddyAddr, args.addr)
	caddy := runCaddy(config)
	defer caddy.Kill()

	time.Sleep(2 * time.Second) // wait server ready

	httpGet := func(path string) (s string, err error) {
		defer err2.Handle(&err)
		link := fmt.Sprintf("http://%s%s", caddyAddr, path)
		res := try.To1(http.Get(link))
		b := try.To1(io.ReadAll(res.Body))
		return string(b), nil
	}

	t.Run("base", func(t *testing.T) {
		index := try.To1(httpGet("/"))
		assert.Equal(index, "index\n")

		hello1 := try.To1(httpGet("/hello1"))
		assert.Equal(hello1, "hello1\n")

		hello2 := try.To1(httpGet("/hello2"))
		assert.Equal(hello2, "hello2\n")
	})
	t.Run("fsnet", func(t *testing.T) {
		fsnet := try.To1(httpGet("/cat-index"))
		assert.Equal(fsnet, "cat-index\nindex\n")
	})
}

func TestMain(m *testing.M) {
	compileWasm()
	m.Run()
}

var testPWD = filepath.Join(try.To1(os.Getwd()), "../../")

func compileWasm() {
	cmd := exec.Command("make", "build-demo")
	cmd.Dir = testPWD
	try.To(cmd.Run())
}

var caddyConfigTpl = `
{
  admin off
}

http://%s {
  root ./example
  route {
    php_fastcgi %s {
      env WASI_NET allow
    }
    respond 404
  }
}
`

func runCaddy(conf string) *os.Process {
	cmd := exec.Command("caddy", "run", "-config", "-", "-adapter", "caddyfile")
	cmd.Dir = testPWD
	cmd.Stdin = bytes.NewBufferString(conf)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	try.To(cmd.Start())
	return cmd.Process
}

func getRandomAddr() string {
	l := try.To1(net.Listen("tcp", "127.0.0.1:0"))
	defer l.Close()
	return l.Addr().String()
}
