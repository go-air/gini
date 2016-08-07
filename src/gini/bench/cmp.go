// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package bench

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// SolveRate gives the rate in terms of solutions found
// per second for the run r.  It returns the results in
// terms of real time, user time, and system time.
//
// SolveRate is only valid w.r.t. a instance/global timeout
// because unsolved instances have unknown time.  SolveRate
// counts unsolved time.
func SolveRate(r *Run, unit time.Duration) (real float64, user float64, sys float64) {
	dur := time.Duration(0)
	uDur := time.Duration(0)
	sDur := time.Duration(0)
	ttl := 0
	for _, ir := range r.InstRuns {
		dur += ir.Dur
		uDur += ir.UDur
		sDur += ir.SDur
		if ir.Result == 0 {
			continue
		}
		ttl++
	}
	real = float64(ttl) / (float64(int64(dur)) / float64(int64(unit)))
	user = float64(ttl) / (float64(int64(uDur)) / float64(int64(unit)))
	sys = float64(ttl) / (float64(int64(sDur)) / float64(int64(unit)))
	return
}

func TotalResult(r *Run, filt func(r int) bool) int {
	ttl := 0
	for _, ir := range r.InstRuns {
		if filt(ir.Result) {
			ttl++
		}
	}
	return ttl
}

// SolveTotal gives the total number of solved instances
// for the run r.
func SolveTotal(r *Run) int {
	return TotalResult(r, func(r int) bool { return r != 0 })
}

func SatTotal(r *Run) int {
	return TotalResult(r, func(r int) bool { return r == 1 })
}

func UnsatTotal(r *Run) int {
	return TotalResult(r, func(r int) bool { return r == -1 })
}

func UnknownTotal(r *Run) int {
	return TotalResult(r, func(r int) bool { return r == 0 })
}

func Times(r *Run) (real float64, user float64, sys float64) {
	sec := float64(time.Second)
	for _, ir := range r.InstRuns {
		real += float64(ir.Dur) / sec
		user += float64(ir.UDur) / sec
		real += float64(ir.SDur) / sec
	}
	return
}

// SolvePortion gives the portion of instances in r solved.
func SolvePortion(r *Run) float64 {
	ttl := float64(SolveTotal(r))
	return ttl / float64(len(r.InstRuns))
}

// Type Scatter contains info necessary for a scatter plot
// of 2 runs.
type Scatter struct {
	Runs [2]*Run
	Xs   []time.Duration // Xs[i], Ys[i] gives point for run 0,1 on instance i.
	Ys   []time.Duration
}

// NewScatter creates a new Scatter object for a pair of runs r1, r2.
// iFilt, if non-nil, selects the instances to show in the scatter.
func NewScatter(r1, r2 *Run, iFilt func(suite *Suite, i int) bool) *Scatter {
	if r1.Suite != r2.Suite {
		panic(fmt.Sprintf("cannot scatter plot across suites %s != %s\n", r1.Suite.Root, r2.Suite.Root))
	}
	s := &Scatter{}
	s.Runs[0] = r1
	s.Runs[1] = r2
	s.Xs = make([]time.Duration, 0, len(r1.InstRuns))
	for _, ir := range r1.InstRuns {
		if iFilt == nil || iFilt(ir.Run.Suite, ir.Inst) {
			s.Xs = append(s.Xs, ir.Dur)
		}
	}
	s.Ys = make([]time.Duration, 0, len(r1.InstRuns))
	for _, ir := range r2.InstRuns {
		if iFilt == nil || iFilt(ir.Run.Suite, ir.Inst) {
			s.Ys = append(s.Ys, ir.Dur)
		}
	}
	return s
}

