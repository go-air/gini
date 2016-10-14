// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package main

import (
	"flag"
	"fmt"
	"gini/bench"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var selFlags = flag.NewFlagSet("sel", flag.ExitOnError)

type selOptsT struct {
	N       *int
	Link    *bool
	Name    *string
	Pattern *string
}

func (sel *selOptsT) Run(flags *flag.FlagSet) {
	insts := bench.MatchSelect(*sel.Pattern, *sel.N, flags.Args()...)
	var suite *bench.Suite
	var e error
	if *sel.Link {
		suite, e = bench.LinkSuite(*sel.Name, insts)
	} else {
		suite, e = bench.CreateSuite(*sel.Name, insts)
	}
	_ = suite
	if e != nil {
		fmt.Fprintf(os.Stderr, "error creating suite: %s\n", e)
	} else {
		fmt.Printf("created suite '%s'\n", *sel.Name)
	}
}

var selOpts = &selOptsT{
	N:       selFlags.Int("n", 100, "number of files to select"),
	Link:    selFlags.Bool("link", false, "symlink files to new names instead of copying"),
	Name:    selFlags.String("name", "bench", "put benchmark in this directory"),
	Pattern: selFlags.String("pattern", "", "match this pattern when selecting files.")}

var runFlags = flag.NewFlagSet("run", flag.ExitOnError)

type runOptsT struct {
	Name   *string
	Del    *string
	Cmd    *string
	Dur    *time.Duration
	GDur   *time.Duration
	Desc   *string
	Commit *string
}

var runOpts = &runOptsT{
	Cmd:    runFlags.String("cmd", "", "command to run on each instance"),
	Dur:    runFlags.Duration("dur", 5*time.Second, "max per-instance duration"),
	GDur:   runFlags.Duration("gdur", 1*time.Hour, "max run duration"),
	Desc:   runFlags.String("desc", "", "description of run."),
	Commit: runFlags.String("commit", "", "commit id of command"),
	Name:   runFlags.String("name", "run", "name of the run"),
	Del:    runFlags.String("d", "", "delete the run")}

func (r *runOptsT) Run(flags *flag.FlagSet) {
	if *r.Del != "" {
		r.delRuns(flags)
		return
	}
	if *r.Cmd == "" {
		fmt.Fprintf(os.Stderr, "must specify a command (-cmd).\n")
		os.Exit(1)
	}
	if *r.Name == "" {
		fmt.Fprintf(os.Stderr, "must specify a run name (-name).\n")
		os.Exit(1)
	}
	for i := 0; i < flags.NArg(); i++ {
		arg := flags.Arg(i)
		suite, e := bench.OpenSuite(arg)
		if e != nil {
			fmt.Fprintf(os.Stderr, "error opening suite %s: %s\n", arg, e)
			continue
		}
		run, e := suite.Run(*r.Name, *r.Cmd, *r.Dur, *r.GDur)
		if e != nil {
			fmt.Fprintf(os.Stderr, "error initializing run %s for suite %s: %s\n", *r.Name, arg, e)
			continue
		}
		for j := 0; j < run.Len(); j++ {
			start := time.Now()
			log.Printf("starting instance %d of %s\n", j, arg)
			irun, e := run.Do(j)
			if e != nil {
				fmt.Fprintf(os.Stderr, "error executing %s for inst %d of suite %s: %s\n", *r.Cmd, j, arg, e)
				continue
			}
			dur := time.Since(start)
			log.Printf("done instance %d of %s in %s: %d\n", j, arg, dur, irun.Result)
		}
	}
}

func (r *runOptsT) delRuns(flags *flag.FlagSet) {
	for i := 0; i < flags.NArg(); i++ {
		arg := flags.Arg(i)
		suite, e := bench.OpenSuite(arg)
		if e != nil {
			fmt.Fprintf(os.Stderr, "cannot open suite %s: %s\n", arg, e)
			continue
		}
		if e := suite.RemoveRun(*r.Del); e != nil {
			fmt.Fprintf(os.Stderr, "error removing run %s from suite %s: %s\n", *r.Del, arg, e)
		}
	}
}

var cmpFlags = flag.NewFlagSet("cmp", flag.ExitOnError)

type cmpOptsT struct {
	Listing *bool
	Sat     *bool
	Unsat   *bool
	Summary *bool
	Cactus  *bool
	Scatter *bool
	Runs    *string
}

var cmpOpts = &cmpOptsT{
	Listing: cmpFlags.Bool("list", false, "list all instances in all runs."),
	Sat:     cmpFlags.Bool("sat", false, "list only sat instances in runs."),
	Unsat:   cmpFlags.Bool("unsat", false, "list only unsat instances in runs."),

	Summary: cmpFlags.Bool("sum", true, "suite summary."),
	Cactus:  cmpFlags.Bool("cactus", false, "cactus plot"),
	Scatter: cmpFlags.Bool("scatter", false, "scatter plot of run pairs"),
	Runs:    cmpFlags.String("runs", "*", "comma separated list of run globs")}

func (co *cmpOptsT) runFilt() func(*bench.Run) bool {
	parts := strings.Split(*co.Runs, ",")
	return func(r *bench.Run) bool {
		for _, m := range parts {
			m, b := filepath.Match(m, r.Name)
			if b != nil {
				log.Printf("warning: match '%s' gave an error on '%s'", m, r.Name)
				continue
			}
			if !m {
				continue
			}
			return true
		}
		return false
	}
}

func satFilt(s *bench.Suite, i int) bool {
	for _, r := range s.Runs {
		ir := r.InstRuns[i]
		if ir.Result == 1 {
			return true
		}
	}
	return false
}

func unsatFilt(s *bench.Suite, i int) bool {
	for _, r := range s.Runs {
		ir := r.InstRuns[i]
		if ir.Result == -1 {
			return true
		}
	}
	return false
}

func (c *cmpOptsT) Run(flags *flag.FlagSet) {
	if *c.Sat && *c.Unsat {
		log.Printf("cannot specify both -sat and -unsat")
		return
	}
	var iFilt func(s *bench.Suite, i int) bool
	if *c.Sat {
		iFilt = satFilt
	} else if *c.Unsat {
		iFilt = unsatFilt
	}

	for i := 0; i < flags.NArg(); i++ {
		arg := flags.Arg(i)
		suite, err := bench.OpenSuite(arg)
		if err != nil {
			log.Printf("error opening suite %s: %s\n", arg, err)
			continue
		}
		if *c.Summary {
			fmt.Println(bench.Summary(suite))
		}
		if *c.Listing {
			fmt.Println(bench.Listing(suite, c.runFilt()))
		}
		if *c.Scatter {
			rFilt := c.runFilt()
			for i, ra := range suite.Runs {
				if !rFilt(ra) {
					continue
				}
				for j := 0; j < i; j++ {
					rb := suite.Runs[j]
					if !rFilt(rb) {
						continue
					}
					sc := bench.NewScatter(ra, rb, iFilt)
					fmt.Printf("%s: %s v %s\n%s\n", suite.Root, ra.Name, rb.Name, sc.Utf8(40))
				}
			}
		}
		if *c.Cactus {
			rf := c.runFilt()
			cactus := bench.NewCactus(suite, rf, iFilt)
			fmt.Printf("\n%s\n", cactus.Utf8(40))
		}
	}
}

func main() {
	log.SetPrefix(" [bench] ")
	selFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "sel [seloptions] dir [ dir [ dir ... ] ]\n")
		fmt.Fprintf(os.Stderr, "\tsel selects files and puts them in bench suite format.\n")
		selFlags.PrintDefaults()
	}
	runFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "run [runoptions] suite [ suite [ suite ... ] ]\n")
		fmt.Fprintf(os.Stderr, "\trun runs commands on benchmark suites enforcing timeouts and\n")
		fmt.Fprintf(os.Stderr, "\trecording results.\n")
		runFlags.PrintDefaults()
	}
	cmpFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "cmp [cmp options] suite\n")
		cmpFlags.PrintDefaults()
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "bench <cmd> [options] arg arg ...\n")
		fmt.Fprintf(os.Stderr, "<cmd> may be\n\tsel\n\trun\n\tcmp\n")
		fmt.Fprintf(os.Stderr, "For help with a command, run bench <cmd> -h.\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "sel":
		selFlags.Parse(os.Args[2:])
		selOpts.Run(selFlags)
	case "run":
		runFlags.Parse(os.Args[2:])
		runOpts.Run(runFlags)
	case "cmp":
		cmpFlags.Parse(os.Args[2:])
		cmpOpts.Run(cmpFlags)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
