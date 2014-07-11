// Copyright ©2014 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package arrgh_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kortschak/arrgh"
)

func Example_1() {
	r, err := arrgh.NewLocalSession("", "", 3000, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Send a query to get a JSON representation of results
	// from rnorm(n=10, mean=10, sd=10).
	resp, err := r.Post(
		"library/stats/R/rnorm/json",
		"application/json",
		strings.NewReader(`{"n":10, "mean": 10, "sd":10}`),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Decode the results in to a slice of float64.
	var rnorm []float64
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&rnorm)

	fmt.Println(rnorm, err)
}

func Example_2() {
	r, err := arrgh.NewRemoteSession("http://public.opencpu.org", "", 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Send a query to get a session result for the linear
	// regression: coef(lm(speed~dist, data=cars)).
	resp, err := r.Post(
		"library/base/R/identity",
		"application/x-www-form-urlencoded",
		strings.NewReader(`x=coef(lm(speed ~ dist, data = cars))`),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Get each part of the session result and display it,
	// keeping the location of the linear regression.
	sc := bufio.NewScanner(resp.Body)
	var val string
	for sc.Scan() {
		p, err := filepath.Rel(r.Root(), sc.Text())
		if err != nil {
			log.Fatal(err)
		}
		if filepath.Base(p) == ".val" {
			val = p
		}
		fmt.Printf("%s:\n", p)

		resp, err := r.Get(p, nil)
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(os.Stdout, resp.Body)
		fmt.Print("\n\n")
		resp.Body.Close()
	}

	// Get the linear regression result as JSON.
	res, err := r.Get(path.Join(val, "json"), url.Values{"digits": []string{"10"}})
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	// Decode the result into a [2]float64.
	var lm [2]float64
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&lm)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("lm: intercept=%f dist=%f\n", lm[0], lm[1])
}

func mask(r io.Reader) io.Reader {
	re := regexp.MustCompile("x[0-9a-f]{10}")
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return bytes.NewReader(re.ReplaceAll(buf.Bytes(), []byte("xXXXXXXXXXX")))
}

func Example_3() {
	r, err := arrgh.NewRemoteSession("http://public.opencpu.org", "", 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Upload the contents of the file "mydata.csv" and send
	// it to the read.csv function.
	f, err := os.Open("mydata.csv")
	if err != nil {
		log.Fatal(err)
	}
	resp, err := r.PostMultipart(
		"library/utils/R/read.csv",
		nil,
		arrgh.Files{"file": f},
	)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, mask(resp.Body))

	// Output:
	//
	// /ocpu/tmp/xXXXXXXXXXX/R/.val
	// /ocpu/tmp/xXXXXXXXXXX/stdout
	// /ocpu/tmp/xXXXXXXXXXX/source
	// /ocpu/tmp/xXXXXXXXXXX/console
	// /ocpu/tmp/xXXXXXXXXXX/info
	// /ocpu/tmp/xXXXXXXXXXX/files/mydata.csv
}
