// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package dimacs

import (
	"bytes"
	"gini/z"
	"testing"
)

type dimacsTestData struct {
	D         string
	Strict    bool
	NonStrict bool
}

var cnfs = []dimacsTestData{dimacsTestData{`c this
c is
c a 
c comment
c but
c there 
c is 
c no 
c body
`, false, true}, // should be false/true?
	dimacsTestData{`c
p cng 7 7
1 0
2 0
3 0
4 0
5 0
6 0
7 0`, false, false},
	dimacsTestData{`p cnf 6 6
-1 0
-2 0
-3 0
-4 0
-5 0
-6 0
`, true, true},
	dimacsTestData{`p cnf 2 3
1 0
2 0`, false, true},
	dimacsTestData{`c hello
c world
10 11 23 44 -55 0`, false, true}}

type vis struct{}

func (v *vis) Add(m z.Lit) {
}

func (v *vis) Init(nv, nc int) {
}

func (v *vis) Eof() {
}

func TestDimacsStrict(t *testing.T) {
	var e error
	for _, d := range cnfs {
		b := bytes.NewBufferString(d.D)
		e = ReadCnfStrict(b, &vis{}, true)
		if d.Strict != (e == nil) {
			t.Errorf("strict/error mismatch %t/%t: %s", d.Strict, e == nil, e)
		}
	}
}

func TestDimacsNonStrict(t *testing.T) {
	var e error
	for _, d := range cnfs {
		b := bytes.NewBufferString(d.D)
		e = ReadCnf(b, &vis{})
		if d.NonStrict != (e == nil) {
			t.Errorf("non-strict/error mismatch %t/%t: %s", d.NonStrict, e == nil, e)
		}
	}
}
