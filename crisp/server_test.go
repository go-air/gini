// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"fmt"
	"github.com/irifrance/g/gen"
	"github.com/irifrance/g/internal/xo"
	"github.com/irifrance/g/z"
	"log"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

func init() {
	os.Remove("crisp0")
	go func() {
		if e := ListenAndServe("@crisp0"); e != nil {
			log.Fatalf("error setting up server: %s\n", e)
			return
		}
		log.Printf("server running.\n")
	}()
	// TBD: let the tests know when server is running
	time.Sleep(50 * time.Millisecond)
}

func TestServer(t *testing.T) {

	for j := 0; j < 8; j++ {
		log.Printf("dialing\n")
		c, e := Dial("@crisp0")
		if e != nil {
			t.Error(e)
			return
		}
		log.Printf("got a client\n")
		defer func() { c.Close() }()

		for i := 1; i < 10; i++ {
			c.Add(z.Var(i).Pos())
			c.Add(0)
		}

		r, e := c.Solve()
		if e != nil {
			t.Error(e)
			return
		}
		if r != 1 {
			t.Errorf("bad sat result: %d\n", r)
		}

		if e := c.Quit(); e != nil {
			t.Error(e)
			return
		}
		c.Close()
	}
}

func TestServerAssume(t *testing.T) {
	c, e := Dial("@crisp0")
	if e != nil {
		t.Error(e)
		return
	}

	for i := 0; i < 10; i++ {
		v := z.Var(i + 1)
		m := v.Neg()
		w := z.Var(i + 2)
		n := w.Pos()
		for _, m := range []z.Lit{m, n, z.Lit(0)} {
			if e := c.Add(m); e != nil {
				t.Error(e)
				return
			}
		}
	}

	m := z.Var(3).Pos()
	n := z.Var(6).Pos()

	// assume consistent
	if e := c.Assume([]z.Lit{m.Not(), n.Not()}...); e != nil {
		t.Error(e)
		return
	}
	r, e := c.Solve()
	if e != nil {
		t.Error(e)
		return
	}
	if r != 1 {
		t.Errorf("expected sat got %d\n", r)
		return
	}
	model, e := c.ModelFor(nil, []z.Lit{m, n})
	if e != nil {
		t.Error(e)
		return
	}
	if len(model) < 3 {
		t.Errorf("model length: %d < 3", len(model))
		return
	}
	if model[0] || model[1] {
		t.Errorf("bad model %+v", model)
		return
	}
	// assume inconsistent (and call assume 2x)
	if e := c.Assume(m); e != nil {
		t.Error(e)
		return
	}
	if e := c.Assume(n.Not()); e != nil {
		t.Error(e)
		return
	}
	r, e = c.Solve()
	if e != nil {
		t.Error(e)
		return
	}
	if r != -1 {
		t.Errorf("expected unsat got %d\n", r)
		return
	}
	failed, e := c.Why(nil)
	if e != nil {
		t.Error(e)
		return
	}
	if len(failed) != 2 {
		t.Errorf("expected 2 failed got %d", len(failed))
		return
	}
	if failed[0] != m && failed[0] != n.Not() {
		t.Errorf("bad failed lit: %s\n", failed[0])
	}
	if failed[1] != m && failed[1] != n.Not() {
		t.Errorf("bad failed lit: %s\n", failed[1])
	}
	if e := c.Quit(); e != nil {
		t.Errorf("quit failed in test assume: %s\n", e)
	}
	c.Close()
}

func php(t *testing.T, p, h int) error {
	log.Printf("php %d/%d\n", p, h)
	c, e := Dial("@crisp0")
	if e != nil {
		return e
	}
	//log.Printf("connected.\n")
	xo := xo.NewS()
	gen.Php(xo, p, h)
	if e := c.fromXo(xo); e != nil {
		return fmt.Errorf("php(%d,%d) fromXo: %s\n", p, h, e)
	}
	r, e := c.Solve()
	if e != nil {
		return fmt.Errorf("php(%d,%d) solve: %s\n", p, h, e)
	}
	if h < p && r == 1 {
		return fmt.Errorf("expected unsat(-1) got %d\n", r)
	}
	if h >= p && r == -1 {
		return fmt.Errorf("expected sat(1) got %d\n", r)
	}
	if e := c.Quit(); e != nil {
		return fmt.Errorf("error quit: %s\n", e)
	}
	c.Close()
	return nil
}

