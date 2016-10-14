// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package logic

import (
	"github.com/irifrance/g/inter"
	"github.com/irifrance/g/z"
)

// Type C represents a formula or combinational circuit.
type C struct {
	nodes  []node   // list of all nodes
	strash []uint32 // strash
	F      z.Lit    // false literal
	T      z.Lit
}

type node struct {
	a z.Lit  // input a
	b z.Lit  // input b
	n uint32 // next strash
}

// NewC create a new circuit.
func NewC() *C {
	phi := &C{}
	initC(phi, 128)
	return phi
}

// NewCCap creates a new combinational circuit with initial capacity capHint.
func NewCCap(capHint int) *C {
	phi := &C{}
	initC(phi, capHint)
	return phi
}

func initC(c *C, capHint int) {
	c.nodes = make([]node, 2, capHint)
	c.strash = make([]uint32, capHint)
	c.F = z.Var(1).Neg()
	c.T = c.F.Not()
}

// ToCnf creates a conjunctive normal form of p in
// adder.
//
// Adder uses basic Tseitinization.
func (p *C) ToCnf(dst inter.Adder) {
	e := len(p.nodes)
	for i := 1; i < e; i++ {
		n := p.nodes[i]
		a := n.a
		if a == z.LitNull || a == p.F || a == p.T {
			continue
		}
		b := n.b
		g := z.Var(i).Pos()
		addAnd(dst, g, a, b)
	}
}

func addAnd(dst inter.Adder, g, a, b z.Lit) {
	dst.Add(g.Not())
	dst.Add(a)
	dst.Add(0)
	dst.Add(g.Not())
	dst.Add(b)
	dst.Add(0)
	dst.Add(g)
	dst.Add(a.Not())
	dst.Add(b.Not())
	dst.Add(0)
}

// ToCnfFrom creates a conjunctive normal form of p in
// adder, including only the part of the circuit reachable
// from some root in roots.
func (p *C) ToCnfFrom(dst inter.Adder, roots ...z.Lit) {
	dfs := make([]int8, len(p.nodes))
	var vis func(m z.Lit)
	vis = func(m z.Lit) {
		v := m.Var()
		if dfs[v] == 1 {
			return
		}
		n := &p.nodes[v]
		if n.a == z.LitNull || n.a == p.T || n.a == p.F {
			dfs[v] = 1
			return
		}
		vis(n.a)
		vis(n.b)
		g := m
		if !m.IsPos() {
			g = m.Not()
		}
		addAnd(dst, g, n.a, n.b)
		dfs[v] = 1
	}
	for _, root := range roots {
		vis(root)
	}
}

// Len returns the length of C, the number of
// internal nodes used to represent C.
func (c *C) Len() int {
	return len(c.nodes)
}

// At returns the i'th element.  Elements from
// 0..Len(c) are in topological order:  if i < j
// then c.At(j) is not reachable from c.At(i)
// via the edge relation defined by c.Ins().
// All elements are positive literals.
//
// One variable for internal use, with index 1, is created when
// c is created.  All other variables created by NewIn, And, ...
// are created in sequence starting with index 2.  Internal
// variables may be created by c.  c.Len() - 1 is the maximal
// index of a variable.
//
// Hence, the implementation of At(i) is simply z.Var(i).Pos().
func (c *C) At(i int) z.Lit {
	return z.Var(i).Pos()
}

// NewIn returns a new variable/input to p.
func (p *C) NewIn() z.Lit {
	m := len(p.nodes)
	p.newNode()
	return z.Var(m).Pos()
}

// InPos returns the positions of all inputs
// in c in the sequence attainable via Len() and
// At().  The result is placed in dst if there is space.
//
// If c is part of S, then latches are not included.
func (c *C) InPos(dst []int) []int {
	dst = dst[:0]
	for i, n := range c.nodes {
		if i == 0 {
			continue
		}
		if n.a == z.LitNull && n.b == z.LitNull {
			dst = append(dst, i)
		}
	}
	return dst
}

// Eval evaluates the circuit with values vs, where
// for each literal m in the circuit, vs[i] contains
// the value for m's variable if m.Var() == i.
//
// vs should contain values for all inputs.
func (c *C) Eval(vs []bool) {
	for i := range c.nodes {
		n := &c.nodes[i]
		if n.a == z.LitNull {
			continue
		}
		a, b := n.a, n.b
		va, vb := vs[a.Var()], vs[b.Var()]
		if !a.IsPos() {
			va = !va
		}
		if !b.IsPos() {
			vb = !vb
		}
		g := z.Var(i)
		vs[g] = va && vb
	}
}

