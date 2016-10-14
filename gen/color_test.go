// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen_test

import (
	"gini/gen"
	"testing"
)

func TestRandGraph(t *testing.T) {
	g := gen.RandGraph(100, 2000)
	if len(g) != 100 {
		t.Errorf("wrong number of nodes %d != %d\n", len(g), 100)
	}
	m := 0
	for _, es := range g {
		m += len(es)
	}
	if m != 4000 {
		t.Errorf("wrong number of edges: %d != 4000\n", m)
	}
}
