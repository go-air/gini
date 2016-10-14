// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"fmt"
	"github.com/irifrance/g"
	"github.com/irifrance/g/inter"
	"github.com/irifrance/g/z"
	"log"
	"net"
	"os"
)

var trace = false

func init() {
	if os.Getenv("TRACE_CRISP") != "" {
		trace = true
	}
}

// Type Handler holds state for managing a single connection
type Handler struct {
	id        int
	conn      net.Conn
	uio       *vu32io
	gini      *gini.Gini
	solveConn inter.Solve
	proto     ProtoPoint
	ms        []z.Lit
	trace     *bool
	steps     int
}

func newHandler(i int, tr *bool) *Handler {
	h := &Handler{
		id:    i,
		proto: Unknown,
		ms:    make([]z.Lit, 0, 1024),
		trace: tr}
	if os.Getenv("TRACE_CRISP") != "" {
		*tr = true
	}
	return h
}

func (h *Handler) serve(cc <-chan net.Conn) {
	for {
		conn, ok := <-cc
		if !ok {
			// shutdown, channel closed
			return
		}
		if *h.trace {
			log.Printf("%d start serving %+v.\n", h.id, conn)
		}
		h.serveConn(conn)
		if *h.trace {
			log.Printf("%d done serving.\n", h.id)
		}
	}
}

func (h *Handler) serveConn(conn net.Conn) {
	h.conn = conn
	h.uio = newVu32Io(conn)
	h.gini = gini.New()
	h.proto = Unknown
	if e := h.Crisp(); e != nil {
		log.Printf("%d crisp() result error %s", h.id, e)
	}
	h.cleanup()
}

func (h *Handler) cleanup() {
	if *h.trace {
		log.Printf("%d cleanup\n", h.id)
	}
	h.conn.Close()
	h.uio = nil
	h.gini = nil
	h.ms = h.ms[:0]
	h.proto = Unknown
}

func (h *Handler) Crisp() error {
	if e := h.Hello(); e != nil {
		return e
	}
	uio := h.uio
	for {
		u, e := uio.readu32()
		if e != nil {
			return e
		}
		p := ProtoPoint(u)
		if *h.trace {
			log.Printf("%d received %s\n", h.id, p)
		}
		switch p {
		case Key:
			if e := h.handleKey(); e != nil {
				return e
			}
		case Add:
			if e := h.handleAdd(); e != nil {
				return e
			}
		case Assume:
			if e := h.handleAssume(); e != nil {
				return e
			}
		case Solve:
			if e := h.handleSolve(); e != nil {
				return e
			}
		case Continue:
			if e := h.handleContinue(); e != nil {
				return e
			}
		case End:
			if e := h.handleEnd(); e != nil {
				return e
			}
		case Model:
			if e := h.handleModel(); e != nil {
				return e
			}
		case ModelFor:
			if e := h.handleModelFor(); e != nil {
				return e
			}
		case Failed:
			if e := h.handleFailed(); e != nil {
				return e
			}
		case FailedFor:
			if e := h.handleFailedFor(); e != nil {
				return e
			}
		case Ext:
			if e := h.handleExt(); e != nil {
				return e
			}
		case Reset:
			if e := h.handleReset(); e != nil {
				return e
			}
		case Quit:
			return h.handleQuit()
		default:
			return h.Error(ErrUnknownOp)
		}
	}
}

func (h *Handler) Hello() error {
	s := "CRISP"
	for i := 0; i < len(s); i++ {
		if e := h.uio.writeu32(uint32(s[i])); e != nil {
			return e
		}
	}
	return h.uio.writeflush(uint32(V))
}

func (h *Handler) handleKey() error {
	h.proto = Key
	n, e := h.uio.readu32()
	if e != nil {
		return e
	}
	key := make([]uint32, n)
	for i := uint32(0); i < n; i++ {
		m, e := h.uio.readu32()
		if e != nil {
			return e
		}
		key[i] = m
	}
	// check the key
	return nil
}

func (h *Handler) handleReset() error {
	h.gini = gini.New()
	h.proto = Unknown
	return nil
}

func (h *Handler) handleExt() error {
	// no extensions yet
	return h.uio.writeflush(0)
}

func (h *Handler) handleAdd() error {
	h.proto = Add
	var m uint32
	var e error
	io := h.uio
	g := h.gini
	for {
		m, e = io.readu32()
		if e != nil {
			return e
		}
		if m == uint32(End) {
			if *h.trace {
				log.Printf("%d received %s\n", h.id, End)
			}
			return nil
		}
		if m >= uint32(MinProto) {
			return fmt.Errorf("CRISP protocol error expected data, got %s", ProtoPoint(m))
		}
		g.Add(z.Lit(m))
	}
}

func (h *Handler) handleAssume() error {
	h.proto = Assume
	var e error
	h.ms, e = h.uio.recv(h.ms)
	if e != nil {
		return e
	}
	for _, m := range h.ms {
		h.gini.Assume(m)
	}
	h.ms = h.ms[:0]
	return nil
}

