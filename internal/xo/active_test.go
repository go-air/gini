package xo

import (
	"math/rand"
	"testing"

	"github.com/irifrance/gini/z"
)

func TestActive3(t *testing.T) {
	N := 32
	M := N * 3
	M -= M / 7
	s0 := NewS()
	for i := 0; i < M; i++ {
		m, n, o := randMs(N)
		s0.Add(m)
		s0.Add(n)
		s0.Add(o)
		s0.Add(0)
	}
	if s0.Solve() != 1 {
		t.Logf("unlikely unsat rand 3-CNF, starting over")
		TestActive3(t)
		return
	}
	s1 := s0.Copy()
	NCs := N*5 - M
	Cs := make([]z.Lit, 0, NCs*3)
	acts0 := make([]z.Lit, NCs)
	acts1 := make([]z.Lit, NCs)
	for i := 0; i < NCs; i++ {
		m, n, o := randMs(N)

		s0.Add(m)
		s0.Add(n)
		s0.Add(o)
		acts0[i] = s0.Activate()

		act1 := s1.Lit()
		acts1[i] = act1
		s1.Add(m)
		s1.Add(n)
		s1.Add(o)
		s1.Add(act1.Not())
		s1.Add(0)
		Cs = append(Cs, m, n, o)
	}
	// compare assumptions with activations/de-activations
	active := make([]bool, len(acts0))
	for i := 0; i < 16384; i++ {
		j := rand.Intn(len(acts0))
		if active[j] {
			act := acts0[j]
			s0.Deactivate(act)
			s0.Cdb.gc.CompactCDat(s0.Cdb)
			s0.Cdb.Forall(func(c z.C, h Chd, ms []z.Lit) {
				for _, m := range ms {
					if m.Var().Pos() == act {
						t.Errorf("%s in %s%v\n", act, c, ms)
						break
					}
				}
			})
		} else {
			k := 3 * j
			s0.Add(Cs[k])
			s0.Add(Cs[k+1])
			s0.Add(Cs[k+2])
			acts0[j] = s0.Activate()
		}
		active[j] = !active[j]
		for j := range acts0 {
			if active[j] {
				s0.Assume(acts0[j])
				s1.Assume(acts1[j])
			}
		}
		r0, r1 := s0.Solve(), s1.Solve()
		if r0 != r1 {
			s := s0
			if r0 != -1 {
				s = s1
			}
			t.Fatalf("%d,%d %v\n", r0, r1, s.Why(nil))
		}
		//t.Logf("%05d ok %d <%s,%s> %t->%t (%d,%d)", i, j, acts0[j], acts1[j], !active[j], active[j], r0, r1)
	}
}

func randMs(N int) (m, n, o z.Lit) {
	w, x, y := z.Var(rand.Intn(N)), z.Var(rand.Intn(N)), z.Var(rand.Intn(N))
	w++
	x++
	y++
	for w == x || w == y || x == y {
		if x == y {
			x = z.Var(rand.Intn(N))
		} else {
			w = z.Var(rand.Intn(N))
		}
	}
	m, n, o = w.Pos(), x.Pos(), y.Pos()
	if rand.Intn(2) == 1 {
		m = m.Not()
	}
	if rand.Intn(2) == 1 {
		n = n.Not()
	}
	if rand.Intn(2) == 1 {
		o = o.Not()
	}
	return
}
