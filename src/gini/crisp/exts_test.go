// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"log"
	"strings"
	"testing"
)

type e struct {
	id uint16
	n  uint8
	o  uint8
}

var d = []e{
	{uint16(24), uint8(17), uint8(0)},
	{uint16(77), uint8(3), uint8(20)},
	{uint16(99), uint8(0), uint8(20)},
	{uint16(100), uint8(5), uint8(25)}}

func TestExts(t *testing.T) {
	exts := NewExts()
	for i, extn := range d {
		j, e := exts.Install(extn.id, extn.n)
		if e != nil {
			t.Errorf("error installing: %s\n", e)
			continue
		}
		if j != i {
			t.Errorf("wrong index: %d != %d", j, i)
		}
	}
	for i := range d {
		if exts.Id(i) != d[i].id {
			t.Errorf("didn't get id: %d\n", exts.Id(i))
		}
		ps := exts.Op(i, 0).String()
		if !strings.Contains(ps, "ext") {
			t.Errorf("op not extensions: %s", exts.Op(i, 0))
		}
	}
	log.Printf("tested %s\n", exts)
}
