// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package z

import "testing"

func TestLitDimacs(t *testing.T) {
	for i := 1; i < 100; i++ {
		if Dimacs2Lit(i).Dimacs() != i {
			t.Errorf("dimacs conversion %d", i)
		}
		if Dimacs2Lit(-i).Dimacs() != -i {
			t.Errorf("dimacs - conversion %d", i)
		}
		if !Dimacs2Lit(i).IsPos() {
			t.Errorf("not positive: %d", i)
		}
		if Dimacs2Lit(-i).IsPos() {
			t.Errorf("not negative: -%d", i)
		}
	}
}
