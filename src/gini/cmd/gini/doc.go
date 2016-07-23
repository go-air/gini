// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

// Package main implements the Gini command.
//
//  gini usage: gini <input> <input> <input> ...\n
//  gini reads dimacs cnf inputs and tries to solve them. The inputs
//  may be gzipped or bzip2ed.
//
//  By default, gini prints to stdout one line indicating the result, one of 'sat',
//  'unsat', or 'unknown'.  Following this line, it may print a model if the
//  problem is sat.
//
//  If no inputs are specified, or the input is '-',  gini reads from stdin.
//
//  When gini runs with -satcomp, only one input may be specified and the output and
//  exit codes are in sat competition format.
//
//
//  Additionally, gini has the following options:
//
//    -assume value
//      	add an assumption (default [])
//    -crisp string
//      	address of crisp server to use
//    -failed
//      	output failed assumptions
//    -model
//      	output model (default false)
//    -mon
//      	if true, print statistics during solving (default false, implies -stats)
//    -pprof string
//      	address to serve http profile (eg :6060)
//    -satcomp
//      	if true, exit 10 sat, 20 unsat and output dimacs (default false)
//    -stats
//      	if true, print some statistics after solving (default false)
//    -timeout duration
//      	timeout (default 30s)
//
//
package main
