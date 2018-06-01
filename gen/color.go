// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen

import (
	"math/rand"

	"github.com/irifrance/gini/inter"
	"github.com/irifrance/gini/z"
)

// RandColor creates a formula asking if a random
// (simple) graph with n nodes and m edges can
// be colored with k colors.  Every node must have
// a color and no 2 adjacent nodes may have the same color.
func RandColor(dst inter.Adder, n, m, k int) {
	g := RandGraph(n, m)
	var mkVar = func(n, c int) z.Var {
		return z.Var(n*c + c + 1)
	}
	for i := range g {
		for j := 0; j < k; j++ {
			m := mkVar(i, j).Pos()
			dst.Add(m)
		}
		dst.Add(0)
	}
	for a, es := range g {
		for _, b := range es {
			if b >= a {
				continue
			}
			for c := 0; c < k; c++ {
				a := mkVar(a, c).Neg()
				b := mkVar(b, c).Neg()
				dst.Add(a)
				dst.Add(b)
			}
			dst.Add(0)
		}
	}
}

type edge struct {
	a, b int
}

// RandGraph creates a simple (undirected) random graph with n nodes and m
// edges.  If m > n*(n-1)/2, RandGraph returns nil.
//
// The result is in the form of an edge list, namely each node is idenitified
// by an integer in [0..n) and the edgelist for node i is result[i], which is a
// list of edges.  There are no multi-edges, no self edges, and sampling is
// done without replacement.  The number m is is the number of edges (a, b)
// such that  a < b, the symmetric view of the graph is returned with 2*m edges
// symmetrified.
func RandGraph(n, m int) [][]int {
	if m > n*(n-1)/2 {
		return nil
	}
	ns := make([][]int, n)

	es := make([]edge, 0, n*(n-1)/2)
	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			es = append(es, edge{i, j})
		}
	}

	for i := 0; i < m; i++ {
		el := len(es)
		j := rand.Intn(el)
		e := es[j]
		ns[e.a] = append(ns[e.a], e.b)
		el--
		es[j], es[el] = es[el], es[j]
		es = es[:el]
	}
	// make it symmetric
	for i, es := range ns {
		for _, j := range es {
			ns[j] = append(ns[j], i)
		}
	}
	return ns
}
