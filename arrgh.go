// Copyright ©2014 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package arrgh provides an interface to R via an OpenCPU server.
//
// Interaction with the OpenCPU system is via the OpenCPU API https://www.opencpu.org/api.html.
// Data serialisation and deserialisation at the R end is performed by jsonlite, see
// http://cran.r-project.org/web/packages/jsonlite/jsonlite.pdf for the jsonlite manual.
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
	cmd  *exec.Cmd
	host *url.URL
	root string
}

// NewLocalSession starts an R instance using the executable in the given
// path or the executable "R" in the user's $PATH if path is empty. An OpenCPU
// server is started using the provided port and connection is tested before
// returning. If no connection is possible within the timeout, a nil session and
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
	path, err = exec.LookPath(path)
	if err != nil {
		return nil, err
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
	sess.root = pth.Join("/", root)
	control, err := sess.cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	err = sess.cmd.Start()
	if err != nil {
		return nil, err
	}

	// If people ask why this package has the name it does, just point to this;
	// a version return function in base returns a type that is not accepted by
	// a package whose sole purpose is to parse version values.
	const startServer = `library(opencpu)
library(semver)
if (parse_version(as.character(packageVersion("opencpu"))) < "2.0.0") {
	opencpu$start(%[1]d)
} else {
	ocpu_start_server(port=%[1]d)
}
`
	fmt.Fprintf(control, startServer, port)

	runtime.SetFinalizer(&sess, func(s *Session) { s.Close() })

	start := time.Now()
	u := sess.host.String()
	for {
		time.Sleep(time.Second)
		_, err := http.Get(u)
		if err == nil {
			return &sess, nil
		} else if timeout > 0 && time.Now().Sub(start) > timeout {
			sess.cmd.Process.Kill()
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
	return s.cmd.Process.Kill()
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
	sess.root = pth.Join("/", root)

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
// type using the POST method. The URL parameters specify additional POST parameters.
// These parameters are interpreted by jsonlite.
//
// See https://www.opencpu.org/api.html#api-methods and https://www.opencpu.org/api.html#api-arguments for details.
func (s *Session) Post(path, content string, params url.Values, query io.Reader) (*http.Response, error) {
	if s.host == nil {
		return nil, errors.New("arrgh: POST on closed session")
	}
	u := *s.host
	u.Path = pth.Join(s.host.Path, path)
	u.RawQuery = params.Encode()
	return http.Post(u.String(), content, query)
}

// Get retrieves the given OpenCPU path using the GET method. The URL parameters specify
// GET parameters which are interpreted by jsonlite.
//
// See https://www.opencpu.org/api.html#api-methods for details.
func (s *Session) Get(path string, params url.Values) (*http.Response, error) {
	if s.host == nil {
		return nil, errors.New("arrgh: GET on closed session")
	}
	u := *s.host
	u.Path = pth.Join(s.host.Path, path)
	u.RawQuery = params.Encode()
	return http.Get(u.String())
}

// Params is a collection of parameter names and values to be passed using Multipart.
type Params map[string]string

// NamedReader allows an io.Reader to be passed as a named data file object.
type NamedReader interface {
	io.Reader
	Name() string
}

// Params is a collection of parameter names and file objects to be passed using Multipart.
type Files map[string]NamedReader

// Multipart constructs a MIME multipart body and associated content type from the
// provided parameters and files.
func Multipart(parameters Params, files Files) (content string, body io.Reader, err error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for label, f := range files {
		p, err := w.CreateFormFile(label, filepath.Base(f.Name()))
		if err != nil {
			return "", nil, err
		}
		_, err = io.Copy(p, f)
		if err != nil {
			return "", nil, err
		}
	}

	for k, v := range parameters {
		err := w.WriteField(k, v)
		if err != nil {
			return "", nil, err
		}
	}

	err = w.Close()
	if err != nil {
		return "", nil, err
	}

	return w.FormDataContentType(), &buf, nil
}
