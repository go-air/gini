// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

// Package bench provides solver benchmarking utilities.
//
// Package bench addresses the needs of solver benchmarking by:
//
// 1. providing a format for a benchmark suite.  The format is command line
// friendly for unix commands.
//
// 2. providing a tool to create such benchmarks suites.
//
// 3. providing a format for runs on benchmarks by a solver/solver
// configuration.
//
// 4. providing a tool to run commands on instances in suite and output the format
//
// 5. providing a tool to query and report/compare runtimes.
package bench
