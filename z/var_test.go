// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package z

import (
	"fmt"
	"testing"
)

func TestVar(t *testing.T) {
	v := Var(33)
	m := v.Pos()
	n := v.Neg()
	if m.Sign() != 1 {
		t.Errorf("wrong sign for pos lit %d", m.Sign())
	}
	if n.Sign() != -1 {
		t.Errorf("wrong sign for neg lit %d", m.Sign())
	}
	if m.Not() != n {
		t.Errorf("lit pos/neg not negations")
	}
	if m.Var() != v || n.Var() != v {
		t.Errorf("generated lits not same var")
	}
	if fmt.Sprintf("%s", v) != fmt.Sprintf("v%d", uint32(v)) {
		t.Errorf("format.")
	}
}
