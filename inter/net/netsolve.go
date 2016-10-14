// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package net

import (
	"github.com/irifrance/gini/inter"
	"time"
)

// Interface NetSolve represents an asynchronous  connection to a call to solve
// over a network, such as crisp.(*Client).Solve().  It is analagous to
// gini.Solve, but adds error returns for underlying network/operating system
// errors.
//
// NetSolve may be constructed by a call to crisp.(*Client).GoSolve().
//
// This interface is NOT safe for usage in multiple goroutines and the
// following caveats must be respected:
//
//  1. Once any method of NetSolve returns a result from the underlying solver
//  NetSolve must no longer be used
//  2. Once any method of a NetSolve returns a non-nil error, it must no longer
//  be used.
type Solve interface {
	Test() (int, bool, error)
	Try(d time.Duration) (int, error)
	Stop() (int, error)
}

// ToNetSolve wraps a Solve to make a conforming
// NetSolve, where all errors are nil
func ToSolve(s inter.Solve) Solve {
	return &nsWrap{s}
}

type nsWrap struct {
	s inter.Solve
}

func (w *nsWrap) Test() (int, bool, error) {
	r, b := w.s.Test()
	return r, b, nil
}

func (w *nsWrap) Try(d time.Duration) (int, error) {
	return w.s.Try(d), nil
}

func (w *nsWrap) Stop() (int, error) {
	return w.s.Stop(), nil
}
