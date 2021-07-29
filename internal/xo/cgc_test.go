// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/go-air/gini/z"
)

// nb var cnfDat from clauses_test.go

func TestCgc(t *testing.T) {
	vars := NewVars(1025)
	cdb := NewCdb(vars, 4096)
	for i := 0; i < 1024; i++ {
		v := z.Var(i + 1)
		var w z.Var
		if i+2 == 1025 {
			w = z.Var(1)
		} else {
			w = z.Var(i + 2)
		}
		cdb.Add(v.Neg())
		cdb.Add(w.Pos())
		cdb.Add(z.LitNull)
	}

	ms := make([]z.Lit, 3, 4)
	for i := 0; i < 16384; i++ {
		n := z.Var(rand.Intn(1024) + 1)
		m := z.Var(rand.Intn(1024) + 1)
		o := z.Var(rand.Intn(1024) + 1)
		for m == n {
			m = z.Var(rand.Intn(1024) + 1)
		}
		for m == o || n == o {
			o = z.Var(rand.Intn(1024) + 1)
		}
		ms = ms[:3]
		ms[0] = m.Pos()
		ms[1] = n.Neg()
		ms[2] = o.Pos()
		cdb.Learn(ms, i%7)
		if i%10 == 0 {
			onc := len(cdb.Learnts)
			nU, nC, nL := cdb.MaybeCompact()
			if nU > 0 {
				fmt.Printf("compacted %d:%d\n", nC, nL)
				cdb.gc.CompactCDat(cdb)
				wErrors := cdb.CheckWatches()
				for _, e := range wErrors {
					t.Errorf("watch problem after compact: %s", e)
				}
				if len(wErrors) > 0 {
					fmt.Printf("compact watch errors: %d\n", len(wErrors))
					t.Fatal("watch errors, terminating test.\n")
				}
			}
			if len(cdb.Learnts) != onc-nU {
				t.Errorf("bad number of learnts")
			}
		}
	}
}
