// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package ax

import (
	"time"

	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"
)

// Interface T describes an assumptions exchanger (ax).
//
// Objects implementing T process incremental solving requests
// and give solving responses.  The fundamental operation supported
// is an exchange (Ex), in which exactly one of the following two
// events occur.
//
//  1. A request is submitted to the system and an inter.S is guaranteed
//     to process it unless T is Stop'd.
//  2. A response to some previous request is returned.
//
// Once the user is done with the object, it should be Stopped with Stop().
// The number of pending requests is returned in each Response.
//
// Objects implementing T must be safe for usage in mutlitple goroutines.
type T interface {
	// Ex blocks until an exchange occurs
	// If the request is accepted, ok is true, and false otherwise.
	// If a response is ready, it is returned in resp, otherwise
	// resp is nil.
	//
	// As a special case, if r is nil, then Ex blocks
	// until a response is ready but does not try to submit the request.
	Ex(r *Request) (resp *Response)

	// TryEx tries to perform an exchange but does not block.
	// It has the same semantics for Request and Response as
	// Ex, but it can return a nil response and not process
	// the supplied request.  In this case, ok is false.  Otherwise,
	// ok is true.
	TryEx(r *Request) (resp *Response, ok bool)

	// Stop stops the ax.
	Stop()
}

// NewT creates a new ax.T from a prototype solver inter.S.
// NewT uses proto as the first solving unit and each time
// a request is received, if:
//
//  1. there are less than cap copies; and
//  2. all existing copies are busy.
//
// then T will make another copy of proto.
//
// If cap < 1, then NewT panics.
func NewT(proto inter.S, cap int) T {
	return newAx(proto, cap)
}

// implementation comments from here down.

// Diagram of interactions over channels.  ax, Client, and u* each run a
// goroutine in parallel.  Synchronously, pool processes requests and sends
// them to a u, waiting for one to become available.   When u's are done,
// they send a resp to the pool over a 1-buffered channel, so atleast one
// can finish and become available for processing a request from the pool.
//
// The ax selects (s) between such client requests and responses from
// u's.  Responses from u's are sent immediately unbuffered to the client.
//
// The Client selects on the ability to send requests and receive responses from
// the ax.  She can send up to number(u)+1 requests in parallel.  and up to
// number(u) requests get processed in parallel.  if number(u) requests are
// outstanding (*), then pool accepts one more request and waits for a u to
// become available.
//
// When the ax receives a channel close on the request channel from the
// client, it immediately shuts down the u's, which select on a cancelation
// channel along with their request channel.
//
//
// -ax-(s)-----<Req<----------(s)---Client
//      |    |                 |
//    ---    --(o)             |
//    |      |  |              |
// ------(c)--  |              |
// |  |         |              |
// |  |----(+)--|----->cResp>--|
// |	|         |
// |  ^        (+)
// | uResp[1]   |
// |  ^         ->uReq>===========
// |  |                      | | |
// |  |---(*)---u.1<--(s)----| | |
// |  |---(*)---u.2<--(s)------| |
// |  |---(*)---...<--(s)--------|
// |	                     |  |  |
// |                       |  |  |
// =====(+)====>Cancel>=======---|
//
// (s)             select
// --| or |--      select branch (select all channels on path except if (+) or (*) intervenes)
//   |    |
// >,^             channel direction
// (*)             solve process, unknown duration.
// (+)             combinational code, short/synchronous time step duration
// (c)             channel closure case
// (o)             channel receive w/out closure (open)
// =               many select lines/paths
// u.i             processing unit/solver i
// [n]             n buffered channel
//
//
// Each u has a solver with a CNF.  When the pool dispatches a request to a u, it
// looks for one which has processed the most similar requests in terms of a sort
// of Hamming distance.  This heuristic means the selected u will likely be best
// tuned to solve such problems
//
// Responses are tagged with solve times so the client can decide on the
// estimated difficulty of requests to send.
//
// To solve SAT with DFS/DPLL/CDCL the client runs a solver which can mark
// branches as dispatched as well closed.  dispatched branches have unknown
// solutions/results and may be blocked in the client so the client will not
// revisit the same space as a dispatched solve.  However, if they are blocked,
// and the client side solver gives UNSAT, then the client side must wait for the
// dispatched solve responses to determine unsatisfiability.

type ax struct {
	reqChn   chan *Request
	respChn  chan *Response
	uRespChn chan *Response
	us       []*unit
	avail    []int
	cap      int
}

func newAx(proto inter.S, cap int) *ax {
	if cap < 1 {
		panic("cannot pool <= 0 ginis")
	}
	//log.Printf("creating pool with %s vars, cap %d\n", proto.MaxVar(), cap)
	m := &ax{
		reqChn:   make(chan *Request),
		respChn:  make(chan *Response),
		uRespChn: make(chan *Response, 1),
		us:       make([]*unit, 1, cap),
		avail:    make([]int, 1, cap),
		cap:      cap}
	m.us[0] = newUnit(proto, 0)
	m.avail[0] = 0
	go m.serve()
	return m
}

func (m *ax) Ex(req *Request) *Response {
	if req == nil {
		return <-m.respChn
	}
	select {
	case m.reqChn <- req:
		return nil
	case resp := <-m.respChn:
		return resp
	}
}

