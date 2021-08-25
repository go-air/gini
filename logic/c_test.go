// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package logic

import (
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/go-air/gini"
	"github.com/go-air/gini/gen"
	"github.com/go-air/gini/z"
)

func TestCGrowStrash(t *testing.T) {
	c := NewC()
	N := 1020
	ins := make([]z.Lit, 0, N)
	for i := 0; i < N; i++ {
		ins = append(ins, c.Lit())
	}
	gs := make([]z.Lit, N/2)
	for i := 0; i < N/2; i++ {
		j := len(ins) - 1 - i
		a, b := ins[i], ins[j]
		g := c.And(a, b)
		gs[i] = g
	}
	for i := 0; i < N/2; i++ {
		j := len(ins) - 1 - i
		a, b := ins[i], ins[j]
		g := c.And(a, b)
		if g != gs[i] {
			t.Errorf("invalid strash")
		}
	}
}

type op struct {
	a z.Lit
	b z.Lit
	g z.Lit
}

func TestCLogic(t *testing.T) {
	c := NewC()
	a := c.Lit()
	b := c.Lit()
	ops := []op{
		{a: c.T, b: c.Lit()},
		{a: c.F, b: c.Lit()},
		{a: a, b: a},
		{a: a, b: a.Not()},
		{a: a, b: b},
		{a: b, b: a},
		{a: c.Lit(), b: c.Lit()}}

	for i := range ops {
		ops[i].g = c.And(ops[i].a, ops[i].b)
	}
	if ops[0].g != ops[0].b {
		t.Errorf("t simp")
	}
	if ops[1].g != ops[1].a {
		t.Errorf("f simp")
	}
	if ops[2].g != ops[2].a {
		t.Errorf("= simp")
	}
	if ops[3].g != c.F {
		t.Errorf("!= simp")
	}
	if ops[4].g != ops[5].g {
		t.Errorf("h simp")
	}
}

func TestEval(t *testing.T) {
	c := NewC()
	a, b := c.Lit(), c.Lit()
	g := c.And(a, b)
	_ = g
	log.Printf("%s = %s and %s\n", g, a, b)
	vs := make([]bool, 5)
	vs[2], vs[3] = true, true
	log.Printf("b4 %+v\n", vs)
	c.Eval(vs)
	log.Printf("after %+v\n", vs)

	if !vs[4] {
		t.Errorf("bad and eval")
	}
	if !vs[1] {
		t.Errorf("bad const eval")
	}
}

var rnd = rand.New(rand.NewSource(1))

func TestEval64(t *testing.T) {
	c := NewC()
	a, b := c.Lit(), c.Lit()
	c.And(a, b)
	vs := make([]uint64, 5)
	for i := 0; i < 5; i++ {
		vs[i] = uint64(rnd.Int63())
	}
	c.Eval64(vs)
	for i := 0; i < 63; i++ {
		s := uint64(1 << uint64(i))
		a := (vs[2] & s) != 0
		b := (vs[3] & s) != 0
		c := (vs[4] & s) != 0
		if a && b && !c {
			t.Errorf("not true")
		} else if (!a || !b) && c {
			t.Errorf("not false")
		}
	}
}

func ExampleC_equiv() {
	L := NewC()
	a, b, c := L.Lit(), L.Lit(), L.Lit()
	c1 := L.Ors(a, b, c)
	c2 := L.Ors(a, b, c.Not())
	g1 := L.And(c1, c2)
	g2 := L.Or(a, b)
	// create a "miter", test whether "(a b c) and (a b -c)" is equivalent to "(a b)",
	// by testing whether there is an assignment to {a,b,c} which makes the respective
	// circuits have a different value.
	m := L.Xor(g1, g2)

	// encode to sat
	s := gini.New()
	L.ToCnfFrom(s, m)

	// assume the miter
	s.Assume(m)
	r := s.Solve()
	if r == 1 {
		// not equivalent, model is a witness to different valuations.
		fmt.Printf("sat\n")
	} else {
		// equivalent.
		fmt.Printf("unsat\n")
	}
	//Output: unsat
}

