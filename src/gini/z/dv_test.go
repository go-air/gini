// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package z

import (
	"log"
	"math/rand"
	"testing"
)

func TestVars(t *testing.T) {
	N := 128
	vs := NewVars()
	outers := make([]Lit, 0, N)
	o2i := make([]Lit, N)
	inners := make([]Lit, 0, N)
	for i := 0; i < N; i++ {
		var m Lit
		if i%2 == 0 {
			m = Var(i + 1).Pos()
		} else {
			m = Var(i + 1).Neg()
		}
		outers = append(outers, m)
	}
	p := rand.Perm(N)
	for i := 0; i < N; i++ {
		j := outers[p[i]]
		m := vs.ToInner(j)
		o2i[i] = m
		if rand.Float64() <= 0.13 {
			for k := 0; k < rand.Intn(28); k++ {
				inners = append(inners, vs.Inner())
			}
		}
		if len(inners) > 0 && rand.Float64() <= 0.2 {
			vs.Free(inners[len(inners)-1])
			inners = inners[:len(inners)-1]
		}
	}
	log.Println(vs)
	for i := 0; i < N; i++ {
		j := outers[p[i]]
		m := vs.ToInner(j)
		if o2i[i] != m {
			t.Errorf("non deterministic map %s -> {%s,%s}", j, m, inners[i])
		}
		if vs.ToOuter(m) != j {
			t.Errorf("ToOuter(ToInner(%s)) != %s", j, vs.ToOuter(m))
		}
	}
}
