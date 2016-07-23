// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import "fmt"

// Type CLoc gives a temporary id which directly
// tells where the sequence of literals (zero/LitNull terminated) starts
type CLoc uint32

const (
	CLocNull CLoc = 0
	CLocInf       = 0xffffffff
)

func (p CLoc) String() string {
	return fmt.Sprintf("c%d", p)
}
