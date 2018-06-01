package logic

import (
	"math/rand"
	"testing"

	"github.com/irifrance/gini"
	"github.com/irifrance/gini/z"
)

func TestCardSort(t *testing.T) {
	N := 64
	Iters := 16
	sv := gini.New()
	ms := make([]z.Lit, N)
	for i := range ms {
		ms[i] = sv.Lit()
	}
	cs := NewCardSort(ms, sv)
	for i := 0; i < Iters; i++ {
		rand.Shuffle(len(ms), func(i, j int) { ms[i], ms[j] = ms[j], ms[i] })
		n := rand.Intn(N)
		npos := 0
		nneg := 0
		for j := 0; j < n; j++ {
			m := ms[j]
			if rand.Intn(2) == 1 {
				m = m.Not()
				nneg++
			} else {
				npos++
			}
			sv.Assume(m)
		}
		//t.Logf("assumed %d pos %d neg out of %d\n", npos, nneg, N)
		sv.Test(nil) // puts previous assumptions in different scope
		for i := 0; i <= npos; i++ {
			sv.Assume(cs.Geq(i))
			res := sv.Solve()
			if res != 1 {
				t.Errorf("assumed %d geq %d gave %d\n", npos, i, res)
			}
			sv.Assume(cs.Less(i))
			res = sv.Solve()
			if res != -1 {
				t.Errorf("assumed %d less %d gave %d\n", npos, i, res)
			}
		}
		for i := 0; i <= nneg; i++ {
			sv.Assume(cs.Leq(N - i))
			res := sv.Solve()
			if res != 1 {
				t.Errorf("assume %d false leq %d gave %d\n", nneg, N-i, res)
			}
			sv.Assume(cs.Gr(N - i))
			res = sv.Solve()
			if res != -1 {
				t.Errorf("assume %d false gr %d gave %d\n", nneg, N-i, res)
			}
		}
		sv.Untest()
	}
}

func TestCardNotPowerOfTwo(t *testing.T) {
	s := gini.New()
	ms := make([]z.Lit, 11)
	for i := range ms {
		ms[i] = s.Lit()
	}
	cs := NewCardSort(ms, s)
	for i := 0; i < 11; i++ {
		for j := 0; j < i; j++ {
			s.Assume(ms[j])
		}
		s.Test(nil)
		for j := 0; j < i; j++ {
			s.Assume(cs.Geq(j))
			if s.Solve() != 1 {
				t.Errorf("geq %d gave unsat after %d assumes\n", j, i)
			}
			s.Assume(cs.Less(j))
			if s.Solve() != -1 {
				t.Errorf("less %d gave sat after %d assumes\n", j, i)
			}
		}
		s.Untest()
	}
}
