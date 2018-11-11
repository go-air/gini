// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package logic_test

import (
	"fmt"
	"testing"

	"github.com/irifrance/gini"
	"github.com/irifrance/gini/logic"
	"github.com/irifrance/gini/z"
)

func TestUnrollComb(t *testing.T) {
	s := logic.NewS()
	i0, i1, i2 := s.Lit(), s.Lit(), s.Lit()

	a1 := s.And(i0, i1)
	a2 := s.And(a1, i2)
	a3 := s.And(a1, i1.Not())
	a4 := s.And(a2, a3)

	u := logic.NewUnroll(s)
	u.At(a4, 3)
	if u.C.Len() != 2+((u.S.Len()-2)*4) {
		t.Errorf("wrong length expected %d got %d\n", 2+((u.S.Len()-2)*3), u.C.Len())
	}
}

func TestUnrollLatch(t *testing.T) {
	s := logic.NewS()
	m := s.Latch(s.F)
	s.SetNext(m, m.Not())
	u := logic.NewUnroll(s)
	u.At(m, 3)
}

func TestUnrollCnfSince(t *testing.T) {
	s := logic.NewS()
	m := s.Latch(s.F)
	n := s.Lit()
	o := s.Or(m, n)
	s.SetNext(m, o)

	u := logic.NewUnroll(s)
	var mark []int8
	sat := gini.New()
	ttl := 0
	var a int
	for i := 0; i < 64; i++ {
		p := u.At(o, i)
		mark, a = u.C.CnfSince(sat, mark, p)
		ttl += a
	}
}

func TestUnrollCnfCounter(t *testing.T) {
	// create a 7 bit counter which increments when the 2 inputs are xor
	s := logic.NewS()
	N := 7
	in0, in1 := s.Lit(), s.Lit()
	xo := s.Xor(in0, in1)
	ms := make([]z.Lit, N)
	carry := xo
	for i := range ms {
		ms[i] = s.Latch(s.F)
		s.SetNext(ms[i], s.Choice(carry, ms[i].Not(), ms[i]))
		carry = s.And(carry, ms[i])
	}
	// set up unrolling and sat
	end := 1<<uint(N) - 1
	unroller := logic.NewUnroll(s)
	var mark []int8
	sat := gini.New()
	// for all but 'end', 'carry' should be false.
	for i := 0; i < end; i++ {
		p := unroller.At(carry, i)
		mark, _ = unroller.C.CnfSince(sat, mark, p)
		sat.Assume(p)
		if sat.Solve() != -1 {
			t.Errorf("sat at %d not unsat\n", i)
		}
	}
	p := unroller.At(carry, end)
	mark, _ = unroller.C.CnfSince(sat, mark, p)
	sat.Assume(p)
	if sat.Solve() != 1 {
		t.Errorf("unsat at %d not sat\n", end)
	}
}

func ExampleUnroll() {
	// create a new sequential circuit, a 16 bit counter
	s := logic.NewS()
	N := 16
	c := s.T
	for i := 0; i < N; i++ {
		n := s.Latch(s.F)
		s.SetNext(n, s.Choice(c, n.Not(), n))
		c = s.And(c, n)
	}

	// create an unroller.
	u := logic.NewUnroll(s)
	// unroll until all 1s
	D := (1 << uint(N)) - 1
	errs := 0
	for i := 0; i < N; i++ {
		m := s.Latches[i]
		u.At(m, D)
		if u.At(m, D) != s.T {
			errs++
		}
	}
	fmt.Printf("%d errs\n", errs)
	//Output: 0 errs
}
