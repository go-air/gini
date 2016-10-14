// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import "fmt"

type ProtoPoint uint32

const (
	Top          ProtoPoint = 0xffffffff
	MinProto                = Top - 255
	NumExtProtos            = Reset - MinProto
)

const (
	Key ProtoPoint = Top - iota
	Add
	Assume
	Solve
	Continue
	End
	Error
	Failed
	FailedFor
	Model
	ModelFor
	Sat
	Unsat
	Unknown
	Ext
	Quit
	Reset
)

// String gives human readable english representation of
// protocol points.
func (p ProtoPoint) String() string {
	switch p {
	case Key:
		return "<key>"
	case Add:
		return "<add>"
	case Assume:
		return "<assume>"
	case Solve:
		return "<solve>"
	case Continue:
		return "<continue>"
	case End:
		return "<end>"
	case Error:
		return "<error>"
	case Failed:
		return "<failed>"
	case FailedFor:
		return "<failedfor>"
	case Model:
		return "<model>"
	case ModelFor:
		return "<modelfor>"
	case Sat:
		return "<sat>"
	case Unsat:
		return "<unsat>"
	case Unknown:
		return "<unknown>"
	case Quit:
		return "<quit>"
	case Reset:
		return "<reset>"
	case Ext:
		return "<ext>"
	default:
		if p >= MinProto {
			return fmt.Sprintf("<ext-%d>", p-MinProto)
		}
		return fmt.Sprintf("<!data(%d)!>", uint32(p))
	}
}

func (p ProtoPoint) IsVarLen() bool {
	return p == Add
}
