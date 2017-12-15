// Copyright ©2017 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The nuccore program summarises the lengths of sequences in the nuccore
// database in Entrez that are linked from genomes studies.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/biogo/ncbi"
	"github.com/biogo/ncbi/entrez"
	"github.com/kortschak/arrgh"
)

const (
	genomes = "genome"
	nuccore = "nuccore"
	tool    = "arrgh.example"
)

func main() {
	ncbi.SetTimeout(0)

	var (
		png   = flag.String("png", "", "specifies the out file name for a lenght distribution plot.")
		email = flag.String("email", "", "specifies the email address to be sent to the server (required).")
		help  = flag.Bool("help", false, "prints this message.")
	)

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if *email == "" {
		flag.Usage()
		os.Exit(2)
	}

	// Collate eukaryotic genome projects and find unique
	// records in the nuccore database that are linked from
	// these projects.
	p := &entrez.Parameters{RetMax: 3000}
	s, err := entrez.DoSearch(genomes, "eukaryota", p, nil, tool, *email)
	if err != nil {
		log.Fatalf("search error: %v\n", err)
	}
	l, err := entrez.DoLink(genomes, nuccore, "", "", p, tool, *email, nil, s.IdList)
	if err != nil {
		log.Fatalf("link error: %v\n", err)
	}
	uids := make(map[int]struct{})
	for _, ls := range l.LinkSets {
		for _, e := range ls.IdList {
			uids[e.Id] = struct{}{}
		}
	}
	var ids []int
	for id := range uids {
		ids = append(ids, id)
	}

	// Collect summary documents from the unique records
	// and retain the length item values.
	sum, err := entrez.DoSummary(nuccore, p, tool, *email, nil, ids...)
	if err != nil {
		log.Fatalf("summary error: %v\n", err)
	}
	var lengths []int
	for _, doc := range sum.Documents {
		for _, it := range doc.Items {
			if it.Name == "Length" && it.Type == "Integer" {
				l, err := strconv.Atoi(it.Value)
				if err != nil {
					log.Fatalf("error: %v\n", err)
				}
				lengths = append(lengths, l)
			}
		}
	}
	var buf bytes.Buffer
	for i, l := range lengths {
		if i != 0 {
			fmt.Fprint(&buf, ",")
		}
		fmt.Fprint(&buf, l)
	}

	// Open a session on the public.opencpu.org OpenCPU server.
	r, err := arrgh.NewRemoteSession("http://public.opencpu.org", "", 10*time.Second)
	if err != nil {
		log.Fatalf("error opening opencpu connection: %v", err)
	}
	defer r.Close()

	// Send a small script with the lengths embedded into it.
	// The script calculates the basic summary statistics of
	// the lengths an plots a ggplot2 plot using the density
	// aesthetic.
	resp, err := r.Post(
		"library/base/R/identity",
		"application/x-www-form-urlencoded",
		nil,
		strings.NewReader("x="+url.QueryEscape(fmt.Sprintf(`
library("ggplot2")

data <- c(%s)
sum <- summary(data)
df <- as.data.frame(data)
ggplot(df, aes(x=data)) +
	geom_density() +
	ggtitle("Plot of genome study-associated nuccore sequence lengths") +
	theme(plot.title = element_text(hjust = 0.5))
`,
			&buf))),
	)
	if err != nil {
		log.Fatalf("error posting to opencpu: %v", err)
	}
	defer resp.Body.Close()

	// Get each part of the session result and check it,
	// retaining the summary and the plot if requested.
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		// API root path stripping here depends on consistency
		// between os.Separator and the URL path separator.
		p, err := filepath.Rel(r.Root(), sc.Text())
		if err != nil {
			log.Fatal(err)
		}
		switch {
		case *png != "" && strings.Contains(p, "graphics/1"):
			p = filepath.Join(p, "png")
			resp, err := r.Get(p, nil)
			if err != nil {
				log.Fatal(err)
			}
			out, err := os.Create(*png)
			if err != nil {
				log.Fatalf("error creating plot file: %v", err)
			}
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Fatalf("error writing plot file: %v", err)
			}
			err = out.Close()
			if err != nil {
				log.Fatalf("error closing plot file: %v", err)
			}
			resp.Body.Close()
		case path.Base(p) == "sum":
			resp, err := r.Get(p, nil)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Summary statistics for genome study-associated nuccore sequence lengths:\n")
			io.Copy(os.Stdout, resp.Body)
			fmt.Print("\n")
			resp.Body.Close()
		}
	}
}
