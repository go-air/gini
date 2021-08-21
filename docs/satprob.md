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
into CNF, so this is not so much a restricting factor.  Nonetheless, we present
CNF and the problem below in a brief self-contained fashion which we find
useful.  Readers interested in more depth should consult Wikipedia, or The
Handbook of Satisfiability, or Donald Knuth's latest volume of The Art of
Computer Programming.

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
can be very large, exponentially larger than the input problem.

Modern SAT solvers mostly rely on performing operations which correspond to
bounded size (in terms of number of variables) number of resolutions.  Given
this fact together with the fact that the minimal proofs can be exponentially
large in the number of variables, some problems can take an exponential amount
of time.

Nonetheless, many SAT solvers have heuristics and are optimised so much that
even hard problems become tractable.  With up to several tens of millions of
resolutions happening per second on one modern single core CPU, even problems
with known exponential bounds on resolution steps can be solved.
