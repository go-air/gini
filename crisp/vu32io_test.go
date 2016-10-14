// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestVu32Io(t *testing.T) {
	N := 1024
	rw := bytes.NewBuffer(nil)
	io := newVu32Io(rw)
	ds := make([]uint32, 0, 8)
	for i := 0; i < N; i++ {
		m := rand.Intn(8) + 1
		ds := ds[:0]
		for j := 0; j < m; j++ {
			u := rand.Uint32()
			if e := io.writeu32(u); e != nil {
				t.Error(e)
				return
			}
			ds = append(ds, u)
		}
		if e := io.flush(); e != nil {
			t.Error(e)
			return
		}
		for j := 0; j < m; j++ {
			v, e := io.readu32()
			if e != nil {
				t.Error(e)
				return
			}
			if ds[j] != v {
				t.Errorf("write/read %d/%d\n", ds[j], v)
			}
		}
	}
}
