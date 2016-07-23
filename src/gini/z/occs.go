// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package z

// Type Occs provides a minimal occurance list for a CNF
// under unit propagation.
type Occs struct {
	D     [][]int
	C     [][]Lit
	U     []int
	UFunc func(m Lit)
	head  int
	top   int
	free  []int
}

func NewOccs() *Occs {
	o := &Occs{}
	o.C = append(o.C, nil)
	return o
}

func (o *Occs) Add(m Lit) int {
	if m == LitNull {
		res := o.top
		o.top++
		o.C = append(o.C, nil)
		return res
	}
	s := int(m)
	for s >= len(o.D) {
		o.D = append(o.D, nil)
	}
	o.D[m] = append(o.D[m], o.top)
	o.C[o.top] = append(o.C[o.top], m)
	return -1
}

func (o *Occs) Set(m Lit) {
}

func (o *Occs) Remove(m Lit, c int) {
}
