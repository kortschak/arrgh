// Copyright ©2014 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Not shown for windows due to path separator discordance.

// +build !windows

package arrgh_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kortschak/arrgh"
)

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
		nil,
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
		// API root path stripping here depends on consistency
		// between os.Separator and the URL path separator.
		p, err := filepath.Rel(r.Root(), sc.Text())
		if err != nil {
			log.Fatal(err)
		}
		if path.Base(p) == ".val" {
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
