package main

import (
	"fmt"
	"net/http"
	"net/http/cgi"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "index")
	})
	http.HandleFunc("/hello1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello1")
	})
	http.HandleFunc("/hello2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello2")
	})
	if err := cgi.Serve(nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
