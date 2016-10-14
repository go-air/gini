// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"fmt"
	"github.com/irifrance/gini/gen"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestSolveTest(t *testing.T) {
	s := NewS()
	gen.HardRand3Cnf(s, 1024)
	c := s.GoSolve()
	for i := 0; i < 10; i++ {
		r, ok := c.Test()
		fmt.Printf("result: %d, %t\n", r, ok)
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.Stop()
}

func TestSolveStatsTest(t *testing.T) {
	s := NewS()
	gen.HardRand3Cnf(s, 350)
	c := s.GoSolve().(*Ctl)
	src := c.TryStats(500*time.Millisecond, 50*time.Millisecond)
	for sr := range src {
		fmt.Printf("%s\nc %d\n", sr.Stats, sr.Result)
	}
}

func TestSolveTryHard(t *testing.T) {
	s := NewS()
	gen.HardRand3Cnf(s, 1024)
	c := s.GoSolve()
	r := c.Try(10 * time.Millisecond)
	if r != 0 {
		t.Errorf("solved hard problem too fast")
	}
}

func TestSolveTryEasy(t *testing.T) {
	s := NewS()
	gen.BinCycle(s, 4096)
	c := s.GoSolve()
	r := c.Try(10 * time.Millisecond)
	if r != 1 {
		t.Errorf("couldn't solve easy problem")
	}
}

func TestSolvePauseUnpause(t *testing.T) {
	s := NewS()
	gen.HardRand3Cnf(s, 1024)
	c := s.GoSolve()
	for i := 0; i < 10; i++ {
		if res, _ := c.Pause(); res != 0 {
			t.Errorf("very unlikely")
			return
		}
		d := time.Duration(rand.Intn(100)+1) * time.Millisecond
		log.Printf("paused solver, waiting %s\n", d)
		<-time.After(d)
		c.Unpause()
		log.Printf("unpaused.\n")
	}
	c.Stop()
}