func TestParaPhp(t *testing.T) {
	var wg sync.WaitGroup
	for p := 7; p < 9; p++ {
		for h := p - 1; h < p+2; h++ {
			wg.Add(1)
			go func(p, h int) {
				if e := php(t, p, h); e != nil {
					t.Error(e)
				}
				wg.Done()
			}(p, h)
		}
	}
	wg.Wait()
}

func TestPhp(t *testing.T) {
	for p := 7; p < 9; p++ {
		for h := p - 1; h < p+2; h++ {
			if e := php(t, p, h); e != nil {
				t.Error(e)
			}
		}
	}
}

func TestFailedFor(t *testing.T) {
	c, e := Dial("@crisp0")
	if e != nil {
		t.Errorf("dial: %s\n", e)
		return
	}
	for i := 0; i < 10; i++ {
		xos := xo.NewS()
		gen.Rand3Cnf(xos, 100, 300)
		if e := c.fromXo(xos); e != nil {
			t.Errorf("fromXo: %s\n", e)
			return
		}
		r, e := c.Solve()
		if e != nil {
			t.Errorf("solve: %s\n", e)
			return
		}
		if r != 1 {
			continue
		}
		// exercise model
		_, e = c.Model(nil)
		if e != nil {
			t.Errorf("model: %s\n", e)
		}
		// find inconsistent assumptions
		for j := 0; j < 3; j++ {
			ms := make([]z.Lit, 0, 100)
			var fs []z.Lit
			var m z.Lit
			for v := z.Var(1); v <= z.Var(100); v++ {
				if rand.Intn(2) == 0 {
					m = v.Pos()
				} else {
					m = v.Neg()
				}
				ms = append(ms, m)
				if len(ms) < 8 {
					continue
				}
				if e := c.Assume(ms...); e != nil {
					t.Errorf("assume: %s\n", e)
					return
				}
				r, e := c.Solve()
				if e != nil {
					t.Errorf("solve: %s\n", e)
					return
				}
				if r != -1 {
					continue
				}
				log.Printf("found bad set of assumptions %+v\n", ms)
				fs = ms[:6]
				sfs, e := c.WhySelect(nil, fs)
				if e != nil {
					t.Errorf("failed for: %s\n", e)
					return
				}
				log.Printf("bad subset of interest: %+v\n", fs)
				log.Printf("sub subset %+v\n", sfs)
				// check order of sfs consistent with order of fs
				j := 0
				for _, f := range fs {
					if j >= len(sfs) {
						break
					}
					if f == sfs[j] {
						j++
					}
				}
				if j != len(sfs) {
					t.Errorf("%+v not order consistent with %+v\n", sfs, fs)
				}
				c.Quit()
				c.Close()
				return
			}
		}
	}
	// 10 random 100/300 3cnfs not sat, highly unlikely
	t.Errorf("highly unlikely: too many unsat 100/300 3cnfs\n")
}

func TestReset(t *testing.T) {
	c, e := Dial("@crisp0")
	if e != nil {
		t.Error(e)
		return
	}
	log.Printf("got a client\n")
	defer func() { c.Quit(); c.Close() }()
	if e := c.Add(z.Lit(11)); e != nil {
		t.Error(e)
		return
	}
	if e := c.Add(z.Lit(0)); e != nil {
		t.Error(e)
		return
	}
	if e := c.Reset(); e != nil {
		t.Error(e)
		return
	}
	if e := c.Add(z.Lit(11).Not()); e != nil {
		t.Error(e)
		return
	}
	if e := c.Add(z.Lit(0)); e != nil {
		t.Error(e)
		return
	}
	n, e := c.Solve()
	if e != nil {
		t.Error(e)
		return
	}
	if n != 1 {
		t.Errorf("reset apparently gave unsat due to previous clauses")
	}
}

func TestExt(t *testing.T) {
	c, e := Dial("@crisp0")
	if e != nil {
		t.Error(e)
		return
	}
	log.Printf("got a client\n")
	defer func() { c.Quit(); c.Close() }()
	_, e = c.GetExts()
	if e != nil {
		t.Error(e)
	}
}
