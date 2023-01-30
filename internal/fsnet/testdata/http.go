package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/lainio/err2/try"
	"github.com/shynome/go-fsnet/dev"
)

func main() {
	switch os.Args[1] {
	case "http":
		testhttp()
	case "tcp":
		testtcp()
	default:
		fmt.Println("default")
	}
}

func testhttp() {
	var client = &http.Client{Transport: dev.Transport}
	req := try.To1(http.NewRequest(http.MethodGet, os.Args[2], nil))
	resp := try.To1(client.Do(req))
	fmt.Println("status", resp.Status)
	try.To1(io.Copy(os.Stdout, resp.Body))
	fmt.Println("http")
}

func testtcp() {
	f := try.To1(
		os.OpenFile(os.Args[2], os.O_RDWR, os.ModePerm))
	defer f.Close()
	try.To1(f.Write([]byte("bodyyyyyyyyyyyyyyy\n")))
	b := try.To1(io.ReadAll(f))
	fmt.Println(string(b))
	fmt.Println("tcp")
}
