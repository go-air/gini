// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package main

import (
	"flag"
	"fmt"
	"github.com/irifrance/gini/crisp"
	"log"
	"os"
	"path/filepath"
)

var CrispD *crisp.Server
var trace = flag.Bool("trace", false, "turn on protocol tracing")

func main() {
	flag.Usage = func() {
		p := os.Args[0]
		_, p = filepath.Split(p)
		fmt.Fprintf(os.Stderr, usage, p, p)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	s, e := crisp.NewServer(flag.Arg(0))
	if e != nil {
		log.Printf("error starting CRISP-1.0 server: %s\n", e)
		return
	}
	CrispD = s
	CrispD.Trace(*trace)
	log.Println(CrispD.Serve())
}
