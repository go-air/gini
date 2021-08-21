# Gini SAT Solver

Gini is a fast SAT solver written in Go.

[![GoDoc](https://godoc.org/github.com/go-air/gini?status.svg)](https://godoc.org/github.com/go-air/gini)

[Google Group](https://groups.google.com/d/forum/ginisat) 


## Build/Install

For the impatient:

    go get github.com/go-air/gini/...

Then run 'gini' or 'bench'

```
gini usage: gini <input> <input> <input> ...\n
gini reads dimacs cnf or icnf inputs and tries to solve them. The inputs
may be gzipped or bzip2ed.

If the input is '-',  gini reads from stdin.  

When gini runs with -satcomp, and there are no inputs, gini reads from stdin. Also, 
only one input may be specified and the output and exit codes are in sat 
competition format:
	- 10 for sat
	- 20 for unsat
	-  0 for unknown

Additionally, gini has the following options:

  -assume value
    	add an assumption
  -ax
    	run the assumption exchanger for icnf inputs
  -crisp string
    	address of crisp server to use
  -failed
    	output failed assumptions
  -model
    	output model (default false)
  -mon duration
    	if non-zero, print statistics during solving (default 0, implies -stats)
  -pprof string
    	address to serve http profile (eg :6060)
  -satcomp
    	if true, exit 10 sat, 20 unsat and output dimacs (default false)
  -stats
    	if true, print some statistics after solving (default false)
  -timeout duration
    	timeout (default 30s)

```

```
bench <cmd> [options] arg arg ...
<cmd> may be
	sel
	run
	cmp
For help with a command, run bench <cmd> -h.
```

## The SAT problem

[satprob.md](satprob.md)


## Features

Our [User guide](manual.md) gives an overview of Gini's features.