func (h *Handler) handleSolve() error {
	h.proto = Solve
	h.solveConn = h.gini.GoSolve()
	h.steps = 0
	return h.handleStep()
}

func (h *Handler) handleContinue() error {
	if h.proto != Solve {
		log.Printf("continue but not solving.\n")
		return ErrOutOfOrder
	}
	return h.handleStep()
}

func (h *Handler) handleEnd() error {
	if h.proto != Solve {
		log.Printf("end but not solving\n")
		return ErrOutOfOrder
	}
	h.proto = End
	return h.sendRes(h.solveConn.Stop(), true)
}

func (h *Handler) handleStep() error {
	res, done := h.solveConn.Test()
	if res == 0 && done {
		panic("Test bombed")
	}
	h.steps++
	return h.sendRes(res, false)
}

func (h *Handler) sendRes(res int, end bool) error {
	switch res {
	case -1:
		if e := h.uio.writeflush(uint32(Unsat)); e != nil {
			return e
		}
		if *h.trace {
			log.Printf("%d sent %s\n", h.id, Unsat)
		}
		return nil
	case 1:
		if e := h.uio.writeflush(uint32(Sat)); e != nil {
			return e
		}
		if *h.trace {
			log.Printf("%d sent %s\n", h.id, Sat)
		}
		return nil
	case 0:
		op := uint32(Unknown)
		if end {
			op = uint32(End)
		}
		if e := h.uio.writeflush(op); e != nil {
			return e
		}
		if *h.trace {
			log.Printf("%d sent %s\n", h.id, Unknown)
		}
		return nil
	default:
		panic(fmt.Sprintf("unknown result %d", res))
	}
}

func (h *Handler) handleModel() error {
	if h.proto != Solve {
		log.Printf("model but not solving.\n")
		return h.Error(ErrOutOfOrder)
	}
	var u uint32
	M := uint32(h.gini.MaxVar())
	N := M
	if N%32 != 0 {
		N = N + (32 - (N % 32))
	}
	if N%32 != 0 {
		panic("fred")
	}
	if e := h.uio.writeu32(N / 32); e != nil {
		return e
	}
	writes := 0
	for i := uint32(0); i < N; i++ {
		j := i % 32
		if i < M && h.gini.Value(z.Var(i+1).Pos()) {
			u = u | (1 << j)
		}
		if j == uint32(31) {
			if e := h.uio.writeu32(u); e != nil {
				return e
			}
			writes++
			u = uint32(0)
		}
	}
	return h.uio.flush()
}

func (h *Handler) handleModelFor() error {
	if h.proto != Solve {
		return h.Error(ErrOutOfOrder)
	}
	ms, e := h.uio.recv(h.ms)
	if e != nil {
		return e
	}
	M := uint32(len(ms))
	N := M
	if N%32 != 0 {
		N = N + (32 - (N % 32))
	}
	if e := h.uio.writeu32(N / 32); e != nil {
		return e
	}
	var u uint32
	for i := uint32(0); i < N; i++ {
		j := i % 32
		if i < M && h.gini.Value(ms[i]) {
			u = u | (1 << j)
		}
		if j == uint32(31) {
			if e := h.uio.writeu32(u); e != nil {
				return e
			}
			u = uint32(0)
		}
	}
	return h.uio.flush()
}

func (h *Handler) handleFailed() error {
	if h.proto != Solve {
		log.Printf("failed but not solve\n")
		return h.Error(ErrOutOfOrder)
	}
	fails := h.gini.Why(h.ms)
	if e := h.uio.send(fails); e != nil {
		return e
	}
	return h.uio.flush()
}

func (h *Handler) handleFailedFor() error {
	if h.proto != Solve {
		return h.Error(ErrOutOfOrder)
	}
	fMap := make(map[z.Lit]struct{})
	h.ms = h.ms[:0]
	fs := h.gini.Why(h.ms)
	for _, m := range fs {
		fMap[m] = struct{}{}
	}

	h.ms = h.ms[:0]
	ms, e := h.uio.recv(h.ms)
	if e != nil {
		return e
	}

	j := 0
	for _, m := range ms {
		_, ok := fMap[m]
		if ok {
			ms[j] = m
			j++
		}
	}
	ms = ms[:j]

	if e := h.uio.send(ms); e != nil {
		return e
	}
	return h.uio.flush()
}

func (h *Handler) handleQuit() error {
	return nil
}

func (h *Handler) Error(e ProtoErr) error {
	if *h.trace {
		log.Printf("%d <error> %s\n", h.id, e)
	}
	if e := h.uio.writeu32(uint32(Error)); e != nil {
		return e
	}
	if e := h.uio.writeflush(uint32(e)); e != nil {
		return e
	}
	h.cleanup()
	return e
}
