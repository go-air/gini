// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package logic_test

import (
	"fmt"
	"gini/logic"
	"testing"
)

func TestUnrollComb(t *testing.T) {
	s := logic.NewS()
	i0, i1, i2 := s.NewIn(), s.NewIn(), s.NewIn()

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
		if u.At(m, D) != s.C.T {
			errs++
		}
	}
	fmt.Printf("%d errs\n", errs)
	//Output: 0 errs
}
