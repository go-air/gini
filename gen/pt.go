// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen

import (
	"github.com/irifrance/gini/z"
	"sort"
)

// PartVar returns a variable if element i is in partition k
// for a set of n elements.
func PartVar(i, k, n int) z.Lit {
	return z.Var(k*n + i + 1).Pos()
}

// Partition adds constraints to dst stating that there exists a partition of n
// elements into k parts.  Every model of the result is a partition with
// PartVar(i, k, n) true if and only if element i is in partition k.
func Partition(dst Dest, n, k int) {
	for i := 0; i < n; i++ {
		for j := 0; j < k; j++ {
			dst.Add(PartVar(i, j, n))
		}
		dst.Add(0)
	}
	for i := 0; i < n; i++ {
		for j := 0; j < k; j++ {
			for h := 0; h < j; h++ {
				dst.Add(PartVar(i, j, n).Not())
				dst.Add(PartVar(i, h, n).Not())
				dst.Add(0)
			}
		}
	}
}

// PyTriples adds constraints stating there is no triple (i,j,k)
// s.t i^2 + j^2 = k^2 in some partition of a k-partition of [1..n]
func PyTriples(dst Dest, n, k int) {
	Partition(dst, n, k)
	_, ts := pytriples(n)
	for _, t := range ts {
		for p := 0; p < k; p++ {
			a := PartVar(t.a, p, n)
			b := PartVar(t.b, p, n)
			c := PartVar(t.c, p, n)
			dst.Add(a.Not())
			dst.Add(b.Not())
			dst.Add(c.Not())
			dst.Add(0)
		}
	}
}

// Py2Triples adds constraints to dst stating that there is a
// 2-partition of [1..n] such that no triple (a,b,c) appears
// in one partition with a^2 + b^2 = c^2.
func Py2Triples(dst Dest, n int) {
	_, ts := pytriples(n)
	for _, t := range ts {
		a, b, c := z.Var(t.a).Pos(), z.Var(t.b).Pos(), z.Var(t.c).Pos()
		dst.Add(a)
		dst.Add(b)
		dst.Add(c)
		dst.Add(0)
		dst.Add(a.Not())
		dst.Add(b.Not())
		dst.Add(c.Not())
		dst.Add(0)
	}
	// by symmetry, we can assign 1 variable
	//dst.Add(z.Var(1).Pos())
	//dst.Add(0)
}

type squares struct {
	d []int
}

func (s *squares) get(i int) int {
	t := s.d
	for len(t) <= i {
		t = append(t, len(t)*len(t))
	}
	s.d = t
	return t[i]
}

func (s *squares) root(v int) int {
	t := s.d
	for len(t)*len(t) < v {
		t = append(t, len(t)*len(t))
	}
	s.d = t
	if t[len(t)-1] == v {
		return len(t) - 1
	}
	i := sort.Search(len(t), func(i int) bool { return t[i] >= v })
	if i < len(t) && t[i] == v {
		return i
	}
	return -1
}

type triple struct {
	a, b, c int
}

func pytriples(n int) (map[int]int, []triple) {
	ai, bi := 1, 2
	res := make([]triple, 0, n)
	sqrs := &squares{make([]int, 0, n)}
	in := make(map[int]int, n)
	for len(res) < n {
		a2, b2 := sqrs.get(ai), sqrs.get(bi)
		c2 := a2 + b2
		ci := sqrs.root(c2)
		if ci != -1 {
			in[ai] = 0
			in[bi] = 0
			in[ci] = 0
			res = append(res, triple{ai, bi, ci})
		}
		ai++
		if ai == bi {
			ai = 1
			bi++
		}
	}
	ins := make([]int, 0, len(in))
	for k := range in {
		ins = append(ins, k)
	}
	sort.Ints(ins)
	for i, s := range ins {
		in[s] = i
	}
	return in, res
}

func counts(ts []triple) []int {
	res := make([]int, 0, len(ts)+len(ts)/2)
	for _, t := range ts {
		for _, v := range []int{t.a, t.b, t.c} {
			for len(res) <= v {
				res = append(res, 0)
			}
			res[v]++
		}
	}
	return res
}

// eliminates all triples which contain a variable
// which occurs in only 1 triple, iteratively until
// fixed point.  any model for the resulting formula
// can be extended to the original by picking a value
// for the single-triple variable which is not in
// the partition of atleast one of the other variables.
//
// for some reason, this slows down gini, so we don't use
// it.
func ptElim(ts []triple) []triple {
	counts := counts(ts)
	for {
		j := 0
		for _, t := range ts {
			if counts[t.a] == 1 || counts[t.b] == 1 || counts[t.c] == 1 {
				counts[t.a]--
				counts[t.b]--
				counts[t.c]--
				continue
			}
			ts[j] = t
			j++
		}
		if j == len(ts) {
			return ts
		}
		ts = ts[:j]
	}
}
