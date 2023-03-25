package fsnet

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
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
	"github.com/lainio/err2/try"
	"github.com/shynome/go-fsnet"
	"github.com/tetratelabs/wazero"
	gojs "github.com/tetratelabs/wazero/experimental/gojs"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

const testChunk = "http"

func TestWasmNet(t *testing.T) {
	// select {}
	stderr, stdout := bytes.Buffer{}, bytes.Buffer{}
	defer err2.Catch(func(err error) {
		stderrStr := stderr.String()
		t.Error(err, stderrStr)
	})

	ctx := context.Background()
	compileWasm()
	rt, m := initRuntimeAndModule(ctx)

	config := wazero.NewModuleConfig()

	fsc := wazero.NewFSConfig()
	fsc = fsc.WithFSMount(fsnet.New("/dev/"), "/dev")
	config = config.
		WithArgs("wasm").
		WithFSConfig(fsc).
		WithStdout(&stdout).WithStderr(&stderr)

	switch testChunk {
	case "http":
		config = config.WithArgs("wasm", "http", "http://127.0.0.1:7072")
	case "tcp":
		config = config.WithArgs("wasm", "tcp", "/dev/tcp/127.0.0.1/7072")
	}

	err := gojs.Run(ctx, rt, m, gojs.NewConfig(config))
	// _, err := rt.InstantiateModule(ctx, m, config)
	if e, ok := err.(*sys.ExitError); ok && e.ExitCode() == 0 {
		err = nil
	}
	try.To(err)

	stdoutStr := stdout.String()
	t.Log(stdoutStr)
}

func httpServer(l net.Listener) {
	http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer err2.Catch(func(err error) {
			http.Error(w, err.Error(), 400)
		})
		rr := try.To1(io.ReadAll(r.Body))
		_ = rr
		for i := 0; i < 5; i++ {
			io.WriteString(w, "hello world\n")
			if w, ok := w.(http.Flusher); ok {
				w.Flush()
			}
			fmt.Println(time.Now())
			time.Sleep(time.Second)
		}
	}))
}

func tcpServer(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			break
		}
		go func(conn net.Conn) {
			defer conn.Close()
			line, _ := try.To2(
				bufio.NewReader(conn).ReadLine())
			fmt.Println(string(line))
			io.WriteString(conn, "HTTP/1.1 200 OK\n")
			io.WriteString(conn, "\n")
			io.WriteString(conn, "hello world\n")
		}(conn)
	}
}

var l = func() (l net.Listener) {
	l = try.To1(net.Listen("tcp", "127.0.0.1:7072"))
	switch testChunk {
	case "http":
		go httpServer(l)
	case "tcp":
		go tcpServer(l)
	}
	return
}()

func compileWasm() {
	pwd := try.To1(os.Getwd())
	pwd = filepath.Join(pwd, "testdata")
	cmd := exec.Command("make")
	cmd.Dir = pwd
	try.To(cmd.Run())
}

func initRuntimeAndModule(ctx context.Context) (rt wazero.Runtime, m wazero.CompiledModule) {
	rtc := wazero.NewRuntimeConfig().
		WithDebugInfoEnabled(true)
	rt = wazero.NewRuntimeWithConfig(ctx, rtc)
	gojs.MustInstantiate(ctx, rt)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	code := try.To1(os.ReadFile("testdata/test.wasm"))
	m = try.To1(rt.CompileModule(ctx, code))
	return
}
