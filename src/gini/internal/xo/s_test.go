// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package xo

import (
	"gini/gen"
	"gini/z"
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestSRand3Cnf(t *testing.T) {
	//s := genRand3Cnf(400, 1575)
	s := NewS()
	gen.Rand3Cnf(s, 300, 1206)
	r := s.Solve()
	_ = r
}

func TestSPhp(t *testing.T) {
	for p := 4; p < 9; p++ {
		for _, d := range [...]int{-2, -1, 0, 1, 2} {
			h := p + d
			s := NewS()
			gen.Php(s, p, h)
			r := s.Solve()
			if h >= p && r != 1 {
				t.Errorf("php %d/%d not sat", p, p+d)
			}
			if h < p && r != -1 {
				t.Errorf("php %d/%d not unsat", p, p+d)
			}
			//log.Printf("php %d/%d %d (%d props) %s\n", p, h, r, s.SolveStats.Props, s.Driver)
		}
	}
}

func TestSAssume(t *testing.T) {
	N := 10
	s := NewS()
	gen.BinCycle(s, 100)
	for i := 0; i < N; i++ {
		u := z.Var(((i + 4) % N) + 1)
		v := z.Var(((i + 1) % N) + 1)
		if i%2 == 0 {
			// assume some var and negation of another: unsat
			s.Assume(u.Pos())
			s.Assume(v.Neg())

			if s.Solve() == 1 {
				t.Errorf("sat[%s,%s] shouldn't be\n", u.Pos(), v.Neg())
			}
			y := s.Why(nil)
			if len(y) != 2 {
				t.Errorf("why wrong: %s\n", y)
			}
			continue
		}
		s.Assume(u.Pos())
		if s.Solve() == -1 {
			t.Errorf("unsat shouldn't be\n")
		}
		y := s.Why(nil)
		if len(y) != 0 {
			t.Errorf("call was sat, but Why returned %s\n", y)
		}
	}
}

func TestSAddEmpty(t *testing.T) {
	s := NewS()
	s.Add(z.LitNull)
	s.Add(z.Lit(17))
	s.Add(z.LitNull)
	if s.Solve() != -1 {
		t.Errorf("sat on add empty\n")
	}
	s.Assume(z.Lit(4))
	if s.Solve() != -1 {
		t.Errorf("sat on add empty under assumption\n")
	} else {
		if len(s.Why(nil)) != 0 {
			t.Errorf("why not empty after add empty\n")
		}
	}
}

func TestSGrow(t *testing.T) {
	s := NewSV(10)
	s.Add(z.Lit(20))
	s.Add(z.Lit(50))
	s.Add(z.Lit(150))
	s.Add(z.LitNull)
	if s.Solve() != 1 {
		t.Errorf("not sat on grow")
	}
}

func TestSTimeout(t *testing.T) {
	s := NewS()
	gen.Rand3Cnf(s, 3000, 12000)
	r := s.GoSolve().Try(640 * time.Millisecond)
	if r != 0 {
		t.Errorf("didn't timeout\n")
	}
	//log.Printf("%d props (%s)\n", s.SolveStats.Props, s.Driver)
}

// test until conflict, go back
func TestSTest(t *testing.T) {
	N := 100
	s := NewS()
	gen.Rand3Cnf(s, N, 400)
	props := make([]z.Lit, 0, 128)
	res := 0
	tests := 0
	breakpoint := 0
	for res == 0 {
		res, props = s.Test(props)
		tests++
		if res == 1 {
			for tests > 0 {
				if r := s.Untest(); r == -1 {
					t.Errorf("untest gave unsat on sat problem")
				}
				tests--
			}

			return
		}
		if res == -1 {
			log.Printf("assumed randomly %d times before unsat\n", tests-1)
			breakpoint = tests
			break
		}
		for k := 0; k < 2; k++ {
			v := z.Var(rand.Intn(100) + 1)
			if rand.Intn(2) == 1 {
				s.Assume(v.Pos())
			} else {
				s.Assume(v.Neg())
			}
		}
	}
	tests--
	for tests > 0 && s.Untest() == -1 {
		tests--
	}
	log.Printf("back to %d breakpoint %d\n", tests, breakpoint)
}

// test,untest,untest panics
func TestSTestUnTest(t *testing.T) {
	s := NewS()
	gen.Rand3Cnf(s, 10, 40)
	s.Test(nil)
	if s.Untest() != 0 {
		t.Errorf("untest bad result\n")
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("didn't panic on untest.")
		} else {
			//log.Printf("ok panicked as expected\n")
		}
	}()
	s.Untest()
}

// test creates sat and checks reasons
func TestSTestBin(t *testing.T) {
	N := 10
	s := NewS()
	gen.BinCycle(s, N)
	props := make([]z.Lit, 0)
	res := 0
	res, props = s.Test(props)
	if res != 0 {
		t.Errorf("unexpected %d\n", res)
	}
	if len(props) != 0 {
		t.Errorf("unexpected propagations: %+v\n", props)
	}
	s.Assume(z.Var(4).Pos())
	res, props = s.Test(props)
	if res != 1 {
		t.Errorf("should be sat, got %d\n", res)
	}
	if len(props) != N {
		t.Errorf("bad props: %+v\n", props)
	}
	reasons := make([]z.Lit, 0)
	for _, prop := range props {
		reasons = s.Reasons(reasons, prop)
		if prop == z.Var(4).Pos() {
			if len(reasons) != 0 {
				t.Errorf("should be decision, got %+v reasons\n", reasons)
			}
			continue
		}
		if len(reasons) != 1 {
			t.Errorf("wrong length of reasons: %+v\n", reasons)
		}
		if !reasons[0].IsPos() {
			t.Errorf("negative reason %+v => %s\n", reasons, prop)
		}
	}
}

