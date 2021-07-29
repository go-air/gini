// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package net

import (
	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"
)

type Solvable interface {
	Solve() (result int, err error)
}

type GoSolvable interface {
	GoSolve() Solve
}

type Adder interface {
	Add(m z.Lit) error
}

type Model interface {
	Value(m z.Lit) (bool, error)
}

type Assumable interface {
	Assume(m ...z.Lit) error
	Why(dst []z.Lit) ([]z.Lit, error)
}

type Testable interface {
	Test(dst []z.Lit) (result int, out []z.Lit, err error)
	Untest() (int, error)
	Reasons(rs []z.Lit, implied z.Lit) ([]z.Lit, error)
}

// Interface NetS composes solving, asynchronous interface,
// model extraction, assumptions, and scoping with
// error returns for net operations.
type S interface {
	Add(m z.Lit) error
	Solvable
	GoSolvable
	Model
	Assumable
	//	NetTestable
}

// ToNetS wraps an S making it return nil errors
// to conform to the NetS interface.
func ToS(s inter.S) S {
	return &nWrap{s}
}

type nWrap struct {
	s inter.S
}

func (w *nWrap) Add(m z.Lit) error {
	w.s.Add(m)
	return nil
}

func (w *nWrap) Assume(ms ...z.Lit) error {
	w.s.Assume(ms...)
	return nil
}

func (w *nWrap) Solve() (int, error) {
	return w.s.Solve(), nil
}

func (w *nWrap) Why(dst []z.Lit) ([]z.Lit, error) {
	return w.s.Why(dst), nil
}

func (w *nWrap) Value(m z.Lit) (bool, error) {
	return w.s.Value(m), nil
}

func (w *nWrap) GoSolve() Solve {
	return ToSolve(w.s.GoSolve())
}

func (w *nWrap) Test(dst []z.Lit) (int, []z.Lit, error) {
	r, ms := w.s.Test(dst)
	return r, ms, nil
}

func (w *nWrap) Untest() (int, error) {
	return w.s.Untest(), nil
}

func (w *nWrap) Reasons(dst []z.Lit, implied z.Lit) ([]z.Lit, error) {
	return w.s.Reasons(dst, implied), nil
}
