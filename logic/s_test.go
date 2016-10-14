// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package logic_test

import (
	"github.com/irifrance/gini/logic"
	"testing"
)

func TestS(t *testing.T) {
	s := logic.NewS()
	toggle := s.NewIn()
	r := s.Latch(s.F)
	c := s.Choice(toggle, r, r.Not())
	s.SetNext(r, c)

	if s.Next(r) != c {
		t.Errorf("next not expected: expected %s got %s", c, s.Next(r))
	}
	if s.Init(r) != s.F {
		t.Errorf("init: expected %s got %s\n", s.F, s.Init(r))
	}
}
