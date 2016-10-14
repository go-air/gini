// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"fmt"
	"github.com/irifrance/g/z"
	"io"
)

// This file implements varint encoding/decoding + buffered io
// for uint32s.  We don't use encoding/binary varint because
//
// 1. encoding/binary is 64bit
// 2. encoding/binary keeps a separate buffer for each value, we code directly
// to the underlying buffer of the buffered io.
// 3. we are not safe for multiple goroutines (unliked buffered io)
// 4. encoding/binary's buffered io needs a separate buffer for reading and
// writing, we don't in this context.

type vu32io struct {
	rw   io.ReadWriter
	buf  []byte
	r, w int
}

const (
	VarUintMask = uint32((1 << 7) - 1)
)

func newVu32Io(rw io.ReadWriter) *vu32io {
	return &vu32io{
		rw:  rw,
		r:   0,
		w:   0,
		buf: make([]byte, 1024)}
}

func (io *vu32io) readu32() (uint32, error) {

	res := uint32(0)
	s := uint32(0)
	var b byte
	var e error

	for i := 0; i < 5; i++ {
		b, e = io.readByte()
		if e != nil {
			return 0, e
		}
		z := uint32(b) & VarUintMask

		res = res | (z << s)
		if b&(1<<7) == 0 {
			return res, nil
		}
		s += 7
	}
	return 0, ErrVarUint
}

func (io *vu32io) readu32Data() (uint32, error) {
	n, e := io.readu32()
	if e != nil {
		return 0, e
	}
	if n >= uint32(MinProto) {
		return n, ErrNotLit
	}
	return n, nil
}

func (io *vu32io) writeu32(d uint32) error {
	var b byte
	for {
		b = byte(d & VarUintMask)
		d = d >> 7
		if d > 0 {
			b |= (1 << 7)
			if e := io.writeByte(b); e != nil {
				return e
			}
			continue
		}
		if e := io.writeByte(b); e != nil {
			return e
		}
		return nil
	}
}

func (io *vu32io) send(ms []z.Lit) error {
	for _, m := range ms {
		if e := io.writeu32(uint32(m)); e != nil {
			return e
		}
	}
	return io.writeu32(uint32(0))
}

func (io *vu32io) recv(dst []z.Lit) ([]z.Lit, error) {
	for {
		d, e := io.readu32()
		if e != nil {
			return dst, e
		}
		if d == 0 {
			return dst, nil
		}
		if d >= uint32(MinProto) {
			return dst, ErrNotLit
		}
		dst = append(dst, z.Lit(d))
	}
}

func (io *vu32io) writeflush(u uint32) error {
	if e := io.writeu32(u); e != nil {
		return e
	}
	return io.flush()
}

func (io *vu32io) readByte() (byte, error) {
	if io.r >= io.w {
		if e := io.fill(); e != nil {
			return 0, e
		}
	}
	b := io.buf[io.r]
	io.r++
	return b, nil
}

func (io *vu32io) writeByte(b byte) error {
	if io.r >= len(io.buf) {
		if e := io.flush(); e != nil {
			return e
		}
	}
	io.buf[io.r] = b
	io.r++
	return nil
}

func (vio *vu32io) fill() error {
	if vio.r > vio.w {
		return fmt.Errorf("fill without write flush")
	}
	if vio.r == vio.w {
		vio.r = 0
		vio.w = 0
	}
	// shouldn't give both zero length read and nil error
	// but just in case we try a few times
	for i := 0; i < 10; i++ {
		nn, e := vio.rw.Read(vio.buf[vio.w:])
		vio.w += nn
		if nn > 0 {
			return nil
		}
		if e != nil {
			return e
		}
		//log.Printf("vu32io read without data %d/%d\n", vio.r, vio.w)
	}
	return fmt.Errorf("vu32io: too many reads without data\n")
}

func (io *vu32io) flush() error {
	n := io.r
	m := io.w
	k := 0
	for m < n {
		o, e := io.rw.Write(io.buf[m:n])
		m += o
		if o > 0 {
			k = 0
		}
		if e != nil {
			return e
		}
		k++
		if k >= 10 {
			return fmt.Errorf("repeated zero length no-error writes.")
		}
	}
	io.r = 0
	io.w = 0
	return nil
}
