// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen

import (
	"math/rand"
	"sync"

	"github.com/irifrance/gini/inter"
	"github.com/irifrance/gini/z"
)

/// make the rng seedable
var rng = rand.New(rand.NewSource(33))
var mu sync.Mutex

func Seed(s int64) {
	mu.Lock()
	defer mu.Unlock()
	rng = rand.New(rand.NewSource(s))
}

// BinCycle generates
// (1,-2) (2,-3), (3,-4) ... (n-1, -(n)), (n, 1)
func BinCycle(dst inter.Adder, n int) {
	N := n + 1

	for i := 1; i < N; i++ {
		j := i + 1
		if j == N {
			j = 1
		}
		m, o := z.Var(i).Pos(), z.Var(j).Neg()
		dst.Add(m)
		dst.Add(o)
		dst.Add(z.LitNull)
	}
}

// Rand3Cnf generates a random 3cnf with
// n variables and m clauses.
func Rand3Cnf(dst inter.Adder, n, m int) {
	mu.Lock() // for package rng
	defer mu.Unlock()
	ms := make([]z.Lit, 3)
	for i := 0; i < m; i++ {
		for j := 0; j < 3; j++ {
			m := z.Lit(rng.Intn(2*n) + 2)
			ms[j] = m
			for j == 1 && ms[0].Var() == ms[1].Var() {
				ms[j] = z.Lit(rng.Intn(2*n) + 2)
			}
			for j == 2 && (ms[0].Var() == ms[2].Var() || ms[1].Var() == ms[2].Var()) {
				ms[j] = z.Lit(rng.Intn(2*n) + 2)
			}
		}
		dst.Add(ms[0])
		dst.Add(ms[1])
		dst.Add(ms[2])
		dst.Add(z.LitNull)
	}
}

// HardRand3Cnf generates a random 3cnf
// with n variables.
func HardRand3Cnf(dst inter.Adder, n int) {
	Rand3Cnf(dst, n, 4*n)
}

// Php generates a pigeon hole problem asking
// whether or not P pigeons can be placed
// in H holes with 1 pigeon per hole.
func Php(dst inter.Adder, P, H int) {
	for i := 0; i < P; i++ {
		for j := 0; j < H; j++ {
			dst.Add(PartVar(i, j, P))
		}
		dst.Add(0)
	}
	for i := 0; i < P; i++ {
		for j := 0; j < i; j++ {
			for h := 0; h < H; h++ {
				dst.Add(PartVar(i, h, P).Not())
				dst.Add(PartVar(j, h, P).Not())
				dst.Add(0)
			}
		}
	}
}
