// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the host side of CGI (being the webserver
// parent process).

// Package cgi implements CGI (Common Gateway Interface) as specified
// in RFC 3875.
//
// Note that using CGI means starting a new process to handle each
// request, which is typically less efficient than using a
// long-running server. This package is intended primarily for
// compatibility with existing systems.
package wagi

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/tetratelabs/wazero"
	"golang.org/x/net/http/httpguts"
)

var trailingPort = regexp.MustCompile(`:([0-9]+)$`)

// Handler runs an executable in a subprocess with a CGI environment.
type Handler struct {
	Path string // path to the CGI executable
	Root string // root URI prefix of handler or empty for "/"

	Env        map[string]string // extra environment variables to set, if any, as "key=value"
	InheritEnv []string          // environment variables to inherit from host, as "key"
	Logger     *log.Logger       // optional log for errors or nil to use log.Print
	Args       []string          // optional arguments to pass to child process
	Stderr     io.Writer         // optional stderr for the child process; nil means os.Stderr
	Wagi       WASIRuntime

	// PathLocationHandler specifies the root http Handler that
	// should handle internal redirects when the CGI process
	// returns a Location header value starting with a "/", as
	// specified in RFC 3875 § 6.3.2. This will likely be
	// http.DefaultServeMux.
	//
	// If nil, a CGI response with a local URI path is instead sent
	// back to the client and not redirected internally.
	PathLocationHandler http.Handler
}

