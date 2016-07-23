// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import "fmt"

type Version uint32

const (
	V = Version(1 << 23)
)

func (v Version) Major() int {
	return int(v >> 23)
}

func (v Version) Minor() int {
	return int(v & 0xfffff)
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d", v.Major(), v.Minor())
}
