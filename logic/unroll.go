// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package logic

import "github.com/irifrance/g/z"

// Type Unroll creates an unroller of sequential logic into
// combinational logic.
type Unroll struct {
	S    *S // the sequential circuit
	C    *C // the resulting comb circuit
	dmap [][]z.Lit
}

// NewUnroll creates a new unroller for s
func NewUnroll(s *S) *Unroll {
	u := &Unroll{
		S:    s,
		C:    NewCCap(s.Len() * 10),
		dmap: make([][]z.Lit, s.Len())}
	return u
}

// At returns the value of literal m from sequential circuit
// u.S at time/depth d as a literal in u.C
//
// If d < 0, then At panics.
func (u *Unroll) At(m z.Lit, d int) z.Lit {
	v := m.Var()
	if len(u.dmap[v]) < d {
		u.At(m, d-1)
	}
	var res, a, b z.Lit
	var n node
	if len(u.dmap[v]) > d {
		res = u.dmap[v][d]
		goto Done
	}
	n = u.S.nodes[v]
	if n.b == z.LitNull {
		// input
		res = u.C.NewIn()
		u.dmap[v] = append(u.dmap[v], res)
		goto Done
	}
	if d == 0 {
		if n.a == z.LitNull {
			// latch init X
			res = u.C.NewIn()
			u.dmap[v] = append(u.dmap[v], res)
			goto Done
		}
		if n.a == u.S.F {
			u.dmap[v] = append(u.dmap[v], u.C.F)
			res = u.C.F
			goto Done
		}
		if n.a == u.S.T {
			u.dmap[v] = append(u.dmap[v], u.C.T)
			res = u.C.T
			goto Done
		}
	}
	if n.a == u.S.F || n.a == u.S.T || n.a == z.LitNull {
		res = u.At(n.b, d-1) // next state time d - 1
		u.dmap[v] = append(u.dmap[v], res)
		goto Done
	}
	a, b = u.At(n.a, d), u.At(n.b, d)
	res = u.C.And(a, b)
	u.dmap[v] = append(u.dmap[v], res)
Done:
	if !m.IsPos() {
		return res.Not()
	}
	return res
}