type cAdder struct {
	c   *C
	f   z.Lit
	buf []z.Lit
}

func (a *cAdder) Add(m z.Lit) {
	if m != z.LitNull {
		a.buf = append(a.buf, m)
		return
	}
	clause := a.c.F
	for _, m := range a.buf {
		clause = a.c.Or(clause, m)
	}
	a.buf = a.buf[:0]
	a.f = a.c.And(a.f, clause)
}

// spread returns a number [0,1] that indicates how full the strash array is
// a value of 0 means that no entries are filled, a value of 1 means that all entries are filled
// a higher value indicates better spread of the hash keys over the array
func spread(c *C) float64 {
	filled := 0
	for _, v := range c.strash {
		if v != 0 {
			filled++
		}
	}
	return float64(filled) / float64(len(c.strash))
}

func BenchmarkStrash(b *testing.B) {
	var spreadSum float64
	strashFuncs := map[string]func(a, b z.Lit) uint32{
		"shift-negate-multiply": func(a, b z.Lit) uint32 {
			return uint32(^(a << 13) * b)
		},
		"fastutil-phimix": func(a, b z.Lit) uint32 {
			h := uint32(a * b * 0x9E3779B9)
			return h ^ (h >> 16)
		},
		"fnv": func(a, b z.Lit) uint32 {
			hasher := fnv.New32()
			hasher.Write([]byte{byte(a), byte(a >> 8), byte(a >> 16), byte(a >> 24), byte(b), byte(b >> 8), byte(b >> 16), byte(b >> 24)})
			return hasher.Sum32()
		},
		"fnv-direct": func(a, b z.Lit) uint32 {
			hash := uint32(2166136261)
			prime := uint32(16777619)
			hash = (hash * prime) ^ uint32(a)
			hash = (hash * prime) ^ uint32(a>>8)
			hash = (hash * prime) ^ uint32(a>>16)
			hash = (hash * prime) ^ uint32(a>>24)
			hash = (hash * prime) ^ uint32(b)
			hash = (hash * prime) ^ uint32(b>>8)
			hash = (hash * prime) ^ uint32(b>>16)
			hash = (hash * prime) ^ uint32(b>>24)
			return hash
		},
		"add": func(a, b z.Lit) uint32 {
			return uint32(a + b)
		},
		"mult": func(a, b z.Lit) uint32 {
			return uint32(a * b)
		},
		"and": func(a, b z.Lit) uint32 {
			return uint32(a & b)
		},
		"or": func(a, b z.Lit) uint32 {
			return uint32(a | b)
		},
		"64bit->32bit magic number hash": func(a, b z.Lit) uint32 {
			return uint32((uint64(a)<<32 | uint64(b)) * 0x8000000080000001 >> 32)
		},
		"2x32bit->16bit magic number hash, both msb": func(a, b z.Lit) uint32 {
			return (((uint32(a) * 0x80008001) >> 16) << 16) | ((uint32(b) * 0x80008001) >> 16)
		},
		"2x32bit->16bit magic number hash, a msb, b lsb": func(a, b z.Lit) uint32 {
			return (((uint32(a) * 0x80008001) >> 16) << 16) | ((uint32(b) * 0x80008001) >> 16)
		},
	}
	for name, strash := range strashFuncs {
		strashCode = strash
		for n := 1; n <= 1000; n *= 10 {
			b.Run(fmt.Sprintf("%s-%d", name, n), func(b *testing.B) {
				spreadSum = 0
				for i := 0; i < b.N; i++ {
					circuit := NewC()
					ca := &cAdder{
						c: circuit,
						f: circuit.T}
					gen.Seed(time.Now().Unix())
					b.StartTimer()
					gen.Rand3Cnf(ca, n*100, n*300)
					b.StopTimer()
					spreadSum += spread(circuit)
				}
				b.ReportMetric(spreadSum/float64(b.N), "spread")
			})
		}
	}
}
