// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

// Command bench provides benchmarking tools for sat solvers.
//  ⎣ ⇨ bench
//  bench <cmd> [options] arg args ...
//  <cmd> may be
//  	sel
//  	run
//  	cmp
//  For help with a command, run bench <cmd> -h.
//
//  ⎣ ⇨ bench sel -h
//  sel [seloptions] dir [ dir [ dir ... ] ]
//  	sel selects files and puts them in bench suite format.
//    -link
//      	symlink files to new names instead of copying
//    -n int
//      	number of files to select (default 100)
//    -name string
//      	put benchmark in this directory (default "bench")
//    -pattern string
//      	match this pattern when selecting files.
//
//  ⎣ ⇨ bench run -h
//  run [runoptions] suite [ suite [ suite ... ] ]
//  	run runs commands on benchmark suites enforcing timeouts and
//  	recording results.
//    -cmd string
//      	command to run on each instance
//    -commit string
//      	commit id of command
//    -d string
//      	delete the run
//    -desc string
//      	description of run.
//    -dur duration
//      	max per-instance duration (default 5s)
//    -gdur duration
//      	max run duration (default 1h0m0s)
//    -name string
//      	name of the run (default "run")
//
//  ⎣ ⇨ bench cmp -h
//   cmp [cmp options] suite
//     -cactus
//       	cactus plot
//     -list
//       	list all instances in all runs.
//     -sat
//       	list only sat instances in runs.
//     -scatter
//       	scatter plot of run pairs
//     -sum
//       	suite summary. (default true)
//     -unsat
//       	list only unsat instances in runs.
package main
