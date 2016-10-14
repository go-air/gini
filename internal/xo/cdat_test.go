// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"fmt"
	"github.com/irifrance/gini/z"
	"testing"
)

var cnf = [][]z.Lit{
	{z.Lit(3), z.Lit(5), z.Lit(6), z.Lit(24)},
	{},
	{},
	{},
	{z.Lit(104), z.Lit(97), z.Lit(17), z.Lit(19), z.Lit(3), z.Lit(9), z.Lit(10), z.Lit(12), z.Lit(14), z.Lit(20), z.Lit(22), z.Lit(24), z.Lit(26),
		z.Lit(28), z.Lit(30), z.Lit(32), z.Lit(34), z.Lit(36), z.Lit(38), z.Lit(40), z.Lit(42), z.Lit(44), z.Lit(46), z.Lit(48), z.Lit(50), z.Lit(52), z.Lit(54),
		z.Lit(56), z.Lit(58), z.Lit(60), z.Lit(62), z.Lit(64), z.Lit(66), z.Lit(68), z.Lit(70)},
	{z.Lit(33)},
	{}}

var hds = []Chd{
	MakeChd(false, 0, 4),
	MakeChd(true, 0, 0),
	MakeChd(false, 0, 0),
	MakeChd(false, 4, 0),
	MakeChd(true, 0, 35), // size too big for size modulus in header, positioned to require iteration afterwards.
	MakeChd(true, 0, 1),
	MakeChd(true, 0, 0)}

// for compaction testing: remove clauses at indices in rmi, leave behind clauses with indices in left
var rmi = [...]int{0, 2, 3, 5}
var left = [...]int{1, 4, 6}

func TestCDat(t *testing.T) {
	ldb := NewCDat(8)
	locs := make([]CLoc, 0, 10)
	for i, cls := range cnf {
		locs = append(locs, ldb.AddLits(hds[i], cls))
	}
	ms := make([]z.Lit, 0, 10)
	for i, p := range locs {
		ms = ms[:0]
		ms = ldb.Load(p, ms)
		if len(ms) != len(cnf[i]) {
			t.Errorf("bad load or put: %s != %s", ms, cnf[i])
		}
		for j, m := range ms {
			n := cnf[i][j]
			if m != n {
				t.Errorf("mismatched clause %d[%d]: %s != %s", i, j, m, n)
			}
		}
	}
	for i, j := 0, 1; j < len(locs); i++ {
		if locs[i] == locs[j] {
			t.Errorf("adjacent locs: %d, %d", i, j)
		}
		j++
	}

	// test compact
	rm := make([]CLoc, 4, 4)
	for i, j := range rmi {
		rm[i] = locs[j]
	}

	//fmt.Printf("before compact:\n")
	//fmt.Print(ldb)

	//fmt.Printf("\ncompacting...\n")
	m, _ := ldb.Compact(rm)
	//fmt.Printf("\nafter compact:\n")
	//fmt.Print(ldb)
	//fmt.Printf("relocation map: %s\n", m)

	for _, i := range left {
		p, ok := m[locs[i]]
		if !ok {
			t.Errorf("missing location")
		}
		if p == CLocNull {
			t.Errorf("left clause indicated as removed in map")
		}
		ms = ms[:0]
		ms = ldb.Load(p, ms)

		if len(ms) != len(cnf[i]) {
			t.Errorf("bad load or put: %s != %s", ms, cnf[i])
		}
		for j, m := range ms {
			n := cnf[i][j]
			if m != n {
				t.Errorf("mismatched clause %d[%d]: %s != %s", i, j, m, n)
			}
		}
		hd := ldb.Chd(p)

		if hd != hds[i] {
			t.Errorf("mismatched head after compact: %s != %s", hd, hds[i])
		}
	}
	// for coverage, not really value-tested...
	_ = fmt.Sprintf("%s", ldb)
}
