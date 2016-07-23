// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package ax

import (
	"gini/z"
	"time"
)

// Type Response describes the result of an ax Request.
type Response struct {
	Req     *Request      // The request which was sent.
	Who     int           // which pool processing unit processed the request
	Res     int           // the result
	Dur     time.Duration // how long it took (wall clock time)
	Pending int           // how many solves were pending when this solve finished.
	Ms      []z.Lit       // data in response
}

// Pending returns whether the response is the last one
// of all requests sent so far to the ax.  It is ok
// for r to be nil.
func (r *Response) HasPending() bool {
	if r == nil {
		return false
	}
	return r.Pending == 0
}
