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

	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"github.com/shynome/go-wagi"
)

var args struct {
	addr string
	ttl  time.Duration
}

var f *flag.FlagSet
var Version = "dev"

func init() {
	f = flag.NewFlagSet("wagi "+Version, flag.ExitOnError)
	f.StringVar(&args.addr, "addr", "127.0.0.1:7071", "")
	f.DurationVar(&args.ttl, "cachettl", 15*time.Second, "")
}

func main() {
	f.Parse(os.Args[1:])

	runWagi()
}

func runWagi() {
	runtime := wagi.NewWagi(wagi.WagiConfig{
		CacheTTL: args.ttl,
	})
	l := try.To1(net.Listen("tcp", args.addr))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer err0.Then(&err, nil, func() {
			w.WriteHeader(500)
			fmt.Fprintf(w, "%s \r\n", err)
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
	log.Println(f.Name(), "is running on:", l.Addr().String())
	try.To(fcgi.Serve(l, h))
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
