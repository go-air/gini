package logic

import (
	"github.com/irifrance/gini/inter"
	"github.com/irifrance/gini/z"
)

// Card provides an interface for different implementations
// of cardinality constraints.
type Card interface {
	Leq() z.Lit
	Geq() z.Lit
	Less() z.Lit
	Gr() z.Lit
	N() int
}

// CardSort provides cardinality constraints via sorting networks.
//
// Sorting Networks
//
// CardSort uses sorting networks which implement O log(|ms|)**2 compare/swap
// to sort |ms| literals. Each compare/swap is coded symbolically and generates
// 6 clauses with 2 new variables.  The resulting network helps the solver
// achieve arc consistency w.r.t. the variables in ms and the output
// cardinality constraints.  Namely, any partial valuation of ms will cause the
// solver to deduce the corresponding valid and unsat card constraints by unit
// propagation.
//
// While not a best fit coding mechanism for all cases, sorting networks are a
// good choice for a general use single mechanism for coding cardinality
// constraints and hence solving Boolean optimisation problems.
//
// The idea was originally presented by Nicolas Sorensson and Nicolas Een in
// "Translating Pseudo-Boolean Constraints into SAT" Journal on Satisfiability,
// Boolean Modelng, and Computation.
type CardSort struct {
	n   int
	va  LitAdder
	ms  []z.Lit
	tmp []z.Lit
	one z.Lit
}

// VarAdder gives an interface to something which can generate
// fresh variables and add constraints.
type LitAdder interface {
	inter.Adder
	inter.Liter
}

// NewCardSort creates a new Card object which gives access to unary Cardinality
// constraints over ms.  The resulting predicates reflect how many of the literals
// in ms are true.
//
func NewCardSort(ms []z.Lit, va LitAdder) *CardSort {
	p := uint(0)
	for 1<<p < len(ms) {
		p++
	}
	ns := make([]z.Lit, 1<<p)
	copy(ns, ms)
	c := &CardSort{ms: ns, va: va, n: len(ms)}
	c.one = va.Lit()
	va.Add(c.one)
	va.Add(z.LitNull)
	for i := len(ms); i < len(ns); i++ {
		ns[i] = c.one
	}
	c.sort(0, len(ns))
	return c
}

func (c *CardSort) Valid() z.Lit {
	return c.one
}

// Less returns a literal which is true iff and only if the number of true
// literals over the set to be counted does not exceed b
func (c *CardSort) Less(b int) z.Lit {
	return c.Leq(b - 1)
}

func (c *CardSort) Leq(b int) z.Lit {
	if b >= c.n {
		return c.one
	}
	if b < 0 {
		return c.one.Not()
	}
	return c.ms[(c.n-1)-b].Not()
}

func (c *CardSort) Geq(b int) z.Lit {
	if b <= 0 {
		return c.one
	}
	if b >= c.n+1 {
		return c.one.Not()
	}
	return c.Leq(b - 1).Not()
}

func (c *CardSort) Gr(b int) z.Lit {
	return c.Geq(b + 1)
}

// N returns the number of literals whose
// cardinality is tested.  N is len(ms) when
// the caller calls
//
//    NewCard(ms, va)
func (c *CardSort) N() int {
	return c.n
}

func (n *CardSort) sort(l, h int) {
	if h-l <= 1 {
		return
	}
	//fmt.Printf("sort [%d..%d)\n", l, h)
	m := l + (h-l)/2
	n.sort(l, m)
	n.sort(m, h)
	n.merge(l, h, 1)
}

//
// odd even merge sort
//
func (n *CardSort) merge(l, h, s int) {
	if h <= l+s {
		return
	}
	//fmt.Printf("merge [%d..%d) by %d\n", l, h, s)
	var ml, mh z.Lit
	ss := 2 * s
	if ss >= h-l {
		ml, mh = n.lh(l, l+s)
		n.ms[l], n.ms[l+s] = ml, mh
		return
	}
	n.merge(l, h, ss)
	n.merge(l+s, h, ss)
	lim := h - s
	for i := l + s; i < lim; i += ss {
		ml, mh = n.lh(i, i+s)
		n.ms[i], n.ms[i+s] = ml, mh
	}
}

// compare-and-swap (low-high)
func (n *CardSort) lh(i, j int) (z.Lit, z.Lit) {
	mi, mj := n.ms[i], n.ms[j]
	a, b := n.va.Lit(), n.va.Lit()
	n.add(mi, mj, a)
	n.add(mi.Not(), mj.Not(), b.Not())
	return a, b
}

func (n *CardSort) add(mi, mj, c z.Lit) {
	// if i is 0 c is 0
	n.va.Add(mi)
	n.va.Add(c.Not())
	n.va.Add(z.LitNull)
	// if j is 0 c is 0
	n.va.Add(mj)
	n.va.Add(c.Not())
	n.va.Add(z.LitNull)

	// if i and j are both 1 c is 1
	n.va.Add(mi.Not())
	n.va.Add(mj.Not())
	n.va.Add(c)
	n.va.Add(z.LitNull)
}
