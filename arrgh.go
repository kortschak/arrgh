// Copyright ©2014 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package arrgh provides an interface to R via an OpenCPU server.
//
// Interaction with the OpenCPU system is via the OpenCPU API https://www.opencpu.org/api.html.
package arrgh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	pth "path"
	"path/filepath"
	"runtime"
	"time"
)

// Session holds OpenCPU session connection information.
type Session struct {
	cmd     *exec.Cmd
	host    *url.URL
	control io.Writer
	root    string
}

// NewLocalSession starts an R instance using the executable in the given
// path or the executable "R" in the user's $PATH if path is empty. An OpenCPU
// server is started using the provided port and connection is tested before
// returning if no connection is possible within the timeout, a nil session and
// an error are returned. The root of the OpenCPU API is set to "/ocpu" if it is
// left empty.
//
// It is important that Close() be called on sessions returned by NewLocalSession.
func NewLocalSession(path, root string, port int, timeout time.Duration) (*Session, error) {
	var (
		sess Session
		err  error
	)

	if path == "" {
		path = "R"
	}
	if filepath.Base(path) == path {
		path, err = exec.LookPath(path)
		if err != nil {
			return nil, err
		}
	}
	sess.cmd = exec.Command(path, "--vanilla", "--slave")
	sess.host, err = url.Parse(fmt.Sprintf("http://localhost:%d/", port))
	if err != nil {
		panic(fmt.Sprintf("arrgh: unexpected error: %v", err))
	}
	if root == "" {
		root = "ocpu"
	}
	sess.host.Path = pth.Join(sess.host.Path, root)
	sess.root = filepath.Join("/", root)
	sess.control, err = sess.cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	err = sess.cmd.Start()
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(sess.control, "library(opencpu); opencpu$start(%d)\n", port)

	runtime.SetFinalizer(&sess, func(s *Session) { s.Close() })

	start := time.Now()
	u := sess.host.String()
	for {
		time.Sleep(time.Second)
		_, err := http.Get(u)
		if err == nil {
			return &sess, nil
		} else if timeout > 0 && time.Now().Sub(start) > timeout {
			sess.control.Write([]byte("opencpu$stop()\n"))
			return nil, err
		}
	}
}

// Root returns the OpenCPU root path.
func (s *Session) Root() string { return s.root }

// Close shuts down a running local session, terminating the OpenCPU server
// and the R session. It is a no-op on a remote session.
func (s *Session) Close() error {
	if s.cmd == nil || s.host == nil {
		return nil
	}
	s.host = nil
	s.control.Write([]byte("opencpu$stop()\nq()\n"))
	s.control = nil
	return s.cmd.Wait()
}

// NewRemoteSession connects to the OpenCPU server at the specified host. The
// root of the OpenCPU API is set to "/ocpu" if it is left empty.
func NewRemoteSession(host, root string, timeout time.Duration) (*Session, error) {
	var (
		sess Session
		err  error
	)
	sess.host, err = url.Parse(host)
	if err != nil {
		return nil, err
	}
	if root == "" {
		root = "ocpu"
	}
	sess.host.Path = pth.Join(sess.host.Path, root)
	sess.root = filepath.Join("/", root)

	start := time.Now()
	u := sess.host.String()
	for {
		time.Sleep(time.Second)
		_, err := http.Get(u)
		if err == nil {
			return &sess, nil
		} else if timeout > 0 && time.Now().Sub(start) > timeout {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				return &sess, err
			}
			return nil, err
		}
	}
}

// Post sends the query content to the given OpenCPU path as the specified content
// type using the POST method.
//
// See https://www.opencpu.org/api.html#api-methods and https://www.opencpu.org/api.html#api-arguments for details.
func (s *Session) Post(path, content string, query io.Reader) (*http.Response, error) {
	if s.host == nil {
		return nil, errors.New("arrgh: POST on closed session")
	}
	u := *s.host
	u.Path = pth.Join(s.host.Path, path)
	return http.Post(u.String(), content, query)
}

// Get sends the query to the given OpenCPU path using the GET method.
//
// See https://www.opencpu.org/api.html#api-methods for details.
func (s *Session) Get(path, query string) (*http.Response, error) {
	if s.host == nil {
		return nil, errors.New("arrgh: GET on closed session")
	}
	u := *s.host
	u.Path = pth.Join(s.host.Path, path)
	return http.Get(u.String())
}

// Params is a collection of parameter names and values to be passed using PostMultipart.
type Params map[string]string

// NamedReader allows a io.Reader to be passed as named data file objects.
type NamedReader interface {
	io.Reader
	Name() string
}

// Params is a collection of parameter names and file objects to be passed using PostMultipart.
type Files map[string]NamedReader

// PostMultipart send the query content to the given OpenCPU path as the "multipart/form-data"
// content type using the POST method.
//
// See https://www.opencpu.org/api.html#api-methods and https://www.opencpu.org/api.html#api-arguments for details.
func (s *Session) PostMultipart(path string, p Params, f Files) (*http.Response, error) {
	if s.host == nil {
		return nil, errors.New("arrgh: POST on closed session")
	}
	u := *s.host
	u.Path = pth.Join(s.host.Path, path)
	return multi(u.String(), p, f)
}

func multi(url string, params Params, files Files) (*http.Response, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for label, f := range files {
		p, err := w.CreateFormFile(label, filepath.Base(f.Name()))
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(p, f)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range params {
		err := w.WriteField(k, v)
		if err != nil {
			return nil, err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	return http.Post(url, w.FormDataContentType(), &buf)
}
