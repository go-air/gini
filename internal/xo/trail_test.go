// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"github.com/irifrance/g/gen"
	"github.com/irifrance/g/z"
	"math/rand"
	"testing"
)

func TestTrailBack(t *testing.T) {
	N := 1024
	dst := NewS()
	gen.Rand3Cnf(dst, N, N+50)
	trail := NewTrail(dst.Cdb, newGuess(N))

	for i := 0; i < N/2; i++ {
		for j := 0; j < i; j++ {
			m := z.Var(j + 1).Pos()
			trail.Assign(m, CLocNull)
			x := trail.Prop()
			if x != CLocNull {
				trail.Back(0)
				return
			}
		}
		for j := i; j >= 0; j-- {
			if j%7 != 0 {
				continue
			}
			trail.Back(j)
			if trail.Level > i {
				t.Errorf("level %d != %d", trail.Level, i)
			}
		}
	}
}
func TestTrailBinarySat(t *testing.T) {
	N := 8
	dst := NewS()
	gen.BinCycle(dst, N)
	cdb := dst.Cdb
	trail := NewTrail(cdb, newGuess(N))
	trail.Assign(z.Lit(2), CLocNull)
	x := trail.Prop()
	if x != CLocNull {
		t.Errorf("binary cycle: unexpected conflict")
	}
	if trail.Tail != N {
		t.Errorf("binary cycle: tail %d != %d", trail.Tail, N)
	}
}

func TestTrailBinaryUnsat(t *testing.T) {
	N := 8
	dst := NewS()
	gen.BinCycle(dst, N)
	cdb := dst.Cdb
	cdb.Add(z.Lit(4))
	p, _ := cdb.Add(z.LitNull)
	trail := NewTrail(cdb, newGuess(N))
	trail.Assign(z.Lit(4), p)
	trail.Assign(z.Lit(7), CLocNull)
	x := trail.Prop()
	if x == CLocNull {
		t.Errorf("binary cycle: expected conflict")
	}
}

func TestTrailTernary(t *testing.T) {
	N := 128
	dst := NewS()
	gen.Rand3Cnf(dst, N, N*4)
	cdb := dst.Cdb
	trail := NewTrail(cdb, newGuess(N))
	x := CLocNull
	vals := cdb.Vars.Vals
	for x == CLocNull && trail.Tail != N {
		m := z.Lit(2)
		for vals[m] != 0 {
			m = z.Lit(rand.Intn(N*2) + 2)
		}
		trail.Assign(m, CLocNull)
		x = trail.Prop()
		if x != CLocNull {
			trail.Back(trail.Level - 1)
		}
		errs := cdb.CheckWatches()
		for _, e := range errs {
			t.Errorf("%s", e)
		}
		if len(errs) > 0 {
			return
		}
	}
}
