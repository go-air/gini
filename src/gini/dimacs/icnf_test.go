// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package dimacs

import (
	"bytes"
	"fmt"
	"gini/z"
	"testing"
)

var iCnf = `p inccnf
55 3 0
11 
0
44 13 0 21
0
a 5 0
a 3 2
1 0
33 2 0
`

type iCnfLine struct {
	t  *testing.T
	A  bool
	Ms []z.Lit
}

func (f *iCnfLine) Add(m z.Lit) {
	if f.A {
		f.t.Errorf("Add when assuming %s", m)
	}
	f.do(m)
}

func (f *iCnfLine) Assume(m z.Lit) {
	f.A = true
	f.do(m)
}

func (f *iCnfLine) Eof() {
}

func (f *iCnfLine) do(m z.Lit) {
	if m == z.LitNull {
		if f.A {
			fmt.Printf("a %+v\n", f.Ms)
		} else {
			fmt.Printf("c %+v\n", f.Ms)
		}
		f.A = false
		f.Ms = f.Ms[:0]
		return
	}
	f.Ms = append(f.Ms, m)
}

func TestICnf(t *testing.T) {
	r := bytes.NewBufferString(iCnf)
	if e := ReadICnf(r, &iCnfLine{t: t}); e != nil {
		t.Errorf("error icnf: %s\n", e)
	}
}
