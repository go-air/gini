// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/irifrance/gini/inter"
	"github.com/irifrance/gini/z"
)

// RandS creates an inter.S which just returns  result to Solve() within a
// random period of time chosen from [0..d).  The result may be specified
// ahead of time, however if the result is 0, then a random value from
// {-1,1} is chosen.
//
// GoSolve() works as usual.
//
// other inter.S methods are just stubs with default
// values.
//
// This is useful for testing applications using inter.S
func RandS(d time.Duration, res int) inter.S {
	return RandSr(d, res, rand.NewSource(33))
}

func RandSr(d time.Duration, res int, src rand.Source) inter.S {
	return &randS{
		cancel: make(chan struct{}),
		pause:  make(chan struct{}),
		dur:    d,
		res:    res,
		rand:   rand.New(src)}
}

type randS struct {
	gmu    sync.Mutex
	rmu    sync.Mutex
	dur    time.Duration
	res    int
	rand   *rand.Rand
	ms     []z.Lit
	mv     z.Var
	cancel chan struct{}
	pause  chan struct{}
}

func (r *randS) lock() {
	r.gmu.Lock()
	r.rmu.Lock()
}

func (r *randS) unlock() {
	r.gmu.Unlock()
	r.rmu.Unlock()
}

func (r *randS) Add(m z.Lit) {
	if m.Var() > r.mv {
		r.mv = m.Var()
	}
}

func (r *randS) Lit() z.Lit {
	m := (r.mv + 1).Pos()
	r.Add(m)
	return m
}

func (r *randS) Assume(ms ...z.Lit) {
	r.lock()
	defer r.unlock()
	r.ms = append(r.ms, ms...)
}

func (r *randS) MaxVar() z.Var {
	r.lock()
	defer r.unlock()
	return r.mv
}

func (r *randS) SCopy() inter.S {
	r.rmu.Lock()
	defer r.rmu.Unlock()
	return &randS{
		dur:    r.dur,
		rand:   rand.New(rand.NewSource(11)),
		cancel: make(chan struct{}),
		pause:  make(chan struct{})}
}

func (r *randS) Value(m z.Lit) bool {
	return r.rand.Intn(2) == 1
}

func (r *randS) Solve() int {
	r.lock()
	defer r.unlock()
	return r.solve()
}

func (r *randS) Try(dur time.Duration) int {
	r.lock()
	defer r.unlock()
	return r.solve()
}

func (r *randS) solve() int {
	r.ms = r.ms[:0]
	ns := r.dur.Nanoseconds()
	w := time.Duration(r.rand.Int63n(ns))
	alarm := time.After(w * time.Nanosecond)
	for {
		select {
		case <-alarm:
			if r.res == 0 {
				if r.rand.Intn(2) == 0 {
					return -1
				}
				return 1
			}
			return r.res
		case <-r.cancel:
			return 0
		case <-r.pause:
			r.pause <- struct{}{}
		}
	}
}

func (r *randS) GoSolve() inter.Solve {
	r.lock()

	ctl := &ctl{
		s:      r,
		resChn: make(chan int),
		cancel: r.cancel,
		pause:  r.pause}

	go func() {
		ctl.resChn <- r.solve()
	}()
	return ctl
}

type ctl struct {
	s      *randS
	resChn chan int
	cancel chan struct{}
	pause  chan struct{}
}

func (c *ctl) Wait() int {
	select {
	case r := <-c.resChn:
		c.s.unlock()
		return r
	case <-c.cancel:
		return 0
	}
}

func (c *ctl) Test() (int, bool) {
	select {
	case r := <-c.resChn:
		c.s.unlock()
		return r, true
	default:
		return 0, false
	}
}

func (c *ctl) Try(d time.Duration) int {
	a := time.After(d)
	select {
	case r := <-c.resChn:
		c.s.unlock()
		return r
	case <-a:
		return 0
	}
}

func (c *ctl) Pause() (int, bool) {
	select {
	case r := <-c.resChn:
		c.s.unlock()
		return r, false
	case c.pause <- struct{}{}:
		c.s.rmu.Unlock()
		return 0, true
	}
}

func (c *ctl) Unpause() {
	c.s.rmu.Lock()
	<-c.pause
}

func (c *ctl) Stop() int {
	defer c.s.unlock()
	select {
	case r := <-c.resChn:
		return r
	case c.cancel <- struct{}{}:
		return 0
	}
}

func (r *randS) Why(dst []z.Lit) []z.Lit {
	r.lock()
	defer r.unlock()
	dst = dst[:0]
	dst = append(dst, r.ms...)
	return dst
}

func (r *randS) Test(dst []z.Lit) (int, []z.Lit) {
	r.lock()
	defer r.unlock()
	dst = dst[:0]
	if r.rand.Intn(2) == 1 {
		return -1, dst
	}
	return 0, dst
}

func (r *randS) Untest() int {
	if r.rand.Intn(2) == 1 {
		return -1
	}
	return 0
}

func (r *randS) Reasons(dst []z.Lit, implied z.Lit) []z.Lit {
	r.lock()
	defer r.unlock()
	dst = dst[:0]
	return dst
}

func (r *randS) String() string {
	return fmt.Sprintf("*randS[%s]", r.dur)
}
