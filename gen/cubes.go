// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package gen

import (
	"math/rand"

	"github.com/go-air/gini/z"
)

type RandCuber interface {
	RandCube(dst []z.Lit) []z.Lit
}

func NewRandCuber(maxSize int, maxVar z.Var) RandCuber {
	return &cubes{
		minSize: 1,
		maxSize: maxSize,
		maxVar:  maxVar,
		uniq:    make(map[z.Var]struct{}, maxSize)}
}

type cubes struct {
	minSize int
	maxSize int
	maxVar  z.Var
	uniq    map[z.Var]struct{}
}

func (c *cubes) SetMinSize(s int) {
	c.minSize = s
}

func (c *cubes) SetMaxSize(s int) {
	c.maxSize = s
}

func (c *cubes) RandCube(dst []z.Lit) []z.Lit {
	dst = dst[:0]
	sz := c.minSize + rand.Intn(c.maxSize-c.minSize+1)
	for k := range c.uniq {
		delete(c.uniq, k)
	}
	for i := 0; i < sz; i++ {
		m := z.Lit((rand.Intn(int(c.maxVar))+1)*2 + rand.Intn(2))
		if _, ok := c.uniq[m.Var()]; ok {
			continue
		}
		dst = append(dst, m)
		c.uniq[m.Var()] = struct{}{}
	}
	return dst
}
