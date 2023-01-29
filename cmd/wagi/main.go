package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"net/http"
	"net/http/fcgi"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/shynome/go-wagi"
)

var args struct {
	addr string
	ttl  time.Duration
	dir  string
}

func init() {
	flag.StringVar(&args.addr, "addr", "127.0.0.1:7071", "")
	flag.DurationVar(&args.ttl, "cachettl", 15*time.Second, "")
	flag.StringVar(&args.dir, "cachedir", ".", "")
}

func main() {
	flag.Parse()

	runtime := wagi.NewWagi(wagi.WagiConfig{
		CacheTTL: args.ttl,
		CacheDir: args.dir,
	})
	l := try.To1(net.Listen("tcp", args.addr))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer err2.Catch(func(err error) {
			responseServerError(w, err)
		})
		env := fcgi.ProcessEnv(r)
		h := wagi.Handler{
			Wagi: runtime,
			Path: env["SCRIPT_FILENAME"],
			Env:  env,
		}
		exists := try.To1(fileExists(h.Path))
		if !exists {
			w.WriteHeader(404)
			return
		}
		h.ServeHTTP(w, r)
	})
	log.Println("wasi fastcgi server is running on:", l.Addr().String())
	try.To(fcgi.Serve(l, h))
}

func responseServerError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	w.WriteHeader(500)
	fmt.Fprintf(w, "%s \r\n", err)
}

func fileExists(name string) (bool, error) {
	fileinfo, err := os.Stat(name)
	if err == nil {
		if fileinfo.IsDir() {
			return false, nil
		}
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
