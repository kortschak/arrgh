// Copyright ©2017 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package arrgh

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var sessionTests = []struct {
	path    string
	content string
	params  url.Values
	query   func() io.Reader

	want string
}{
	{
		path:    "library/base/R/identity",
		content: "application/x-www-form-urlencoded",
		params:  nil,
		query: func() io.Reader {
			return strings.NewReader("x=" + url.QueryEscape("coef(lm(speed ~ dist, data = cars))"))
		},

		want: "[8.2839056418, 0.16556757464]\n",
	},
}

func TestLocalSession(t *testing.T) {
	r, err := NewLocalSession("", "", 3000, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to start local opencpu session: %v", err)
	}
	defer r.Close()

tests:
	for _, test := range sessionTests {
		resp, err := r.Post(test.path, test.content, test.params, test.query())
		if err != nil {
			t.Errorf("unexpected error for POST: %v", err)
			continue
		}
		defer resp.Body.Close()

		sc := bufio.NewScanner(resp.Body)
		var val string
		for sc.Scan() {
			p, err := filepath.Rel(r.Root(), sc.Text())
			if err != nil {
				t.Errorf("failed to get relative filepath: %v", err)
				continue tests
			}
			if path.Base(p) == ".val" {
				val = p
				break
			}
		}

		res, err := r.Get(path.Join(val, "json"), url.Values{"digits": []string{"10"}})
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		got, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Errorf("unexpected error reading result: %v", err)
			continue
		}

		if string(got) != test.want {
			t.Errorf("unexpected result: got:%q want:%q", got, test.want)
		}
	}
}

func TestRemoteSession(t *testing.T) {
	r, err := NewRemoteSession("http://public.opencpu.org", "", 10*time.Second)
	if err != nil {
		t.Fatalf("failed to start local opencpu session: %v", err)
	}
	defer r.Close()

tests:
	for _, test := range sessionTests {
		resp, err := r.Post(test.path, test.content, test.params, test.query())
		if err != nil {
			t.Errorf("unexpected error for POST: %v", err)
			continue
		}
		defer resp.Body.Close()

		sc := bufio.NewScanner(resp.Body)
		var val string
		for sc.Scan() {
			p, err := filepath.Rel(r.Root(), sc.Text())
			if err != nil {
				t.Errorf("failed to get relative filepath: %v", err)
				continue tests
			}
			if path.Base(p) == ".val" {
				val = p
				break
			}
		}

		res, err := r.Get(path.Join(val, "json"), url.Values{"digits": []string{"10"}})
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		got, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Errorf("unexpected error reading result: %v", err)
			continue
		}

		if string(got) != test.want {
			t.Errorf("unexpected result: got:%q want:%q", got, test.want)
		}
	}
}

var multipartTests = []struct {
	params Params
	files  Files
}{
	{
		params: Params{"header": "bar", "baz": "qux"},
		files:  Files{"boop": namedReader{name: "boop", ReadSeeker: strings.NewReader("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")}},
	},
	{
		params: Params{"header": "FALSE"},
		files: Files{"mydata.csv": func() *os.File {
			f, err := os.Open("mydata.csv")
			if err != nil {
				panic("failed to to open test file")
			}
			return f
		}(),
		},
	},
}

func TestMultipart(t *testing.T) {
	for _, test := range multipartTests {
		content, body, err := Multipart(test.params, test.files)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if !strings.HasPrefix(content, "multipart/form-data; boundary=") {
			t.Errorf("unexpected content string: got:%q", content)
		}

		typ, params, err := mime.ParseMediaType(content)
		if err != nil {
			t.Errorf("failed to parse MIME type: %v", err)
			continue
		}
		if !strings.HasPrefix(typ, "multipart/") {
			t.Error("expected multipart MIME")
			continue
		}
		mr := multipart.NewReader(body, params["boundary"])

		gotParams := make(Params)
	parts:
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("failed to parse MIME part: %v", err)
				continue parts
			}
			b, err := ioutil.ReadAll(p)
			if err != nil {
				t.Errorf("failed to read MIME part: %v", err)
				continue parts
			}
			if name := p.FileName(); name != "" {
				rs := test.files[name].(io.ReadSeeker)
				_, err := rs.Seek(0, os.SEEK_SET)
				if err != nil {
					t.Fatalf("failed to seek to start: %v", err)
				}
				want, err := ioutil.ReadAll(rs)
				if err != nil {
					t.Fatalf("failed to read want text: %v", err)
				}
				if !bytes.Equal(b, want) {
					t.Errorf("unexpected file content: got:%q want:%q", b, want)
				}
			} else if name := p.FormName(); name != "" {
				gotParams[name] = string(b)
			}
		}

		if !reflect.DeepEqual(gotParams, test.params) {
			t.Errorf("unexpected parameters: got:%v want:%v", gotParams, test.params)
		}
	}
}

type namedReader struct {
	name string
	io.ReadSeeker
}

func (r namedReader) Name() string { return r.name }
