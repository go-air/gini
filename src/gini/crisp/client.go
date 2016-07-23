// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"fmt"
	"gini/dimacs"
	gnet "gini/inter/net"
	"gini/internal/xo"
	"gini/z"
	"io"
	"log"
	"net"
)

// Type Client is a concrete Solver which communicates using the gini sat protocol.
type Client struct {
	conn    net.Conn
	cancel  chan struct{}
	resChan chan result

	io      *vu32io
	proto   ProtoPoint
	lastSat int
	maxVar  z.Var
	exts    *Exts
	model   []bool
}

type result struct {
	v int
	e error
}

// Dial creates a new Gini protocol client, implementing
// an error-augmented version the Solver interface, for reporting
// underlying protocol/network errors.
func Dial(addr string) (*Client, error) {
	a := ParseAddr(addr)
	conn, err := net.Dial(a.Network, a.NetAddr)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:    conn,
		cancel:  make(chan struct{}),
		resChan: make(chan result),
		io:      newVu32Io(conn),
		proto:   Unknown,
		lastSat: 0}
	if e := c.Hello(); e != nil {
		return nil, e
	}
	return c, nil
}

// Hello reads a greating from a CRISP server.
func (c *Client) Hello() error {
	hdr := make([]byte, 5)
	for i := 0; i < 5; i++ {
		u, e := c.io.readu32()
		if e != nil {
			return e
		}
		hdr[i] = byte(u)
	}
	if string(hdr) != "CRISP" {
		return fmt.Errorf("not a CRISP server: '%s'", string(hdr))
	}
	v, e := c.io.readu32()
	if e != nil {
		return e
	}
	if trace {
		log.Printf("connected to CRISP-%s\n", Version(v))
	}
	return nil
}

// Dimacs takes a dimacs formatted file and loads it
// to the server, returning an error if there was
// a problem
func (c *Client) Dimacs(r io.Reader) error {
	vis := &xo.DimacsVis{}
	if e := dimacs.ReadCnf(r, vis); e != nil {
		return e
	}
	return c.fromXo(vis.S())
}

// TBD: make this stream/buffer the adds instead
// of loading into client memory.
func (c *Client) fromXo(x *xo.S) error {
	cdb := x.Cdb
	D := cdb.CDat.D
	for _, p := range cdb.Added {
		for {
			m := D[p]
			if e := c.Add(z.Lit(m)); e != nil {
				return e
			}
			if m == z.LitNull {
				break
			}
			p++
		}
	}
	return nil
}

// Add adds a literal to the clause constraints in the solver.
// To terminate the clause, m should 0.
func (c *Client) Add(m z.Lit) error {
	if c.proto != Add {
		c.checkState(Add)
	}
	if m.Var() > c.maxVar {
		c.maxVar = m.Var()
	}
	return c.io.writeu32(uint32(m))
}

// Assume adds an assumption to the solver for the next call
// to Solver()
func (c *Client) Assume(ms ...z.Lit) error {
	if len(ms) == 0 {
		return nil
	}
	c.checkState(Assume)
	if e := c.io.send(ms); e != nil {
		return e
	}
	for _, m := range ms {
		if m.Var() > c.maxVar {
			c.maxVar = m.Var()
		}
	}
	return c.io.flush()
}

// MaxVar returns the max var which has been Added or Assumed.
func (c *Client) MaxVar() z.Var {
	return c.maxVar
}

// Solve attempts to solve the current set of constraints under
// the current assumptions.  Solve returns
//
//  1  if the problem is satisfiable
//
//  0  if the the problem was not solved due to some error or
//     if the server asked to stop.
//
//  -1 if the problem is unsatisfiable
func (c *Client) Solve() (int, error) {
	if e := c.checkState(Solve); e != nil {
		return 0, e
	}

	for {
		sentEnd := false
	ReadOp:
		op, err := c.io.readu32()
		// TBD: better way to map errors in protocol to Solver API
		if err != nil && err != io.EOF {
			return 0, err
		}
		switch ProtoPoint(op) {
		case 0:
			if err != nil {
				return 0, err
			}
		case Sat:
			c.lastSat = 1
			return 1, nil
		case Unsat:
			c.lastSat = -1
			return -1, nil
		case End:
			c.lastSat = 0
			return 0, nil
		case Unknown:
			if sentEnd {
				return 0, ErrOutOfOrder
			}
			select {
			case <-c.cancel:
				if e := c.io.writeflush(uint32(End)); e != nil {
					return 0, e
				}
				sentEnd = true
				goto ReadOp
			default:
				if e := c.io.writeflush(uint32(Continue)); e != nil {
					return 0, e
				}
			}
		default:
			if op >= uint32(MinProto) {
				return 0, ErrUnknownOp
			}
			return 0, ErrNotOp
		}
	}
	return 0, nil
}

// GoSolve returns a connection to a solve proces
// in the background.
func (c *Client) GoSolve() gnet.Solve {
	as := &netsolve{
		cancel: c.cancel,
		result: c.resChan}
	go func() {
		v, e := c.Solve()
		c.resChan <- result{v: v, e: e}
	}()
	return as
}

