// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"fmt"
	"strings"
)

// Type Addr is a the type of a CRISP address
//
// CRISP addresses are either pathnames prefixed by
// '@' (for unix domain sockets) or tcp addresses,
// such as 'localhost:6060' or 'example.com:71'.
type Addr struct {
	Network string
	NetAddr string
}

// ParseAddr parses an address, determining whether is a
// unix socket or a tcp address.
func ParseAddr(s string) *Addr {
	if strings.HasPrefix(s, "@") {
		return &Addr{Network: "unix", NetAddr: s[1:]}
	}
	return &Addr{Network: "tcp", NetAddr: s}
}

// String puts the address back in the format provided
// to ParseAddr.
func (a *Addr) String() string {
	if a.Network == "@" {
		return fmt.Sprintf("@%s", a.NetAddr)
	}
	return fmt.Sprintf("%s", a.NetAddr)
}
