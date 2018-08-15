// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen

import (
	"testing"
	"time"

	"github.com/irifrance/gini/z"
)

func TestRands(t *testing.T) {
	d := time.Millisecond
	s := RandS(d, 0)

	s.Add(z.Var(1024).Pos())
	if s.MaxVar() != z.Var(1024) {
		t.Errorf("max var")
	}

	for i := 0; i < 10; i++ {
		start := time.Now()
		s.Solve()
		d := time.Since(start)
		if d > time.Millisecond+500*time.Microsecond {
			// the CI builders can't handle this.
			t.Logf("took too long %s\n", d)
		}
	}

	s = RandS(time.Millisecond*100, 0)
	for i := 0; i < 10; i++ {
		p := s.GoSolve()
		_, ok := p.Test()
		if ok {
			t.Errorf("test immediately returned\n")
		}
		p.Stop()
	}

}
