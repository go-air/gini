// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"github.com/irifrance/gini/z"
	"testing"
)

var littleLits = [...]z.Lit{
	z.Lit(3),
	z.Lit(33),
	z.Lit(7),
	z.Lit(8)}

var biggerLits = [...]z.Lit{
	z.Lit(127),
	z.Lit(151),
	z.Lit(99)}

func TestVars(t *testing.T) {
	N := 44
	vars := NewVars(N)
	for _, m := range littleLits {
		vars.Set(m)
	}
	for _, m := range littleLits {
		if vars.Sign(m) != 1 {
			t.Errorf("vals from vars.")
		}
	}

	vars.growToVar(88)
	for _, m := range biggerLits {
		vars.Set(m)
	}
	for _, m := range littleLits {
		if vars.Sign(m) != 1 {
			t.Errorf("vals from vars after grow")
		}
	}
}
