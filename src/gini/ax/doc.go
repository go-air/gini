// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

// Package ax supplies an "assumption eXchange".
//
// An ax.T allows running solve requests for sets of assumptions all of which
// share a set of constraints in different goroutines.
//
// The main object is an ax.T, which allows submitting and retrieving solve
// requests.  Solve requests are requests to solve with a given set of
// assumptions, possibly with some options specified.  Each request is related
// to the same fixed set of constraints, which are provided when the ax.T is
// constructed.
//
// An ax.T is constructed with a single incremental solver, and a capacity C.
// The solver should have some constraints added.  New solvers are created by
// copying an existing solver in the ax.T if a solve request is made and there
// is only 1 solver available and there are fewer than C solvers.
//
// Solve responses are supplied via the ax.T interface.  The user may request
// models or explanations, but must know if she wants this information when the
// request is submitted.
package ax