func (m *ax) TryEx(req *Request) (*Response, bool) {
	if req == nil {
		select {
		case resp := <-m.respChn:
			return resp, true
		default:
			return nil, false
		}
	}
	// Q(wsc) Priority to resp for routing?
	select {
	case m.reqChn <- req:
		return nil, true
	case resp := <-m.respChn:
		return resp, true
	default:
		return nil, false
	}
}

func (m *ax) Stop() {
	close(m.reqChn)
}

func (m *ax) serve() {
	for {
		select {
		case req, ok := <-m.reqChn:
			if !ok {
				m.shutdown()
				return
			}
			m.handleReq(req)
		case resp := <-m.uRespChn:
			m.handleResp(resp)
		}
	}
}

func (m *ax) handleReq(req *Request) {
	if len(m.avail) == 0 {
		resp := <-m.uRespChn
		m.handleResp(resp)
	}
	u := m.getunit(req)
	//log.Printf("start request %d by %d\n", req.Id, u.I)
	u.handleReq(m.uRespChn, req)
}

// invariant: we always have atleast one
// non-nil free u in m.us until
// the client requests len(m.us) solves
func (m *ax) getunit(req *Request) *unit {
	if len(m.us) < m.cap && len(m.avail) == 1 {
		// grow only as needed.  we create the next available gini
		// ahead of time so that we can copy a non-running
		// S without calling pause.
		ai := m.avail[0]
		protounit := m.us[ai]
		m.avail = append(m.avail, len(m.us))
		m.us = append(m.us, newUnit(protounit.S.SCopy(), len(m.us)))
	}

	//log.Printf("avail %+v/%d/%d\n", m.avail, len(m.us), m.cap)
	scMax := -(1 << 30)
	aiMax := -1
	uiMax := -1
	for i, a := range m.avail {
		u := m.us[a]
		sc := u.score(req.Ms)
		if sc > scMax {
			scMax = sc
			aiMax = i
			uiMax = a
		}
	}
	al := len(m.avail) - 1
	m.avail[aiMax], m.avail[al] = m.avail[al], m.avail[aiMax]
	m.avail = m.avail[:al]
	return m.us[uiMax]
}

func (m *ax) handleResp(resp *Response) {
	m.avail = append(m.avail, resp.Who)
	m.respChn <- resp // shouldn't block if client agrees to API
}

func (m *ax) shutdown() {
	for _, u := range m.us {
		if u != nil {
			u.cancel <- struct{}{}
		}
	}
}

type unit struct {
	I      int
	S      inter.S
	Solve  inter.Solve
	cancel chan struct{}
	Start  time.Time
	Pos    []int // Pos[var] gives count of supplied cubes with positive sign for var
	Neg    []int
}

func newUnit(s inter.S, i int) *unit {
	return &unit{
		I:      i,
		S:      s,
		Solve:  nil,
		cancel: make(chan struct{}),
		Pos:    make([]int, s.MaxVar()+1),
		Neg:    make([]int, s.MaxVar()+1)}
}

func (u *unit) handleReq(respChn chan<- *Response, req *Request) {
	u.Start = time.Now()
	// TBD: add trail size, new units in response.

	//log.Printf("start req %d by %d:%p\n", req.Id, u.I, u.S)

	u.S.Assume(req.Ms...)
	u.Solve = u.S.GoSolve()
	ticker := time.NewTicker(100 * time.Microsecond)
	var alarm <-chan time.Time
	if req.Limit != 0 {
		//log.Printf("limiting to %s\n", req.Limit)
		alarm = time.After(req.Limit)
	}
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-alarm: // alarm nil/blocks if not time limit.
				res, ok := u.Solve.Test()
				if !ok {
					u.Solve.Stop()
					u.Solve = nil
				}
				u.handleResult(req, res, respChn)
				return
			case <-ticker.C:
				res, ok := u.Solve.Test()
				if ok {
					u.handleResult(req, res, respChn)
					return
				}
			case <-u.cancel:
				//log.Printf("cancelling u %d\n", u.I)
				u.Solve.Stop()
				u.Solve = nil
				return
			}
		}
	}()
}

func (u *unit) handleResult(req *Request, res int, c chan<- *Response) {
	resp := &Response{
		Who: u.I,
		Req: req,
		Res: res,
		Dur: time.Since(u.Start)}
	// TBD units, model, etc
	u.Solve = nil

	// get model if available and requested
	if res == 1 && req.Flag.Model() {
		M := u.S.MaxVar()
		for i := z.Var(1); i <= M; i++ {
			m := i.Pos()
			if !u.S.Value(i.Pos()) {
				m = m.Not()
			}
			resp.Ms = append(resp.Ms, m)
		}
	}

	// get why if avaialable and requested
	if res == -1 && req.Flag.Why() {
		resp.Ms = u.S.Why(resp.Ms)
	}

	//log.Printf("end req %d by %d:%p\n", req.Id, u.I, u.S)
	c <- resp
}

func (u *unit) score(ms []z.Lit) int {
	if true {
		return 0
	}
	res := 0
	for _, m := range ms {
		v := m.Var()
		if m.IsPos() {
			res += u.Pos[v]
			res -= u.Neg[v]
			continue
		}
		res += u.Neg[v]
		res -= u.Pos[v]
	}
	return res
}
