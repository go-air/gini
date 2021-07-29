// Copyright 2018 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package aiger

import (
	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"
)

type sDfs struct {
	marks []byte
	s     *logic.S
	fn    func(s *logic.S, m z.Lit)
}

func newsDfs(s *logic.S, f func(s *logic.S, m z.Lit)) *sDfs {
	ms := make([]byte, s.Len())
	return &sDfs{marks: ms, s: s, fn: f}
}

func (d *sDfs) reset() {
	for i := range d.marks {
		d.marks[i] = 0
	}
}

func (d *sDfs) post(ms ...z.Lit) {
	for _, m := range ms {
		d.vis(m)
	}
}

func (d *sDfs) vis(m z.Lit) {
	if d.marks[m.Var()] == 2 {
		return
	}
	if d.marks[m.Var()] == 1 {
		panic("loop")
	}
	d.marks[m.Var()] = 1
	if d.s.Type(m) == logic.SAnd {
		c0, c1 := d.s.Ins(m)
		d.vis(c0)
		d.vis(c1)
	}
	d.fn(d.s, m)
	d.marks[m.Var()] = 2
}
