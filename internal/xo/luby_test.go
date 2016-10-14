// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import "testing"

func TestLuby(t *testing.T) {
	luby := NewLuby()
	times := make(map[uint]uint)
	for i := 0; i < 127; i++ {
		n := luby.Next()
		v, ok := times[n]
		if !ok {
			times[n] = n
		} else {
			times[n] = n + v
		}
	}
	timePerStrat := uint(64)
	for k, v := range times {
		if v != timePerStrat {
			t.Errorf("wrong total strategy time for %d: %d != %d", k, v, timePerStrat)
		}
	}
}
