// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package main

var usage = `%s usage: %s <input> <input> <input> ...
%s reads dimacs cnf or icnf inputs and tries to solve them. The inputs
may be gzipped or bzip2ed.

If the input is '-',  %s reads from stdin.  

When %s runs with -satcomp, and there are no inputs, %s reads from stdin. Also, 
only one input may be specified and the output and exit codes are in sat 
competition format:
	- 10 for sat
	- 20 for unsat
	-  0 for unknown

Additionally, %s has the following options:

`
