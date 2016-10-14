// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

type ProtoErr uint32

const (
	ErrVarUint ProtoErr = 1 + iota
	ErrNotSat
	ErrNotUnsat
	ErrOutOfOrder
	ErrUnknownOp
	ErrNotLit
	ErrNotOp
	ErrInternal
)

func (pe ProtoErr) String() string {
	switch pe {
	case ErrVarUint:
		return "varuint32 encoding overflow"
	case ErrNotSat:
		return "last solve wasn't SAT"
	case ErrNotUnsat:
		return "last solve wasn't UNSAT"
	case ErrOutOfOrder:
		return "out of order request"
	case ErrUnknownOp:
		return "unknown operator"
	case ErrNotLit:
		return "not a literal"
	case ErrNotOp:
		return "not an opcode"
	case ErrInternal:
		return "internal server error"
	default:
		return "unknown error"
	}
}

func (pe ProtoErr) Error() string {
	return pe.String()
}