// Utf8 returns an ascii scatter plot with 2n columns and n rows,
// making up for the width/height ratio of most monospaced fonts.
func (s *Scatter) Utf8(n int) string {
	M := 2 * n * (n + 1)
	buf := make([]byte, M)
	for i, _ := range buf {
		buf[i] = byte(' ')
	}
	for i := 2 * n; i < M; i += 2*n + 1 {
		buf[i] = byte('\n')
	}
	var idx = func(x, y int) int {
		return (n-1)*(2*n+1) - (y * (2*n + 1)) + 2*x
	}
	for i := 0; i < n; i++ {
		buf[idx(i, i)] = byte('/')
	}

	maxDur := time.Duration(0)
	for _, d := range s.Xs {
		if d > maxDur {
			maxDur = d
		}
	}
	for _, d := range s.Ys {
		if d > maxDur {
			maxDur = d
		}
	}
	for i, xd := range s.Xs {
		yd := s.Ys[i]
		xr := float64(xd) / float64(maxDur)
		yr := float64(yd) / float64(maxDur)
		xi := int(xr * float64(n-1))
		yi := int(yr * float64(n-1))
		j := idx(xi, yi)
		if xi < yi {
			buf[j] = byte('+')
		} else if xi > yi {
			buf[j] = byte('-')
		} else if rand.Intn(2) == 1 {
			buf[j] = byte('+')
		} else {
			buf[j] = byte('-')
		}
	}
	res := strings.Replace(string(buf), "+", "★", -1)
	res = strings.Replace(res, "-", "☆", -1)
	lines := strings.Split(res, "\n")
	lines = append(lines, strings.Repeat("-", 2*n))
	legend := fmt.Sprintf("\t%s - %s wins\n\t%s - %s wins\n", "★", s.Runs[0].Name,
		"☆", s.Runs[1].Name)

	return fmt.Sprintf("%s\n%s", strings.Join(lines, "\n"), legend)
}

// Type Cactus contains info necessary for cactus plot
// of an arbitrary number of runs.
type Cactus struct {
	Runs   []*Run  // runs to plot.
	D      [][]int // D[i] permutation for run i instances sorted according to dur.
	MaxDur time.Duration
}

// NewCactus makes a new cactus object. filt, if non-nil, selects
// the instances to show in the cactus plot.
func NewCactus(suite *Suite, filt func(*Suite, int) bool) *Cactus {
	maxDur := time.Duration(0)
	for _, run := range suite.Runs {
		for _, irun := range run.InstRuns {
			if filt != nil && !filt(suite, irun.Inst) {
				continue
			}
			if irun.Dur > maxDur {
				maxDur = irun.Dur
			}
		}
	}

	cactus := &Cactus{
		MaxDur: maxDur,
		Runs:   suite.Runs}
	M := len(suite.Runs[0].InstRuns)
	for _, run := range suite.Runs {
		js := make([]int, 0, M)
		for j := 0; j < M; j++ {
			if filt == nil || filt(suite, j) {
				js = append(js, j)
			}
		}
		cactus.D = append(cactus.D, js)
		s := &irSort{Perm: js, IRuns: run.InstRuns}
		sort.Sort(s)
	}
	return cactus
}

type irSort struct {
	Perm  []int
	IRuns []*InstRun
}

func (s *irSort) Len() int {
	return len(s.Perm)
}

func (s *irSort) Swap(i, j int) {
	s.Perm[i], s.Perm[j] = s.Perm[j], s.Perm[i]
}

func (s *irSort) Less(i, j int) bool {
	return s.IRuns[s.Perm[i]].Dur < s.IRuns[s.Perm[j]].Dur
}

