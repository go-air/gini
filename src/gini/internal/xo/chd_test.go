// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import "testing"

var hs = []Chd{
	MakeChd(false, 2, 21),
	MakeChd(true, 2, 21),
	MakeChd(false, 6, 55)}

var ls = []bool{false, true, false}
var lbds = []uint32{2, 2, 6}
var szs = []uint32{21, 21, 55}

func TestChd(t *testing.T) {

	for i, h := range hs {
		if h.Learnt() != ls[i] {
			t.Errorf("%d: %s != %s\n", i, h.Learnt(), ls[i])
		}
		if h.Lbd() != lbds[i] {
			t.Errorf("%d: %d != %d\n", i, h.Lbd(), lbds[i])
		}
		if h.Size()&31 != szs[i]&31 {
			t.Errorf("%d: %d != %d\n", i, h.Size(), szs[i])
		}
	}
}

func TestChdHeat(t *testing.T) {
	for i, h := range hs {
		b, _ := h.Bump(1)
		if b.Heat() <= h.Heat() {
			t.Errorf("bump didn't increase heat")
		}

		d := b.Decay()
		if d.Heat() >= b.Heat() {
			t.Errorf("decay didn't decrease heat")
		}
		for _, hh := range [...]Chd{b, d} {
			if hh.Learnt() != ls[i] {
				t.Errorf("bump or decay then learnt changed")
			}
			if hh.Lbd() != lbds[i] {
				t.Errorf("bump or decay then lbd changed")
			}
			if hh.Size() != szs[i]&31 {
				t.Errorf("bump or decay then size changed")
			}
		}
	}
}