// Eval64 is like Eval but evaluates 64 different inputs in
// parallel as the bits of a uint64.
func (c *C) Eval64(vs []uint64) {
	for i := range c.nodes {
		n := &c.nodes[i]
		if n.a == z.LitNull {
			continue
		}
		a, b := n.a, n.b
		va, vb := vs[a.Var()], vs[b.Var()]
		if !a.IsPos() {
			va = ^va
		}
		if !b.IsPos() {
			vb = ^vb
		}
		g := z.Var(i)
		vs[g] = va & vb
	}
}

// And returns a literal equivalent to "a and b", which may
// be a new variable.
func (p *C) And(a, b z.Lit) z.Lit {
	if a == b {
		return a
	}
	if a == b.Not() {
		return p.F
	}
	if a > b {
		a, b = b, a
	}
	if a == p.F {
		return p.F
	}
	if a == p.T {
		return b
	}
	c := strashCode(a, b)
	l := uint32(cap(p.nodes))
	i := c % l
	si := p.strash[i]
	for {
		n := &p.nodes[si]
		if n.a == a && n.b == b {
			return z.Var(si).Pos()
		}
		if n.n == 0 {
			break
		}
		si = n.n
	}
	m, j := p.newNode()
	m.a = a
	m.b = b
	k := c % uint32(cap(p.nodes))
	m.n = p.strash[k]
	p.strash[k] = j
	return z.Var(j).Pos()
}

// Ands constructs a conjunction of a sequence of literals.
// If ms is empty, then Ands returns p.T.
func (p *C) Ands(ms ...z.Lit) z.Lit {
	a := p.T
	for _, m := range ms {
		a = p.And(a, m)
	}
	return a
}

// Or constructs a literal which is the disjunction of a and b.
func (p *C) Or(a, b z.Lit) z.Lit {
	nor := p.And(a.Not(), b.Not())
	return nor.Not()
}

// Ors constructs a literal which is the disjuntion of the literals in ms.
// If ms is empty, then Ors returns p.F
func (p *C) Ors(ms ...z.Lit) z.Lit {
	d := p.F
	for _, m := range ms {
		d = p.Or(d, m)
	}
	return d
}

// Implies constructs a literal which is equivalent to (a implies b).
func (p *C) Implies(a, b z.Lit) z.Lit {
	return p.Or(a.Not(), b)
}

// Xor constructs a literal which is equivalent to (a xor b).
func (p *C) Xor(a, b z.Lit) z.Lit {
	return p.Or(p.And(a, b.Not()), p.And(a.Not(), b))
}

// Choice constructs a literal which is equivalent to
//  if i then t else e
func (p *C) Choice(i, t, e z.Lit) z.Lit {
	return p.Or(p.And(i, t), p.And(i.Not(), e))
}

// Ins returns the children/ operands of m.
//
//  If m is an input, then, Ins returns z.LitNull, z.LitNull
//  If m is an and, then Ins returns the two conjuncts
func (p *C) Ins(m z.Lit) (z.Lit, z.Lit) {
	v := m.Var()
	n := p.nodes[v]
	return n.a, n.b
}

func (p *C) newNode() (*node, uint32) {
	if len(p.nodes) == cap(p.nodes) {
		p.grow()
	}
	id := len(p.nodes)
	p.nodes = p.nodes[:id+1]
	return &p.nodes[id], uint32(id)
}

func (p *C) grow() {
	newCap := cap(p.nodes) * 2
	nodes := make([]node, cap(p.nodes), newCap)
	strash := make([]uint32, newCap)
	copy(nodes, p.nodes)
	ucap := uint32(newCap)
	for i := range nodes {
		n := &nodes[i]
		if n.a == 0 || n.a == p.F || n.a == p.T {
			continue
		}
		c := strashCode(n.a, n.b)
		j := c % ucap
		n.n = strash[j]
		strash[j] = uint32(i)
	}
	p.nodes = nodes
	p.strash = strash
}

func strashCode(a, b z.Lit) uint32 {
	return uint32((a << 13) * b)
}