// Model returns a model of the formula if the last Solve()
// was satisfiable.
func (c *Client) Model(vs []bool) ([]bool, error) {
	if c.lastSat != 1 {
		return nil, fmt.Errorf("last solve call wasn't sat")
	}
	if e := c.checkState(Model); e != nil {
		return nil, nil
	}
	var u uint32
	var n uint32
	var e error
	n, e = c.io.readu32Data()
	if e != nil {
		return vs, e
	}
	theCap := n*32 + 1
	if cap(vs) < int(theCap) {
		vs = make([]bool, theCap)
	} else {
		vs = vs[:theCap]
	}
	for i := uint32(0); i < theCap-1; i++ {
		j := i % 32
		if j == 0 {
			u, e = c.io.readu32()
			if e != nil {
				return vs, e
			}
		}
		vs[i+1] = u&(1<<j) != 0
	}
	c.model = vs
	return vs, nil
}

// Value implements the inter.NetModel interface
func (c *Client) Value(m z.Lit) (bool, error) {
	if c.model == nil {
		m, e := c.Model(nil)
		if e != nil {
			return false, e
		}
		c.model = m
	}
	if m.IsPos() {
		return c.model[m.Var()], nil
	}
	return !c.model[m.Var()], nil
}

// ModelFor returns a partial model of the
func (c *Client) ModelFor(dst []bool, ms []z.Lit) ([]bool, error) {
	if c.lastSat != 1 {
		return nil, fmt.Errorf("last solve call wasn't sat")
	}
	if e := c.checkState(ModelFor); e != nil {
		return nil, nil
	}
	if e := c.io.send(ms); e != nil {
		return nil, e
	}
	if e := c.io.flush(); e != nil {
		return nil, e
	}
	var u uint32
	var n uint32
	var e error
	n, e = c.io.readu32Data()
	if e != nil {
		return nil, e
	}
	theCap := n*32 + 1
	if cap(dst) < int(theCap) {
		dst = make([]bool, theCap)
	} else {
		dst = dst[:theCap+1]
	}
	for i := uint32(0); i < theCap-1; i++ {
		j := i % 32
		if j == 0 {
			u, e = c.io.readu32()
			if e != nil {
				return nil, e
			}
		}
		dst[i+1] = u&(1<<j) != 0
	}
	return dst, nil
}

// Why retrieves a list of failed assumptions from the server
// and places them in ms.
func (c *Client) Why(ms []z.Lit) ([]z.Lit, error) {
	if c.lastSat != -1 {
		return nil, fmt.Errorf("last solve call wasn't unsat")
	}
	if e := c.checkState(Failed); e != nil {
		return ms, nil
	}
	return c.io.recv(ms)
}

// WhySelect retrieves a sub-list of failed assumptions from
// the server selected from ms and placed in dst.  The
// literals in dst respect the order of the literals in ms.
func (c *Client) WhySelect(dst []z.Lit, ms []z.Lit) ([]z.Lit, error) {
	if c.lastSat != -1 {
		return nil, fmt.Errorf("last solve call wasn't unsat")
	}
	if e := c.checkState(FailedFor); e != nil {
		return dst, nil
	}
	if e := c.io.send(ms); e != nil {
		return nil, e
	}
	if e := c.io.flush(); e != nil {
		return nil, e
	}
	return c.io.recv(dst)
}

// GetExts queries the server for extensions.  Each
// extension has a unique id and a number of op codes
// GetExts returns the number of extensions retrieved.
//
func (c *Client) GetExts() (int, error) {
	if e := c.checkState(Ext); e != nil {
		return 0, e
	}
	if c.exts != nil {
		c.exts = nil
	}
	exts := NewExts()
	for {
		d, e := c.io.readu32()
		if e != nil {
			return 0, e
		}
		if d == 0 {
			break
		}
		exts.Install(uint16(d>>16), uint8(d&0xff))
	}
	c.exts = exts
	return len(c.exts.D), nil
}

// ExtOp returns the runtime opcode of the o'th opcode of extension with
// runtime id i.
func (c *Client) ExtOp(i int, o uint32) ProtoPoint {
	return c.exts.Op(i, o)
}

// Reset requests the server forget all added clauses
// and assumptions and continue as if we just connected.
func (c *Client) Reset() error {
	if e := c.checkState(Reset); e != nil {
		return e
	}
	c.proto = Unknown
	return nil
}

// Quit signals to the server that no further
// operations will take place.
func (c *Client) Quit() error {
	return c.checkState(Quit)
}

// Close closes the connection to the server.
func (c *Client) Close() error {
	return c.conn.Close()
}

// check's whether we're in the correct state
// and sends op codes accordingly.
func (c *Client) checkState(point ProtoPoint) error {
	if c.proto.IsVarLen() {
		if e := c.io.writeflush(uint32(End)); e != nil {
			return e
		}
	}
	var err error
	switch point {
	case Key:
		err = c.io.writeu32(uint32(Key))
		c.proto = Key
	case Add:
		err = c.io.writeu32(uint32(Add))
		c.proto = Add
	case Assume:
		err = c.io.writeu32(uint32(Assume))
		c.proto = Assume
	case Solve:
		err = c.io.writeflush(uint32(Solve))
		c.proto = Solve
	case Model:
		err = c.io.writeflush(uint32(Model))
		c.proto = Model
	case ModelFor:
		err = c.io.writeu32(uint32(ModelFor))
		c.proto = ModelFor
	case Failed:
		err = c.io.writeflush(uint32(Failed))
		c.proto = Failed
	case FailedFor:
		err = c.io.writeu32(uint32(FailedFor))
		c.proto = FailedFor
	case Quit:
		err = c.io.writeflush(uint32(Quit))
		c.proto = Quit
	case Reset:
		err = c.io.writeflush(uint32(Reset))
		c.proto = Reset
	case Ext:
		err = c.io.writeflush(uint32(Ext))
		c.proto = Ext
	default:
		panic("unexpected client state in Gini protocol")
	}
	return err
}
