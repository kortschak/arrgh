// Copyright ©2014 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package arrgh_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kortschak/arrgh"
)

func Example_1() {
	r, err := arrgh.NewLocalSession("", 3000, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	resp, err := r.Post(
		"library/stats/R/rnorm/json",
		"application/json",
		strings.NewReader(`{"n":10, "mean": 10, "sd":10}`),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var rnorm []float64
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&rnorm)

	fmt.Println(rnorm, err)
}

func Example_2() {
	r, err := arrgh.NewRemoteSession("http://public.opencpu.org", 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

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

	io.Copy(os.Stdout, resp.Body)
}
