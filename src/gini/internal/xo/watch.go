// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"fmt"
	"gini/z"
)

// Watch holds other blocking literal, clause location (
// and 1 bit for whether binary
type Watch uint64

// TBD: make this uintptr size dependent for 32 bit architectures
const (
	litBits = 31
	litMask = ((1 << 31) - 1)
	locMask = uint64(0xffffffff << litBits)
	binMask = (1 << 63)
)

// MakeWatch creates a watch object for clause location loc
// blocking literal o, and isBin indicating whether the referred to
// clause is binary (comprised of 2 literals)
func MakeWatch(loc CLoc, o z.Lit, isBin bool) Watch {
	v := uint64(0)
	if isBin {
		v |= binMask
	}
	v |= uint64(o)
	v |= uint64(loc) << litBits
	return Watch(v)
}

// return the other blocking literal
func (w Watch) Other() z.Lit {
	return z.Lit(w & litMask)
}

// whether clause is binary
func (w Watch) IsBinary() bool {
	return w >= binMask
}

// the location of the null-terminated literals in the clause
func (w Watch) CLoc() CLoc {
	return CLoc(w >> litBits)
}

// return a watch with all info the same, but the CLoc updated
// to o.
func (w Watch) Relocate(o CLoc) Watch {
	v := uint64(w)
	v &= ^locMask
	v |= uint64(o) << litBits
	return Watch(v)
}

// a human readable representation
func (w Watch) String() string {
	return fmt.Sprintf("Watch{CLoc: %s, Other: %s, Bin: %t}", w.CLoc(), w.Other(), w.IsBinary())
}
