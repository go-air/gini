// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"bytes"
	"github.com/irifrance/gini/z"
	"os"
	"testing"
)

var cnfDat = [...][]z.Lit{
	{z.Lit(32), z.Lit(11), z.Lit(77)},
	{z.Lit(55), z.Lit(861), z.Lit(860), z.Lit(2)},
	{z.Lit(118), z.Lit(121)},
	{z.Lit(118)}}

var isBins = []bool{
	false,
	false,
	true,
	false}

var isUnits = []bool{
	false,
	false,
	false,
	true}

var learnts = [...][]z.Lit{
	{z.Lit(10), z.Lit(12)},
	{z.Lit(60), z.Lit(77), z.Lit(126)}}

func TestCdbAdd(t *testing.T) {
	vars := NewVars(512)
	cdb := NewCdb(vars, 512)
	locs := make([]CLoc, 0, 12)
	for _, c := range cnfDat {
		for _, m := range c {
			cdb.Add(m)
		}
		p, u := cdb.Add(z.LitNull)
		if cdb.IsUnit(p) && u == z.LitNull {
			t.Errorf("didn't return unit")
		}
		locs = append(locs, p)
	}
	for i, p := range locs {
		if p == CLocNull || p == CLocInf {
			continue
		}
		pIsBin := cdb.IsBinary(p)
		if pIsBin != isBins[i] {
			t.Errorf("isBinary for clause %s", p)
		}
		pIsUnit := cdb.IsUnit(p)
		if pIsUnit != isUnits[i] {
			t.Errorf("isUnit for clause %s", p)
		}
		hd := cdb.Chd(p)
		if hd.Learnt() {
			t.Errorf("learnt for added %s", p)
		}
		if int(hd.Size()) != len(cnfDat[i])&31 {
			t.Errorf("wrong size modulus %s", p)
		}
	}
	for _, e := range cdb.CheckWatches() {
		t.Errorf("%s", e)
	}
}

func TestCdbLearn(t *testing.T) {
	vars := NewVars(512)
	cdb := NewCdb(vars, 512)
	locs := make([]CLoc, 0, 12)
	for i, c := range learnts {
		locs = append(locs, cdb.Learn(c, i))
	}
	for i, p := range locs {
		if cdb.Chd(p).Lbd() != uint32(i) {
			t.Errorf("didn't record lbd")
		}
	}
}

func TestCdbLearnNil(t *testing.T) {
	vars := NewVars(10)
	cdb := NewCdb(vars, 20)
	cdb.Learn(nil, 0)
	if cdb.Bot == CLocNull {
		t.Errorf("cdb.Bot not set\n")
	}
}

func TestCdbWrite(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	vars := NewVars(512)
	cdb := NewCdb(vars, 512)
	for _, c := range cnfDat {
		for _, m := range c {
			cdb.Add(m)
			if vars.Max < m.Var() {
				vars.Max = m.Var()
			}
		}
		cdb.Add(z.LitNull)

	}
	cdb.Write(os.Stdout)
	_ = buf
}

func TestCdbBumpDecay(t *testing.T) {
	vars := NewVars(16)
	cdb := NewCdb(vars, 10)
	cdb.Add(z.Lit(3))
	p, _ := cdb.Add(z.LitNull)
	i := 0
	cdat := cdb.CDat
	for {
		i++
		if cdat.Bump(p) {
			if i != (1<<heatBits)-1 {
				t.Errorf("bump caused decay on %d\n", i)
			}
			break
		}
	}
	cdat.Bump(p) // wraparound overflow to 0
	for i > 0 {
		cdb.Bump(p)
		i--
	}
}
