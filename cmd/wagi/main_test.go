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

	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"github.com/stretchr/testify/assert"
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
		defer err0.Then(&err, nil, nil)
		link := fmt.Sprintf("http://%s%s", caddyAddr, path)
		res := try.To1(http.Get(link))
		b := try.To1(io.ReadAll(res.Body))
		return string(b), nil
	}

	t.Run("base", func(t *testing.T) {
		index := try.To1(httpGet("/"))
		assert.Equal(t, index, "index\n")

		hello1 := try.To1(httpGet("/hello1"))
		assert.Equal(t, hello1, "hello1\n")

		hello2 := try.To1(httpGet("/hello2"))
		assert.Equal(t, hello2, "hello2\n")
	})
	t.Run("fsnet", func(t *testing.T) {
		fsnet := try.To1(httpGet("/cat-index"))
		assert.Equal(t, fsnet, "cat-index\nindex\n")
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
	cmd := exec.Command("caddy", "run", "--config", "-", "--adapter", "caddyfile")
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
