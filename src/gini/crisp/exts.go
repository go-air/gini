// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.
package crisp

import (
	"bytes"
	"fmt"
)

// Type Exts encapsulates installing and getting op codes
// for extensions
type Exts struct {
	D   []Extn
	Off uint32
}

// Type Extn encapsulates what client and server need
// to implement a given extension at runtime.
type Extn struct {
	Id  uint16
	N   uint8
	Off uint32
}

// String produces a pretty string for an installed extension
func (e *Extn) String() string {
	return fmt.Sprintf("extn-%d[%d@%d]", e.Id, e.N, e.Off)
}

// Op returns the op code installed on the server
func (e Extn) Op(o uint32) ProtoPoint {
	return ProtoPoint(o + e.Off)
}

// NewExts returns a new extension manager.
func NewExts() *Exts {
	return &Exts{
		D:   make([]Extn, 0, 256),
		Off: uint32(MinProto)}
}

// Install installs an extension into the Exts structure
// and returns an identifier for it local to the
// client or server using the structure.  It returns
// a non-nil error if the offset (total number of opcodes
// installed) overflows.  This should not exceed NumExtProtos
func (e *Exts) Install(id uint16, n uint8) (int, error) {
	m := uint32(n)
	if e.Off+m >= uint32(MinProto+NumExtProtos) {
		return -1, fmt.Errorf("too many opcodes installed")
	}
	res := len(e.D)
	x := Extn{
		Id:  id,
		N:   n,
		Off: e.Off}
	x.Off = uint32(e.Off)
	e.D = append(e.D, x)
	e.Off += uint32(n)
	return res, nil
}

// Op returns the op code for the o'th op code of extension with local id i
func (e *Exts) Op(i int, o uint32) ProtoPoint {
	return e.D[i].Op(o)
}

// Id Returns the persistent id of the extension with local id i.
func (e *Exts) Id(i int) uint16 {
	return e.D[i].Id
}

// String returns a string for exts
func (e *Exts) String() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[")
	for i, _ := range e.D {
		extn := &e.D[i]
		buf.WriteString(fmt.Sprintf("%s", extn))
		if i < len(e.D)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteString("]")
	return string(buf.Bytes())
}