func (h *Handler) stderr() io.Writer {
	if h.Stderr != nil {
		return h.Stderr
	}
	return os.Stderr
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	root := h.Root
	if root == "" {
		root = "/"
	}

	if len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked" {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("Chunked request bodies are not supported by CGI."))
		return
	}

	pathInfo := req.URL.Path
	if root != "/" && strings.HasPrefix(pathInfo, root) {
		pathInfo = pathInfo[len(root):]
	}

	port := "80"
	if matches := trailingPort.FindStringSubmatch(req.Host); len(matches) != 0 {
		port = matches[1]
	}

	env := map[string]string{
		"SERVER_SOFTWARE":   "go",
		"SERVER_NAME":       req.Host,
		"SERVER_PROTOCOL":   "HTTP/1.1",
		"HTTP_HOST":         req.Host,
		"GATEWAY_INTERFACE": "CGI/1.1",
		"REQUEST_METHOD":    req.Method,
		"QUERY_STRING":      req.URL.RawQuery,
		"REQUEST_URI":       req.URL.RequestURI(),
		"PATH_INFO":         pathInfo,
		"SCRIPT_NAME":       root,
		"SCRIPT_FILENAME":   h.Path,
		"SERVER_PORT":       port,
	}

	if remoteIP, remotePort, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		env["REMOTE_ADDR"] = remoteIP
		env["REMOTE_HOST"] = remoteIP
		env["REMOTE_PORT"] = remotePort
	} else {
		// could not parse ip:port, let's use whole RemoteAddr and leave REMOTE_PORT undefined
		env["REMOTE_ADDR"] = remoteIP
		env["REMOTE_HOST"] = remoteIP
	}

	if req.TLS != nil {
		env["HTTPS"] = "on"
	}

	for k, v := range req.Header {
		k = strings.Map(upperCaseAndUnderscore, k)
		if k == "PROXY" {
			// See Issue 16405
			continue
		}
		joinStr := ", "
		if k == "COOKIE" {
			joinStr = "; "
		}
		env["HTTP_"+k] = strings.Join(v, joinStr)
	}

	if req.ContentLength > 0 {
		env["CONTENT_LENGTH"] = fmt.Sprintf("%d", req.ContentLength)
	}
	if ctype := req.Header.Get("Content-Type"); ctype != "" {
		env["CONTENT_TYPE"] = ctype
	}

	for _, e := range h.InheritEnv {
		if v := os.Getenv(e); v != "" {
			env[e] = v
		}
	}

	if h.Env != nil {
		for k, v := range h.Env {
			env[k] = v
		}
	}

	internalError := func(err error) {
		if err == nil {
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		h.printf("CGI error: %v", err)
	}
	var err error
	defer internalError(err)

	stdoutRead, stdoutWrite := io.Pipe()
	defer stdoutRead.Close()
	args := append([]string{h.Path}, h.Args...)

	config := wazero.NewModuleConfig().
		WithStdout(stdoutWrite).
		WithArgs(args...).
		WithStderr(h.stderr())
	if req.ContentLength != 0 {
		config = config.WithStdin(req.Body)
	}
	for k, v := range env {
		config = config.WithEnv(k, v)
	}

	go func() { // start wasi module
		defer stdoutWrite.Close()
		if err = h.Wagi.Run(h.Path, config); err != nil {
			fmt.Fprintf(os.Stderr, "run %s failed err: %s\n", h.Path, err)
			return
		}
	}()

	linebody := bufio.NewReaderSize(stdoutRead, 1024)
	headers := make(http.Header)
	statusCode := 0
	headerLines := 0
	sawBlankLine := false
	for {
		line, isPrefix, err := linebody.ReadLine()
		if isPrefix {
			rw.WriteHeader(http.StatusInternalServerError)
			h.printf("cgi: long header line from subprocess.")
			return
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			h.printf("cgi: error reading headers: %v", err)
			return
		}
		if len(line) == 0 {
			sawBlankLine = true
			break
		}
		headerLines++
		header, val, ok := strings.Cut(string(line), ":")
		if !ok {
			h.printf("cgi: bogus header line: %s", string(line))
			continue
		}
		if !httpguts.ValidHeaderFieldName(header) {
			h.printf("cgi: invalid header name: %q", header)
			continue
		}
		val = textproto.TrimString(val)
		switch {
		case header == "Status":
			if len(val) < 3 {
				h.printf("cgi: bogus status (short): %q", val)
				return
			}
			code, err := strconv.Atoi(val[0:3])
			if err != nil {
				h.printf("cgi: bogus status: %q", val)
				h.printf("cgi: line was %q", line)
				return
			}
			statusCode = code
		default:
			headers.Add(header, val)
		}
	}
	if headerLines == 0 || !sawBlankLine {
		rw.WriteHeader(http.StatusInternalServerError)
		h.printf("cgi: no headers")
		return
	}

	if loc := headers.Get("Location"); loc != "" {
		if strings.HasPrefix(loc, "/") && h.PathLocationHandler != nil {
			h.handleInternalRedirect(rw, req, loc)
			return
		}
		if statusCode == 0 {
			statusCode = http.StatusFound
		}
	}

	if statusCode == 0 && headers.Get("Content-Type") == "" {
		rw.WriteHeader(http.StatusInternalServerError)
		h.printf("cgi: missing required Content-Type in headers")
		return
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// Copy headers to rw's headers, after we've decided not to
	// go into handleInternalRedirect, which won't want its rw
	// headers to have been touched.
	for k, vv := range headers {
		for _, v := range vv {
			rw.Header().Add(k, v)
		}
	}

	rw.WriteHeader(statusCode)

	_, err = io.Copy(rw, linebody)
	if err != nil {
		h.printf("cgi: copy error: %v", err)
	}
}

func (h *Handler) printf(format string, v ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func (h *Handler) handleInternalRedirect(rw http.ResponseWriter, req *http.Request, path string) {
	url, err := req.URL.Parse(path)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		h.printf("cgi: error resolving local URI path %q: %v", path, err)
		return
	}
	// TODO: RFC 3875 isn't clear if only GET is supported, but it
	// suggests so: "Note that any message-body attached to the
	// request (such as for a POST request) may not be available
	// to the resource that is the target of the redirect."  We
	// should do some tests against Apache to see how it handles
	// POST, HEAD, etc. Does the internal redirect get the same
	// method or just GET? What about incoming headers?
	// (e.g. Cookies) Which headers, if any, are copied into the
	// second request?
	newReq := &http.Request{
		Method:     "GET",
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       url.Host,
		RemoteAddr: req.RemoteAddr,
		TLS:        req.TLS,
	}
	h.PathLocationHandler.ServeHTTP(rw, newReq)
}

func upperCaseAndUnderscore(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - ('a' - 'A')
	case r == '-':
		return '_'
	case r == '=':
		// Maybe not part of the CGI 'spec' but would mess up
		// the environment in any case, as Go represents the
		// environment as a slice of "key=value" strings.
		return '_'
	}
	// TODO: other transformations in spec or practice?
	return r
}