// interleaves solve with test,untest.

// an unusual sequence of solve,test,untest
func TestSTestTestUntestSolve(t *testing.T) {
	s := NewS()
	gen.Php(s, 6, 6)
	r := s.Solve()
	if r != 1 {
		t.Fatalf("php")
	}
	t1 := s.Value(z.Var(1).Pos())
	t2 := s.Value(z.Var(4).Pos())

	r, _ = s.Test(nil)
	if r != 0 {
		t.Fatalf("test@0 gave %d\n", r)
	}

	if t1 {
		s.Assume(z.Var(1).Pos())
	} else {
		s.Assume(z.Var(1).Neg())
	}
	r, _ = s.Test(nil)
	if r != 0 {
		t.Fatalf("test@1 gave %d\n", r)
	}
	if t2 {
		s.Assume(z.Var(2).Pos())
	} else {
		s.Assume(z.Var(2).Neg())
	}
	r, _ = s.Test(nil)
	if r != 0 {
		t.Fatalf("test@2 gave %d\n", r)
	}

	r = s.Solve()
	if r != 1 {
		t.Fatalf("solve not sat %d\n", r)
	}
	if s.Untest() == -1 {
		t.Fatalf("untest gave -1, shouldn't have\n")
		return
	}
	r = s.Solve()
	if r != 1 {
		t.Fatalf("untest solve not sat %d\n", r)
	}
}

// interleaves async solve with test,untest
func TestSTestSolve(t *testing.T) {
	testSolverTestSolve(t, func(s *S) int {
		return s.Solve()
	})
}

// interleaves async solve with test,untest
func TestSTestGoSolve(t *testing.T) {
	testSolverTestSolve(t, func(s *S) int {
		return s.GoSolve().Try(time.Second)
	})
}

func testSolverTestSolve(t *testing.T, sFunc func(s *S) int) {
	N := 50
	s := NewS()
	gen.Rand3Cnf(s, N, N*4+34)

	r, _ := s.Test(nil)
	if r == -1 {
		return
	}
	for i := 0; i < 10; i++ {
		s.Assume(z.Var(i + 1).Neg())
		s.Assume(z.Var(i + 2).Pos())
		//log.Printf("assumed %s %s", z.Var(i+1).Neg(), z.Var(i+2).Pos())
		res, _ := s.Test(nil)
		if res == 0 {
			//log.Printf("solving\n")
			if rand.Intn(3) == 2 {
				s.Assume(z.Var(i + 3).Neg())
				//log.Printf("assuming untested %s\n", z.Var(i+3).Neg())
			}
			r = sFunc(s)
		} else {
			//log.Printf("no need to solve\n")
			r = res
		}
		if r == 0 {
			t.Errorf("couldn't solve easy problem\n")
			break
		}
		//log.Printf("got %d\n", r)
		u := s.Untest()
		//log.Printf("untest gave %d\n", u)
		if u == -1 {
			if s.Untest() == -1 {
				log.Printf("unsat irrespective of assumptions\n")
				return
			}
		}
	}
	s.Untest() // level 0 test
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("untest too many times didn't panic")
		}
	}()
	s.Untest() // too many, panics
}

func TestSBinNew(t *testing.T) {
	N := 10
	s := NewS()
	for i := 1; i <= N; i++ {
		s.Add(z.Var(i).Neg())
		if i < N {
			s.Add(z.Var(i + 1).Pos())
		} else {
			s.Add(z.Var(1).Pos())
		}
		s.Add(0)
	}
	log.Printf("%s\n", s.Solve())
}

func TestSGrowRand(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("error with growing\n")
		}
	}()
	s := NewS()
	for i := 0; i < 512; i++ {
		v := z.Var(rand.Intn(16384 * 2 * 2 * 2 * 2))
		s.Add(v.Pos())
		s.Add(0)
	}
	if r := s.Solve(); r != 1 {
		t.Errorf("rand grow solve: %d\n", r)
	}
}

func TestCopyPause(t *testing.T) {
	for i := 0; i < 10; i++ {
		s0 := NewS()
		gen.HardRand3Cnf(s0, 150)
		ps := s0.GoSolve()
		r, ok := ps.Pause()
		if !ok {
			gen.Seed(int64(i + 127))
			log.Printf("got %d, didn't pause\n", r)
			continue
		}
		s1 := s0.Copy()
		pt := s1.GoSolve()
		ps.Unpause()
		rs := ps.Wait()
		rt := pt.Wait()
		if rs != rt {
			t.Errorf("mismatch result on copy'd solver: %d, %d\n", rs, rt)
		}
		return
	}
	t.Errorf("giving up, couldn't pause\n")
}
