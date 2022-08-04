package main

import (
	"errors"
	"fmt"
	"net"
	"os"

	"net/http"
	"net/http/fcgi"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/shynome/go-wagi"
)

func main() {
	runtime := wagi.NewWagi()
	l := try.To1(net.Listen("tcp", "127.0.0.1:7071"))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer responseServerError(w, err)
		defer err2.Return(&err)
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
