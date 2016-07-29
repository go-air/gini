# Gini SAT Solver

The Gini sat solver is a fast, clean SAT solver written in go. 

[![GoDoc](https://godoc.org/github.com/IRIFrance/gini?status.svg)](https://godoc.org/github.com/IRIFrance/gini)

[Google Group](https://groups.google.com/d/forum/ginisat)

# The SAT Problem

The SAT problem is perhaps the most famous NP-complete problem.  As such, SAT
solvers can be used to try to solve hard problems, such as travelling salesman
or RSA cracking. In practice, many SAT problems are quite easy (but not
decryption problems...yet).  The solvers are used in software verification,
hardware verification and testing, AI planning, routing, etc.

The SAT problem is a Boolean problem.  All variables can either be true or
false, but nothing else.  The SAT problem solves systems of Boolean
constraints, called clauses.  Namely, SAT solvers work on conjunctive normal
form problems (CNFs).  There are many ways to efficiently code arbitrary logic
into CNF, so this is not so much a restricting factor.

## CNF
A CNF is a conjunction of clauses

    c1 and c2 and ... and cM

Each c[i], i in [1..M], is a clause, which is of the form

    m1 or m2 or ... or mN

where each m[i], i in [1..N] is either a Boolean variable (such as x), or the
negation of a Boolean variable (such as not(y)).  An expression which is either
a Boolean variable or its negation is called a "literal".

In the following, we refer to variables simply by integers 1,2,3,...

Clauses are often written in succint form

    -3 11 12 14 -257

Numerical negation indicates logical negation, and spaces are disjunctions
"or".  Sometimes "+" is used for "or".

Conjunctions are just concatenation of clauses.  We can parenthesize clauses
such as

    (1 -2) (2 -3) (3 -4) (4 -1)

which expresses a set of clauses whose satisfying assignments are

    {1,2,3,4}
        or
    {-1,-2,-3,-4}

## Models
A model of a CNF is a value for each of the variables which makes every clause 
in the CNF true.  The SAT problem is determining whether or not a model exists
for a given set of clauses.


## Proofs

### Resolution

Resolution is a form of logical reasoning with conjunctions of clauses.  Given
2 clauses of the form

    C + v
and

    D + -v

We can conclude that 

    C + D

must be true.

Here, C and D are arbitrary clauses.

Resolution proof of unsatisfiability is a derivation of the empty disjuction
(false) by means of resolution.  Resolution proofs, even minimally sized ones,
can be very large, exponentially larger than the innput problem.

Modern SAT solvers mostly rely on performing operations which correspond to
bounded size (in terms of number of variables) number of resolutions.  Given
this fact together with the fact that the minimal proofs can be exponentially
large in the number of variables, some problems can take an exponential amount
of time.

Nonetheless, many SAT solvers have heuristics and are optimised so much that
even hard problems become tractable.  With up to several tens of millions of
resolutions happening per second on one modern single core CPU, even problems
with known exponential bounds on resolution steps can be solved.

# Concurrency
Gini is written in Go and uses several goroutines by default for garbage
collection and system call scheduling.  There is a "core" single-goroutine
solver, xo, which is in an internal package for gutsy low level SAT hackers. 

## Connections to solving processes
Gini provides safe connections to solving processes which are guaranteed to not
lose any solution found, can pause and resume, run with a timeout, test without
solving, etc.

## Solve-time copyable solvers.
Gini provides copyable solvers, which can be safely copied *at solvetime during
a pause*.

## Ax
Gini provides an "Assumption eXchange" package for deploying solves
under different sets of assumptions to the same of underlying constraints
in parallel. This can give linear speed up in tasks, such as PDR/IC3, which 
generate lots of assumptions.

## Concurrency package
A concurrent solver is in the works but not yet publicly available.


# CRISP

Gini provides a definition and reference implementation for CRISP 1.0, the 
compressed incremental SAT protocol.  The protocol is a client-server
wire protocol which can dispatch an incremental sat solver with very
little overhead as compared to direct API calls.  The advantage of 
using a protocol is that it allows arbitrary tools to implement the solving
on arbitrary hardware without affecting the client.  

Many SAT applications are incremental and easily solve huge numbers of problems
while only a few problems are hard.  CRISP facilitates pulling out the big guns
for the hard problems while not affecting performance for easy problems.  Big
guns tend to be harder to integrate into applications because of compilation
issues, hardware requirements, size and complexity of the code base, etc.
Applications that use CRISP can truly isolate themselves from the woes of
integrating big guns while benefiting on hard problems.

CRISP also allows language independent incremental SAT solving.  The applications
and solvers can be readily implemented without the headache of synchronizing
programming language, compilers, or coding style.

We are planning on implementing some CRISP extensions, namely the multiplexing
interface which will enable (possibly remote) clients to control
programmatically partitioning or queuing of related SAT problems.

# Distributed
The CRISP protocol provides a basis for distributed solving.  Gini implements
a CRISP-1.0 client and server.  

A command, crispd, is supplied for the CRISP server.

