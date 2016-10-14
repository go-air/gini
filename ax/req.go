// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package ax

import (
	"fmt"
	"github.com/irifrance/g/z"
	"sync"
	"time"
)

// Type ReqFlag describes options for submitting requests
// to a parallel incremental solver.
type ReqFlag uint32

const (
	// if you want a model back
	ReqModel ReqFlag = 1 << iota
	// if you want an explanation
	ReqWhy
)

// Implements Stringer.
func (f ReqFlag) String() string {
	buf := []byte{'-', '-'}
	if f.Model() {
		buf[0] = 'm'
	}
	if f.Why() {
		buf[1] = 'w'
	}
	return string(buf)
}

// Model tells whether the flag is requesting a model.
func (f ReqFlag) Model() bool {
	return f&ReqModel != 0
}

// Why tells whether the flag is requesting an explanation.
func (f ReqFlag) Why() bool {
	return f&ReqWhy != 0
}

// Type Request is a request to solve under a given set of
// assumptions.
type Request struct {
	Id    int           // An identifier for keeping track of requests.
	Flag  ReqFlag       // Flags for the request.
	Limit time.Duration // time limit (0 for no time limit)
	Ms    []z.Lit       // Assumptions for solve.
}

// String Implements Stringer.
func (r *Request) String() string {
	return fmt.Sprintf("imux.Request[%d %s %+v]", r.Id, r.Flag, r.Ms)
}

// Type RequestGen encapsulates generating requests with the same
// flag.
type RequestGen struct {
	mu    sync.Mutex
	count int
	flag  ReqFlag
	limit time.Duration
}

// SetFlag sets the flags for subsequent generated requests.
func (g *RequestGen) SetFlag(f ReqFlag) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.flag = f
}

// Flag returns the ReqFlag for subsequent generated requests.
func (g *RequestGen) Flag() ReqFlag {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.flag
}

// New creates a new solve request over assumptions ms.
func (g *RequestGen) New(ms ...z.Lit) *Request {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := g.count
	g.count++
	req := &Request{
		Id:    id,
		Flag:  g.flag,
		Limit: g.limit,
		Ms:    make([]z.Lit, len(ms))}
	copy(req.Ms, ms)
	return req
}

// Limit causes requests generated by g.New() to
// have time limit d.
func (g *RequestGen) Limit(d time.Duration) {
	g.limit = d
}

// NewRequestGen creates a new request generator.
func NewRequestGen() *RequestGen {
	return &RequestGen{
		flag:  ReqModel | ReqWhy,
		limit: 0}
}

var reqGen = NewRequestGen()

// SetFlag sets the flag for the package default request generator.
func SetFlag(f ReqFlag) {
	reqGen.SetFlag(f)
}

// Flag retrieves the flag for the package default request generator.
func Flag() ReqFlag {
	return reqGen.Flag()
}

// Limit limits the duration of a subsequent requests generated
// by NewRequestGen.
func Limit(d time.Duration) {
	reqGen.Limit(d)
}

// NewRequest creates a new request with the package default request
// generator.
func NewRequest(ms ...z.Lit) *Request {
	return reqGen.New(ms...)
}
