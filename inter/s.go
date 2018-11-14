// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package inter

import "github.com/irifrance/gini/z"

// Interface Solveable encapsulates a decision
// procedure which may run for a long time.
//
// Solve returns
//
//  1  If the problem is SAT
//  0  If the problem is undetermined
//  -1 If the problem is UNSAT
//
// These error codes are used throughout gini.
type Solvable interface {
	Solve() int
}

// Interface GoSolvable encapsulates a handle
// on a Solve running in its own goroutine.
type GoSolvable interface {
	GoSolve() Solve
}

// Adder encapsulates something to which
// clauses can be added by sequences of
// z.LitNull-terminated literals.
type Adder interface {

	// add a literal to the clauses.  if m is z.LitNull,
	// signals end of clause.
	//
	// For performance reasons of reading big dimacs files,
	// Add should not be used unless no other goroutine
	// is accessing the object implementing adder.  Other
	// methods may provide safety in the presence of multiple
	// goroutines.  Add in general does not.
	//
	// Add should not be called under assumptions or test
	// scopes.  Doing so yields undefined behavior.
	//
	Add(m z.Lit)
}

// Interface MaxVar is something which records the
// maximum variable from a stream of inputs (such
// as Adds/Assumes) and can return the maximum of
// all such variables.
type MaxVar interface {
	MaxVar() z.Var
}

// Liter produces fresh variables and returns the corresponding
// positive literal.
//
type Liter interface {
	Lit() z.Lit
}

// Model encapsulates something from which a model
// can be exracted.
type Model interface {
	Value(m z.Lit) bool
}

// Assumable encapsulates a problem
type Assumable interface {
	Assume(m ...z.Lit)
	Why(dst []z.Lit) []z.Lit
}

// Interface Testable provides an interface for scoped
// assumptions.
//
// Testable provides scoped light weight Solving and tracking
// of implications.
//
// A Solvable and Assumable which also implements Testable has the following
// semantics w.r.t. Assumptions.  All calls to Assume which are subsequently
// Tested before calling Solve() remain until a corresponding call to Untest.
//
// Put another way, Solve() consumes and forgets any untested assumptions
// for a given Solve process.  To forget tested assumptions, the user
// must call Untest().  Tests and Untests may be nested.
type Testable interface {
	Assumable

	// Test the current assumptions under unit propagation.
	// place the resulting propagated literals since the last
	// test in dst and return
	//
	//  result: -1 for UNSAT, 1 for SAT, 0 for UNKNOWN
	//  out: the propagated literals since last Test,
	//       stored in dst if possible, or nil if out is nil.
	//
	// Once Test is called on a set of assumptions, all
	// future calls to Solve do not consume and forget
	// assumptions prior to test.
	Test(dst []z.Lit) (result int, out []z.Lit)

	// Untest removes the assumptions from the last test.
	// Untest returns -1 if the result is UNSAT, and
	// 0 (indicating unknown) otherwise.  If the result is
	// -1, then Test should not be called.
	Untest() int

	// Reasons returns the reasons for implied, storing
	// the result in rs if possible.
	Reasons(rs []z.Lit, implied z.Lit) []z.Lit
}

// Interface S encapsulates something capable
// of a complete incremental SAT interface
// enabling composing solvable, assumable, model, testable,
// and GoSolveable.
type S interface {
	MaxVar
	// Although an S can generate literals via Liter, it
	// doesn't have to.  One can just send arbitrary variables
	// via Adder, Assume, etc.  Liter is useful for applications
	// which need a way to know how to generate new variables/literals.
	Liter
	Adder
	Solvable
	GoSolvable
	Model
	Testable

	// Can create a copy.  A copy copies everything in the S interface
	// and nothing more (such as simplifiers).
	SCopy() S
}

// Interface Sv encapsulates an S which has the need or capacity to use inner
// variables which are hidden from the user.
type Sv interface {
	S

	// Inner returns the positive literal of a new inner variable.
	Inner() z.Lit

	// FreeInner frees the previously inner-allocated variable
	// associated with m.  If m's variables was not previously
	// allocated with Inner, then FreeInner and all subsequent
	// usage of Sv is undefined.
	FreeInner(m z.Lit)
}

// Interface Sc is an interface for a concurrent solver which
// must be stopped in order to free goroutines.
type Sc interface {
	S

	// Stop stops the Sc and should be called once.  Once stop
	// is called all behavior of Sc is undefined.
	Stop()
}

// CnfSimp provides an interface for clause based
// simplifications.
//
//
type CnfSimp interface {
	// OnAdded is called with an identifier `c` and
	// a set of literals `ms` whenever `ms` is added.
	// Since `ms` is added, it is known to be non-tautological,
	// have no duplicate literals, and no literals known to be
	// true or false under all previously added clauses.
	//
	// Also, no learnt clauses are passed to OnAdded.
	OnAdded(c z.C, ms []z.Lit)

	// CRemap is called by the solver whenever it undergoes
	// clause compaction (which in turn happens sometimes
	// during clause garbage collection).  Since z.C values
	// are ephemeral, they may change.  CRemap gives provides
	// Cnf with the changed values.  After CRemap is called,
	// the
	CRemap(cm map[z.C]z.C)

	// Simplify does preprocessing on added clauses.
	//
	// Simplify returns status like Solve (1:sat, -1:unsat, 0:unknown).
	//
	// Simplify should populate `rms` with clauses to be removed once the
	// simplification is done, attempting to use `rmSpace` if possible
	// to house the ids of clauses to be removed.
	//
	// Adding clauses works as follows. If a solver `s` implements Simplify, and
	// has a CnfSimp associated with it, then it calls `CnfSimp.Simplify` to implement
	// s.Simplify. `Cnf.Simplify` may then call `s.Add` like normal.  When
	// a clause is successfully added, `s.Add` will then call `Cnf.OnAdded`.
	//
	// This convoluted way of dealing with adding and removing clauses
	// allows a solver to store and manipulate clauses independently
	// of simplifications.  Since the solvers clause representation can
	// quite complex, subtle and optimised, this interface is in fact
	// much easier than working within most solvers, including xo.
	Simplify(rmSpace []z.C) (status int, rms []z.C)
}

// Simplifier is a facet of a Solver for simplifications.
type Simplifier interface {
	// See CnfSimp
	SetCnfSimp(cnfSimp CnfSimp)

	// Simplify returns 1: if the result is sat, -1 if the result
	// is unsat, and 0 if unknown.
	//
	// Simplify returns 0 and does nothing SetCnfSimp has not been called
	// with a non-nil argument.  Otherwise, Simplify calls cnfSimp.Simplify
	// and returns the status after removing the requested claues.
	Simplify() int
}
