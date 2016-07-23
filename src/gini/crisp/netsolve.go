// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import "time"

type netsolve struct {
	cancel chan<- struct{}
	result <-chan result
}

// Test checks whether or not a result is ready, and if so returns it
// together with true and a nil error.  If not, it returns (0, false, nil),
// unless there is an os/network/or protocol error in which the error is non-nil
func (s *netsolve) Test() (int, bool, error) {
	select {
	case res := <-s.result:
		return res.v, true, res.e
	default:
		return 0, false, nil
	}
}

// Try lets Solve() run for at most d time and then returns the result.
// the result should be ignored if a non-nil error is returned.
func (s *netsolve) Try(d time.Duration) (int, error) {
	a := time.After(d)
	select {
	case <-a:
		return s.Stop()
	case res := <-s.result:
		return res.v, res.e
	}
}

// Stop stops the underlying Solve() call and returns the result.
// the result should be ignored if a non-nil error is returned.
func (s *netsolve) Stop() (int, error) {
	select {
	case res := <-s.result:
		s.cancel <- struct{}{}
		return res.v, res.e
	case s.cancel <- struct{}{}:
		r := <-s.result
		return r.v, r.e
	}
}
