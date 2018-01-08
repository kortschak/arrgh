arrgh (Pronunciation: /ɑː/ or /är/) is an interface to the [OpenCPU](https://www.opencpu.org/) R server system.

[![Build Status](https://travis-ci.org/kortschak/arrgh.svg?branch=master)](https://travis-ci.org/kortschak/arrgh) [![Coverage Status](https://coveralls.io/repos/kortschak/arrgh/badge.svg?branch=master&service=github)](https://coveralls.io/github/kortschak/arrgh?branch=master) [![GoDoc](https://godoc.org/github.com/kortschak/arrgh?status.svg)](https://godoc.org/github.com/kortschak/arrgh)

## Overview

The arrgh package provides API interfaces to remote or local OpenCPU R servers.

Go is a well established network systems language and has seen increasing use in data science and other fields of scientific software development.
The arrgh package allows developers to leverage the rich statistical analysis environment available through R that is lacking in the Go ecosystem.

## Example

```
package main

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

func main() {
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
		strings.NewReader("x="+url.QueryEscape("coef(lm(speed ~ dist, data = cars))")),
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
```

Output:
```
tmp/x0113a3ca85/R/identity:
function (x) 
x
<bytecode: 0x55d13e1f40a0>
<environment: namespace:base>


tmp/x0113a3ca85/R/.val:
(Intercept)        dist 
  8.2839056   0.1655676 


tmp/x0113a3ca85/stdout:
(Intercept)        dist 
  8.2839056   0.1655676 


tmp/x0113a3ca85/source:
identity(x = coef(lm(speed ~ dist, data = cars)))

tmp/x0113a3ca85/console:
> identity(x = coef(lm(speed ~ dist, data = cars)))
(Intercept)        dist 
  8.2839056   0.1655676 

tmp/x0113a3ca85/info:
R version 3.4.1 (2017-06-30)
Platform: x86_64-pc-linux-gnu (64-bit)
Running under: Ubuntu 16.04.2 LTS

Matrix products: default
BLAS: /usr/lib/openblas-base/libblas.so.3
LAPACK: /usr/lib/libopenblasp-r0.2.18.so

locale:
 [1] LC_CTYPE=en_US.UTF-8    LC_NUMERIC=C            LC_TIME=en_US.UTF-8    
 [4] LC_COLLATE=en_US.UTF-8  LC_MONETARY=en_US.UTF-8 LC_MESSAGES=C          
 [7] LC_PAPER=C              LC_NAME=C               LC_ADDRESS=C           
[10] LC_TELEPHONE=C          LC_MEASUREMENT=C        LC_IDENTIFICATION=C    

attached base packages:
[1] stats     graphics  grDevices utils     datasets  methods   base     

other attached packages:
[1] opencpu_2.0.3.1

loaded via a namespace (and not attached):
 [1] Rcpp_0.12.11     lattice_0.20-35  mime_0.5         plyr_1.8.4      
 [5] grid_3.4.1       gtable_0.2.0     sys_1.4          jsonlite_1.5    
 [9] unix_1.3         magrittr_1.5     scales_0.4.1     evaluate_0.10.2 
[13] ggplot2_2.2.1    rlang_0.1.1      stringi_1.1.5    curl_2.7        
[17] lazyeval_0.2.0   webutils_0.6     tools_3.4.1      stringr_1.2.0   
[21] munsell_0.4.3    parallel_3.4.1   sendmailR_1.2-1  compiler_3.4.1  
[25] colorspace_1.3-2 base64enc_0.1-3  openssl_0.9.6    tibble_1.3.3    


tmp/x0113a3ca85/files/DESCRIPTION:
Package: x0113a3ca85
Type: Session
Version: 2.0.3.1
Author: OpenCPU
Date: 2017-07-07
Description: This file is automatically generated by OpenCPU.


lm: intercept=8.283906 dist=0.165568
```


## Installation

arrgh requires a [Go](http://golang.org) installation, and if using a local R instance [OpenCPU](https://www.opencpu.org/download.html) (tested on v1.6 and v2.0) and [semver](https://cran.r-project.org/web/packages/semver/index.html) must be installed as R packages.

```
go get github.com/kortschak/arrgh
```

## Documentation

http://godoc.org/github.com/kortschak/arrgh

## Getting help

Help or similar requests can be asked on the bug tracker, or for more general OpenCPU questions at the OpenCPU google groups.

https://groups.google.com/forum/#!forum/opencpu

## Contributing

If you find any bugs, feel free to file an issue on the github issue tracker.
Pull requests are welcome.

## License

arrgh is distributed under a modified BSD license.
