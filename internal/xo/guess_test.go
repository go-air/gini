// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"gini/z"
	"testing"
)

func TestGuess(t *testing.T) {
	N := 128
	g := newGuess(N)
	for i := 0; i < N-1; i++ {
		g.Push(z.Var(i + 1).Pos())
	}
	for i := 0; i < N-1; i++ {
		m := z.Var(i + 1).Pos()
		b := (i + 1) % 5
		for j := 0; j < b; j++ {
			g.Bump(m)
		}
	}

	mod := z.Var(4)
	for g.Len() > 0 {
		v := g.pop()
		m := v % 5
		if m == mod {
			continue
		}
		if m == mod-1 {
			mod--
			continue
		}
		t.Errorf("modulus shrank.\n")
	}
}
