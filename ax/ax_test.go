// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package ax

import (
	"fmt"
	"github.com/irifrance/g"
	"github.com/irifrance/g/gen"
	"github.com/irifrance/g/z"
	"log"
	"math/rand"
	"testing"
	"time"
)

func ExampleNewT() {
	// create a random duration, random result solver.
	// with 1024 max var.
	d := 15 * time.Millisecond
	randS := gen.RandSr(d, 0, rand.NewSource(44))
	randS.Add(z.Var(1024).Pos())

	// start the ax with randS as the prototype
	ax := NewT(randS, 4)
	defer ax.Stop()

	// code for generating 128 random queries queries
	N := 128
	cubes := gen.NewRandCuber(32, z.Var(1024))

	Limit(0)
	var nGen int = 0
	var genReq = func() *Request {
		if nGen < N {
			nGen++
			return NewRequest(cubes.RandCube(nil)...)
		}
		return nil
	}

	// start the request/response loop.
	var req *Request = genReq()
	ttl := 0
	ttlSat := 0

	for ttl < N {
		resp := ax.Ex(req)
		if resp == nil { // request submitted.
			req = genReq()
			continue
		}
		// request not submitted, but we have a response.
		ttl++
		if resp.Res == 1 {
			ttlSat++
		}
	}
	fmt.Printf("solved %d\n", ttlSat)
	// Output: solved 67
}

func TestLimit(t *testing.T) {
	g := gini.New()
	gen.Rand3Cnf(g, 1000, 4000)

	ax := NewT(g, 24)
	defer ax.Stop()
	Limit(time.Millisecond)
	var req *Request = NewRequest()
	N := 100
	ttl := 0
	ttlr := 0

	start := time.Now()
	for resp := ax.Ex(req); ttl < N; resp = ax.Ex(req) {
		if resp == nil {
			if ttlr < N {
				v := z.Var(rand.Intn(1000) + 1).Pos()
				req = NewRequest(v)
				ttlr++
			} else {
				req = nil
			}
			continue
		}
		ttl++
	}
	dur := time.Since(start)
	if dur > 3*time.Second {
		t.Errorf("took too long: %s >> %s\n", dur, time.Second)
	}
}

func TestModel(t *testing.T) {
	g := gini.New()
	gen.Php(g, 10, 10)

	ax := NewT(g, 2)
	defer ax.Stop()
	Limit(0)
	SetFlag(ReqModel)
	var req *Request = NewRequest()

	for {
		resp := ax.Ex(req)
		if resp == nil {
			req = nil
			continue
		}
		if len(resp.Ms) != 100 {
			t.Errorf("wrong length %d expected %d\n", len(resp.Ms), 100)
		}
		return
	}
}

func TestWhy(t *testing.T) {
	for i := 0; i < 10; i++ {
		g := gini.New()
		gen.Rand3Cnf(g, 50, 180)

		ax := NewT(g, 2)
		defer ax.Stop()
		Limit(0)
		SetFlag(ReqWhy)
		var req *Request = NewRequest(gen.NewRandCuber(10, z.Var(50)).RandCube(nil)...)
		ax.Ex(req)
		resp := ax.Ex(nil)
		if resp.Res != -1 {
			log.Printf("unlikely result: %d\n", resp.Res)
		}
		log.Printf("why: %+v\n", resp.Ms)
		if len(resp.Ms) > 0 {
			return
		}
	}
	t.Errorf("highy unlikely")
}
