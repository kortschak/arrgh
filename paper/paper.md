---
title: 'arrgh: a Go interface to the OpenCPU R server system'
tags:
  - statistics
  - R
  - golang
authors:
 - name: R Daniel Kortschak
   orcid: 0000-0001-8295-2301
   affiliation: 1
affiliations:
 - name: School of Biological Sciences, The University of Adelaide
   index: 1
date: 7 July 2017
bibliography: paper.bib
---

# Summary

OpenCPU [@OpenCPU] provides an interface to the R system for statistical computing [@R].
The OpenCPU API provides a number of HTTP endpoints and a JSON Remote Procedure Call protocol [@API], either to a remote host or to a single user local server.

Go is a simple statically typed compiled language that provides many benefits for scientific computing.
However, Go currently lacks the rich statistical analysis tools available in R.
So an interface between R and Go would allow building analytical tools utilising the simplicity and high performance of Go and the statistical analysis tools available within R.
The arrgh package is intended to be used by Go developers wanting to add rich statistical analysis to Go analytical tools.

The arrgh package provides a Go client for the OpenCPU system, including the management of a local server when needed.
Data exchange between Go and OpenCPU is via standard HTTP content types, with arrgh providing tools to facilitate encoding and endpoint targeting.

# References
