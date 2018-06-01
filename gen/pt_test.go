// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen

import (
	"log"
	"testing"

	"github.com/irifrance/gini"
)

func TestPart(t *testing.T) {
	elts, trips := pytriples(1000)
	for _, p := range trips {
		if p.a*p.a+p.b*p.b != p.c*p.c {
			t.Errorf("bad triple: %d %d %d\n", p.a, p.b, p.c)
		}
	}
	for i, e := range elts {
		_, _ = i, e
		//log.Printf("%d %d\n", i, e)
	}
}

func TestPy2Triples(t *testing.T) {
	g := gini.New()
	N := 7824
	N = 7000
	Py2Triples(g, N)
	log.Printf("%d\n", g.Solve())
}
