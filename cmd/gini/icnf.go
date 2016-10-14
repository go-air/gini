// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package main

import (
	"flag"
	"fmt"
	"github.com/irifrance/g"
	"github.com/irifrance/g/ax"
	"github.com/irifrance/g/dimacs"
	"github.com/irifrance/g/z"
	"io"
	"log"
	"runtime"
	"time"
)

var useAx = flag.Bool("ax", false, "run the assumption exchanger for icnf inputs")

func icnfCap() int {
	n := runtime.NumCPU()
	if n > 2 {
		return n - 1
	}
	return n
}

func runICnfReader(r io.Reader) (int, error) {
	var vis dimacs.ICnfVis
	if *useAx {

		vis = newICnfAx(*timeout, icnfCap())
	} else {
		vis = newICnfGini(*timeout)
	}
	if e := dimacs.ReadICnf(r, vis); e != nil {
		log.Printf("error reading icnf: %s\n", e)
		return 0, e
	}
	return 0, nil
}

type iCnfGini struct {
	start time.Time
	end   time.Time
	g     *gini.Gini
}

func newICnfGini(timeout time.Duration) *iCnfGini {
	now := time.Now()
	return &iCnfGini{
		start: now,
		end:   now.Add(timeout),
		g:     gini.New()}
}

func (i *iCnfGini) Add(m z.Lit) {
	i.g.Add(m)
}

func (i *iCnfGini) Assume(m z.Lit) {
	if m != z.LitNull {
		i.g.Assume(m)
		return
	}
	now := time.Now()
	if i.end.Before(now) {
		fmt.Printf("s UNKNOWN\n")
		return
	}
	remaining := i.end.Sub(now)
	switch i.g.GoSolve().Try(remaining) {
	case 1:
		fmt.Printf("s SATISFIABLE\n")
	case -1:
		fmt.Printf("s UNSATISFIABLE\n")
	case 0:
		fmt.Printf("s UNKNOWN\n")
	default:
		panic("bad solve return")
	}
}

func (i *iCnfGini) Eof() {
}

type iCnfAx struct {
	start    time.Time
	end      time.Time
	g        *gini.Gini
	ax       ax.T
	cubes    [][]z.Lit
	cube     []z.Lit
	reqIdx   int
	capacity int
	solved   int
}

func newICnfAx(timeout time.Duration, cap int) *iCnfAx {
	a := &iCnfAx{
		start: time.Now(),
		g:     gini.New(),
		cubes: make([][]z.Lit, 0, 1024)}
	a.end = a.start.Add(timeout)
	a.capacity = cap
	return a
}

func (i *iCnfAx) Add(m z.Lit) {
	i.solveCubes()
	i.g.Add(m)
}

func (i *iCnfAx) Assume(m z.Lit) {
	if m != z.LitNull {
		i.cube = append(i.cube, m)
		return
	}
	i.cubes = append(i.cubes, i.cube)
	i.cube = nil
}

func (i *iCnfAx) Eof() {
	i.solveCubes()
}

func (i *iCnfAx) solveCubes() {
	if len(i.cubes) == 0 {
		return
	}
	log.Printf("solving %d cubes\n", len(i.cubes))
	i.ax = ax.NewT(i.g, i.capacity)
	defer i.ax.Stop()
	ttl := len(i.cubes)
	req := i.genReq()
	results := make([]int, len(i.cubes))
	for ttl > 0 {
		resp := i.ax.Ex(req)
		if resp == nil {
			req = i.genReq()
			continue
		}
		results[resp.Req.Id] = resp.Res
		if resp.Res != 0 {
			i.solved++
		}
		ttl--
	}
	i.cubes = i.cubes[:0]
	i.reqIdx = 0
	for _, res := range results {
		switch res {
		case 1:
			fmt.Printf("s SATISFIABLE\n")
		case -1:
			fmt.Printf("s UNSATISFIABLE\n")
		case 0:
			fmt.Printf("s UNKNOWN\n")
		default:
			panic("bad ax response: result")
		}
	}
	fmt.Printf("c solved %d/%d\n", i.solved, len(results))
}

func (i *iCnfAx) genReq() *ax.Request {
	if i.reqIdx >= len(i.cubes) {
		return nil
	}
	req := ax.NewRequest(i.cubes[i.reqIdx]...)
	req.Id = i.reqIdx
	req.Limit = i.end.Sub(time.Now())
	i.reqIdx++
	return req
}
