// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gini

import (
	"gini/gen"
	"gini/z"
	"testing"
	"time"
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
	sDur := time.Now().Sub(b4Solve)
	margin := 2 * timeout

	if r != 0 {
		t.Errorf("solve hard php wasn't cancelled in .005 seconds")
	}
	if sDur < timeout-margin {
		t.Errorf("cancelled early %s < %s", sDur, timeout)
	}
	if sDur > timeout+margin {
		t.Errorf("cancelled late. %s > %s", sDur, timeout)
	}
}