// Utf8 produces a text image of the cactus data suitable
// for a utf8 monospaced font terminal.
func (c *Cactus) Utf8(N int) string {
	ticks := "¤♠☆Ϟ★Ω▽◇✠♡☼·₁₂₃₄₅₆₇₈₉"
	_ = ticks
	M := N * (2*N + 1)
	buf := make([]byte, M)
	for i, _ := range buf {
		buf[i] = byte(' ')
	}
	for i := 2 * N; i < M; i += 2*N + 1 {
		buf[i] = byte('\n')
	}
	var idx = func(x, y int) int {
		return (N-1)*(2*N+1) - (y * (2*N + 1)) + 2*x
	}
	for ri, runPerm := range c.D {
		run := c.Runs[ri]
		tick := byte('0') + byte(ri) // change to unicode later for indexing.
		jDen := float64(len(runPerm))
		for j, p := range runPerm {
			jRatio := float64(j) / jDen
			dj := int(jRatio * float64(N-1))
			dur := run.InstRuns[p].Dur
			ratio := float64(dur) / float64(c.MaxDur)
			di := int(ratio * float64(N-1))
			bidx := idx(dj, di)
			if buf[bidx] == byte(' ') {
				buf[bidx] = tick
			} else if rand.Intn(len(c.D)) == ri {
				buf[bidx] = tick
			}
		}
	}
	mds := fmt.Sprintf("%.2fs", float64(c.MaxDur)/float64(time.Second))
	pad := strings.Repeat(" ", len(mds))
	prefix := strings.Repeat(" ", len(mds)+1)
	s := string(buf)
	j := 0
	legend := make([]string, 0, len(c.D))
	for _, r := range ticks {
		s = strings.Replace(s, fmt.Sprintf("%c", byte('0')+byte(j)), string(r), -1)
		legend = append(legend, fmt.Sprintf("%s\t%s - %s\n", prefix, string(r), c.Runs[j].Name))
		j++
		if j >= len(c.D) {
			break
		}
	}
	lines := strings.Split(s, "\n")
	pLines := make([]string, len(lines))
	for i, ln := range lines {
		pLines[i] = fmt.Sprintf("%s|%s", pad, ln)
	}
	pLines[0] = fmt.Sprintf("%s|%s", mds, lines[0])
	pLines[len(pLines)-1] = fmt.Sprintf("%s0s|%s", strings.Repeat(" ", len(mds)-2), lines[len(lines)-1])
	s = strings.Join(pLines, "\n")

	delim := strings.Repeat("-", 2*N)
	sx := fmt.Sprintf("0%s%-5d", strings.Repeat(" ", 2*N+1-5), len(c.D[0]))
	return fmt.Sprintf("%s%s\n%s%s\n%s", s, delim, prefix, sx, strings.Join(legend, ""))
}

// Summary produces a summary of all runs in the Suite s.
func Summary(s *Suite) string {
	hdr := `
Suite %s
-----------------------------------------------------------------------------------------------------------
| Run                  | solved   | sat      | unsat     | unknown |  time      | utime      | stime      |
-----------------------------------------------------------------------------------------------------------`
	rSum := `| %-16s     | %-4d     | %-4d     | %-4d      | %-4d    |  %-7.2fs  | %-7.2fs   | %-7.2fs   |
-----------------------------------------------------------------------------------------------------------`
	_, nm := filepath.Split(s.Root)
	parts := make([]string, 0, len(s.Runs))
	parts = append(parts, fmt.Sprintf(hdr, nm))

	for _, r := range s.Runs {
		real, user, sys := Times(r)
		rs := fmt.Sprintf(rSum, r.Name, SolveTotal(r), SatTotal(r), UnsatTotal(r), UnknownTotal(r),
			real, user, sys)
		parts = append(parts, rs)
	}
	return strings.Join(parts, "\n")
}

// Listing produces a listing of all instances
// in all runs.
func Listing(s *Suite) string {
	cols := make([][]string, len(s.Runs)+2)
	nms := make([]string, len(s.Insts)+1)
	nms[0] = " name             "
	nums := make([]string, len(s.Insts)+1)
	nums[0] = "id   "
	for i, _ := range s.Insts {
		nms[i+1] = rtrunc(s.Map[i], 18)
		nums[i+1] = fmt.Sprintf("%-5d", i)
	}
	cols[0] = nums
	cols[1] = nms

	for i, run := range s.Runs {
		col := make([]string, len(s.Insts)+1)
		col[0] = fmt.Sprintf(" %-10s ", rtrunc(run.Name, 10))
		for j, _ := range s.Insts {
			ir := run.InstRuns[j]
			s := "s"
			if ir.Result == -1 {
				s = "u"
			} else if ir.Result == 0 {
				s = "?"
			}
			ds := float64(ir.Dur) / float64(time.Second)
			col[j+1] = fmt.Sprintf(" %s % 8.2f ", s, ds)
		}
		cols[i+2] = col
	}
	rows := make([]string, len(s.Insts)+2)
	for i := 0; i < len(s.Insts)+1; i++ {
		row := make([]string, len(s.Runs)+2)
		for j := 0; j < len(s.Runs)+2; j++ {
			row[j] = cols[j][i]
		}
		rows[i] = strings.Join(row, " | ")
	}
	return strings.Join(rows, "|\n")
}

func rtrunc(s string, n int) string {
	ct := utf8.RuneCount([]byte(s))
	j := 0
	for i, _ := range s {
		if j >= ct-n {
			return s[i:]
		}
		j++
	}
	return s
}
