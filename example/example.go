package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"github.com/shynome/go-fsnet/dev"
	"github.com/shynome/wcgi"
)

func init() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "index")
	})
	http.HandleFunc("/hello1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello1")
	})
	http.HandleFunc("/hello2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello2")
	})
	http.HandleFunc("/cat-index", func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer err0.Then(&err, nil, func() {
			http.Error(w, err.Error(), 500)
		})
		var client = &http.Client{
			Transport: dev.Transport,
		}
		index := "http://" + r.Host + "/"
		req := try.To1(http.NewRequest("GET", index, nil))
		resp := try.To1(client.Do(req))
		io.WriteString(w, "cat-index\n")
		io.Copy(w, resp.Body)
	})
}

func main() {
	if err := wcgi.Serve(nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
