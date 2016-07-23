// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

// Package crisp provides an implementation (client, server) of the compressed
// incremental sat protocol (CRISP), version 1.0.
//
// Introduction
//
// CRISP is a protocol designed for efficient communication of incremental SAT
// solving accross a wire.  It is designed to be small, simple, and easy to
// deploy.
//
// The main goals of the protocol is to minimize small round-trips for
// communicating data and have a compact representation of the data while
// enabling incremental solving.  One important goal is to make it so that real
// applications can talk the protocol instead of linking against a solver
// without significant loss of performance, even for easy problems.
//
// Another goal of the protocol is be easily extensible to address problems
// such as distributed proof representation, quantifiers, scheduling, problem
// partitioning, etc. These goals are however longer term and likely best
// addressed by extensions.
//
// Wire Data
//
// The basic unit of communication is encoding into unsigned 32bit integers.
// Every piece of data communicated between the client and the server takes the
// logical form of a uint32.
//
// This space houses literals and communication instructions/directives between
// the client and the server.  These instructions/directives are called
// protocol points in the following.
//
// We only use about a dozen protocol points, but to enable extensions, we
// reserve 256 integers for protocol points, at the high end of the range
// representable by uint32. The rest of the space is used to house variables
// and literals, coded in the tradition of SAT solvers (see Adding below).
//
// Flow Overview
//
// The protocol works as follows
//
//  1. client negotiates connection with server
//  2. client then requests (<add> or <assume>) as many times as it likes.
//  The server does not respond to these requests.
//  3. client then requests <solve>.  This enters
//  a loop between the client and the server on the same connection as
//  follows:
//    a. client: <solve>
//    b. server: <unknown|sat|unsat|end>
//    c. client: <continue|end>
//  4. client sends (<model> or <modelfor> or <failed> or <failedfor>) as many times as it likes.
//  Each time it sends one of these operations, the server responds with the requested data.
//  5. Optionally, client sends <reset> and flow goes to state 2.
//
//  6. client sends <quit>, both ends disconnect.
//
// In the above, step 3,
//
//  (b,c) repeats until server sends <sat>, <unsat>, or <end>;
//
// Thus step 3 can be represented by the regular expresion
//
//  client:<solve> (server:<unknown> client:<continue>)* \
//    (server:<unknown> client:<end>)? \
//    (server:<end>|server:<sat>|server:<unsat>))
//
// The meaning of a server sending <end> is that it is not willing or unable to
// make more progress on solving the problem and does not know the answer.  The
// meaning of the server sending <unknown> is that it doesn't know the answer
// and is willing and able to make more progress solving the problem.
//
// In this loop, the client does not need to wait between reads and writes,
// since presumably the server is doing the solving and may want to throttle
// the round trips accordingly on non-trivial problems.
//
// If we call the regular expression above S, the allowable overall (error-free)
// flow interactions can be represented by the regular expression
//
//  1 ((2* S 4*)* 5?)* 6
//
// There are also error conditions, which simply abort the sequence and cause
// a disconnect; these are not represented in this overview but are detailed below.
//
// Varuint Encoding
//
// The wire format then uses var-uint encoding (a stream of bytes each of which
// indicates whether it is the last with 1 bit and uses 7bits to encode the
// non-zero LSBs of the value).  This works as follows.
//
//  Let u[i] for i in 0..31 be the bits of an unsigned 32 bit int u.  The encoding
//  of u is a slice of bytes e.
//    e[0]&127 houses the 7 most LSB of u: u[0:7].
//      If u >> 7 == 0, then the length of e is 1.
//    Otherwise e[1]&127 houses u[7:14].
//      If u >> 14 == 0, then the length of e is 2.
//    Otherwise e[2]&127 houses u[14:21].
//    ...
// This continues for all 32 bits.  Also,
//
//  For every element e[i] of e except the last, e[i]&128 != 0.
//
// Decoding.  The server may decode a stream of varuint32 data and then, for
// each value v, test whether it is a command/code point by testing
//
// v >= 0xffffffff - 256
//
// if true, v is a command, if false, v is a literal.
//
// Rational: We distinguish the special value '0' as a literal to
// zero-terminate clauses and lists of assumptions or lists of failed literals.
// This is conventional and only requires 1 byte in varuint format.  Commands
// happen with much less frequency in the protocol when there is a potential
// need to send lots of data fast (eg loading a big dimacs).  Commands are
// large values and hence have larger varuint size.
//
// Protocol Points
//
// We reserve 256 code points for extensions to learning, optimisation, etc.
// In this proposal we have only the following op codes
//
//   <add>
//   <assume>
//   <solve>
//   <continue>
//   <end>
//   <error>
//   <failed>
//   <failedfor>
//   <model>
//   <modelfor>
//   <sat>
//   <unsat>
//   <unknown>
//   <quit>
//   <reset>
//   <ext>
//   <key>
//
// The remaining protocol points are for extensions.
//
// Variables and Literals
//
// Variables are Boolean (can be either true/false) and each variable is
// associated with a uint32.  A "literal" in propositional logic is just a
// variable or the logical negation of a variable.  CRISP uses standard
// SAT-solver encoding of variables and literals.
//
// If a variable is indicated by some uint32, say x, then a literal m over
// x is
//
//  1. (x<<1)                if m is positive
//  2. (x<<1) | 1            if m is negative
//
// Since we have uint32 coding, and 256 coding points, the maximum variable
// representable in the protocol is
//
//  (0xffffffff - 256) >> 1
//
// or equivalently
//
//  2147483519
//
// Connection negotiation
//
// Upon connecting, the server replies with "CRISP" (as 5 uint32s) followed by
// a uint32 "v" indicating the protocol version number which is comprised of a
// major version and a minor version.  The major version is the upper 8 MSB
// of "v" and the minor protocol version is the 24 LSB of "v".
//
// Some servers may want to protect access by requring a key.  If they do this,
// then they can simply wait for the first op from the client.  If the first op
// is not <key>, then the server can disconnect the client.
//
// Clients connecting to <key>-passed servers can send the <key> op followed by
// the key length in terms of uint32 atoms (it must be 4-byte aligned).  Then
// the client sends the key.
//
// After receiving the key, the server could just disconnect if it doesn't accept
// the client.
//
// Adding
//
// Adding adds permanent constraints to the solver on the server side.
// Each constraint is in the form of a clause, which is a disjunction of
// literals.
//
// When the client adds clauses, it sends the <add> op followed by
// a list of clauses, where each clause is a null-terminated list
// of literals.  When it is done adding clauses, it sends a <end>.
//
// The server does not respond to <add> or <end> ops.
//
// Assuming
//
// When the client makes assumptions, it sends the <assume> op followed by
// a null terminated list of literals.  The server does not respond to this
// op.
//
// The server takes into account the assumptions temporarily, only for the next
// call to solve.  Subsequent calls to solve after the next call are not
// effected.
//
// Solving
//
// Solving is the only part of the protocol which involves several
// round trips between the client and the server.
//
// After the client sends <solve>, it reads from the server in a blocking read
// until the server sends a response.  The response is either <sat>, <unsat>,
// <unknown>, or <end>.
//
//  If the response is <unknown>, then the client must respond with either
//  <continue> or <end>.  If the client sends <continue>,  the protocol enters
//  the same state as if the client had just sent <solve>.  If the client sends
//  <end>, the protocol exits the <solve> state; the client can add more clauses
//  or make more assumptions and try to solve again (or quit).
//
//  If the server response is <sat>,<unsat>, or <end> then client must not respond
//  with <continue> or <end>.  Instead, the client may optionally request models
//  if the response was <sat>, or optionally request failed assumptions if the
//  response was <unsat>.  In any event, the client may continue to <add> or
//  <assume> or may <quit>.
//
// Models
//
// Models are valuations of variables in a problem which satisfy the permanent
// constraints and assumptions in the previous <solve> interaction.  CRISP
// supports 2 mechanisms for models: partial and complete models, corresponding
// respectively to the <modelfor> and <model> operators.
//
// The <model> and <modelfor> operators can only be sent if the previous <solve> operator
// ended with a <sat> operator from the server and no <add> or <assume> has taken place since.
// Otherwise, the server sends <error> and disconnectes.
//
// Complete models are obtained using the <model> operator.  The client
// sends <model> and the server responds with a truth value for every variable,
// in the order of variable index/id 1,2,3,.... up until the maximum
// variable used in any added clause or assumption.
//
// The encoding of the truth values is the same for partial and complete models,
// and is described below.
//
// Partial models are obtained by using the <modelfor> operator.  The
// client sends <modelfor> followed by a null terminated list of literals.
// The server responds with the truth value for each of these literals in
// the same order specified specified by the client.
//
// In both <model> and <modelfor>, the list of truth values is encoded by the
// server as follows.  First, the server sends a uint32 indicating how many
// uint32s are to be sent subsequently to communicate the model.  Then:
//
//  Let N be the length of the list, and let let M = N%32 == 0 ? N : N + (32 - N%32).
//  Then M/32 uint32 values are sent by the server.  Let u[i] be the i'th value.
//  The truth value for element j of the list is
//
//     u[j/32] & (1 << j%32) != 0
//
// Failed Assumptions
//
// Failed assumptions are a subset of the assumptions from the last <solve>
// sequence which are sufficient to render the problem unsat.  Like the <model>
// and <modelfor> operators, there are 2 failed operators, <failed> and
// <failedfor>.
//
// <failed> retrieves a sufficient subset of all previously
// assumed literals to render the last problem posed to <solve> unsat.
//
// <failedfor> specifies a list of assumptions which are of interest to the
// client.  The server responds with the maximal subset of this list which
// intersects what would be returned by <failed>.
//
// Thus, when the client sends <failedfor> it follows this operator with
// a null terminated list of literals. But when the client sends <failed>
// it immediately waits the response from the server.
//
// In both cases, the server replies with a null terminated list of literals.
// If the server replies to <failedfor>, the list of literals in the response
// from the server must respect the order of the list provided by the client
// in <failedfor>.
//
// Error Conditions
//
// Errors are always sent from the server to the client.  Each <error> op
// is followed by a single uint32 value, which encodes more information
// about the error.
//
// Anytime the server responds with an <error>, the server subsequently
// closes the connection.
//
// The following are common errors:
//
// If the varuint encoding encodes a number outside the uint32 range
// then the server responds with an error.
//
// If the client has this error reading from the server, then it must disconnect.
//
// If a client requests a model or partial model and the previous solve dis
// not end with <sat>, the server responds with an error, (ErrNotSat)
//
// If a client requests failed assumptions and the previous solve was not unsat, the
// server responds with an error (ErrNotUnsat)
//
// If a client requests failed assumptions or model and an <add> or an <assume>
// has taken place since the last <solve>, the server responds with an error
// whose code is (ErrOutOfOrder)
//
// Other generic server errors are called "internal server errors", and have
// error code ErrInternal.
//
// Outlook
//
// I imagine that <learn> which adds a clause known to be redundant w.r.t.
// added clauses will become useful for distributed solving, but let's keep
// things simple first...
//
// This leaves space for developers and researchers to extend the protocol in
// the future for many possible means (DRAT, optimisation, learning,
// quantifiers, ...) without needing to re-engineer the wire protocol.
package crisp
