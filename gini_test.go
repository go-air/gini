// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gini

import (
	"math/rand"
	"testing"
	"time"

	"github.com/irifrance/gini/gen"
	"github.com/irifrance/gini/z"
)

func TestGiniTrivUnsat(t *testing.T) {
	g := New()
	g.Add(z.Lit(3))
	g.Add(0)
	g.Add(z.Lit(3).Not())
	g.Add(0)
	if g.Solve() != -1 {
		t.Errorf("basic add unsat failed.")
	}
}

func TestGiniAsync(t *testing.T) {
	// hard problem
	g := New()
	gen.Php(g, 15, 14)
	c := g.GoSolve()
	ticker := time.Tick(5 * time.Millisecond)
	for ticks := 0; ticks < 100; ticks++ {
		select {
		case <-ticker:
			r, b := c.Test()
			if r != 0 && !b {
				t.Errorf("returned solved and not finished")
				return
			}
			if b || r != 0 {
				t.Errorf("returned too soon to be believable")
				return
			}
		}
	}

	timeout := 50 * time.Millisecond
	b4Solve := time.Now()
	r := c.Try(timeout)
	sDur := time.Since(b4Solve)
	margin := 2 * timeout

	if r != 0 {
		t.Errorf("solve hard php wasn't cancelled in .005 seconds")
	}
	if sDur < timeout-margin {
		t.Errorf("cancelled early %s < %s", sDur, timeout)
	}
	if sDur > timeout+margin {
		// CI builders don't like this and have unreasonable values.
		t.Logf("cancelled late. %s > %s", sDur, timeout)
	}
}

type dummySimp struct {
	toRm   []z.C
	toKeep []z.C
	tst    *testing.T
}

func (d *dummySimp) OnAdded(c z.C, ms []z.Lit) {
	for _, m := range ms {
		if m.Var() == 1024 {
			d.toRm = append(d.toRm, c)
			return
		}
	}
	d.toKeep = append(d.toKeep, c)
}

func (d *dummySimp) CRemap(cm map[z.C]z.C) {
	for i, c := range d.toKeep {
		n, ok := cm[c]
		if !ok {
			continue
		}
		if n == 0 {
			d.tst.Errorf("got CNull on remap")
			continue
		}
		d.toKeep[i] = n
	}
}

func (d *dummySimp) Simplify(rmSpace []z.C) (status int, rms []z.C) {
	rms = rmSpace
	rms = append(rms, d.toRm...)
	d.toRm = d.toRm[:0]
	status = 0
	return
}

func TestCnfSimp(t *testing.T) {
	// we do pure lit on 1024, and create a formula with >2048 clauses
	// containing 1024 together with a 64 variable random 3cnf near the
	// cutoff of sat/unsat equal probability
	for n := 0; n < 32; n++ {
		s0, s1 := New(), New()
		s0.SetCnfSimp(&dummySimp{tst: t})
		pure := z.Var(1024).Pos()
		with := z.Var(1025).Pos()
		for i := 0; i < 2049; i++ { // 2049 because 2048 is the limit to cause lit removal.
			// not unit so they will all add
			s0.Add(pure)
			s0.Add(with)
			s0.Add(0)
			s1.Add(pure)
			s1.Add(with)
			s1.Add(0)
		}
		for i := 0; i < 270; i++ {
			va, vb, vc := z.Var(rand.Intn(64)+1), z.Var(rand.Intn(64)+1), z.Var(rand.Intn(64)+1)
			ma := va.Pos()
			if rand.Intn(2) == 1 {
				ma = ma.Not()
			}
			mb := vb.Pos()
			if rand.Intn(2) == 1 {
				mb = mb.Not()
			}
			mc := vc.Pos()
			if rand.Intn(2) == 1 {
				mc = mc.Not()
			}
			s0.Add(ma)
			s0.Add(mb)
			s0.Add(mc)
			s0.Add(0)

			s1.Add(ma)
			s1.Add(mb)
			s1.Add(mc)
			s1.Add(0)
		}
		if n := s0.Simplify(); n != 0 {
			t.Errorf("simplify gave %d\n", n)
		}
		a, b := s0.Solve(), s1.Solve()
		if a != b {
			t.Errorf("something went wrong %d,%d\n", a, b)
		}
		t.Logf("%d ?= %d\n", a, b)
	}
}
