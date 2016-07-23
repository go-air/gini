// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package bench

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Type Suite describes the inputs to benchmark runs
// In general, the info in the struct is read-only.
type Suite struct {
	Root   string   // root directory
	Insts  []string // instance pathnames.
	Map    []string // map from instance to origin path
	Hashes []string // map from instance to hash
	Runs   []*Run   // runs linked to this input
}

// IsSuiteDir returns true if d appears to contain
// a bench.Suite
func IsSuiteDir(d string) bool {
	for _, p := range []string{
		d, suiteMapPath(d), suiteHashPath(d), suiteRunDir(d)} {
		_, ste := os.Stat(p)
		if ste != nil {
			return false
		}
	}
	return true
}

// OpenSuite opens a benchmark suite.
func OpenSuite(root string) (*Suite, error) {
	res := &Suite{Root: root}
	if e := res.readInsts(); e != nil {
		return nil, e
	}
	if e := res.readMap(); e != nil {
		return nil, e
	}
	if e := res.readHashes(); e != nil {
		return nil, e
	}
	if e := res.readRuns(); e != nil {
		return nil, e
	}
	return res, nil
}

// CreateSuite creates a benchmark suite rooted in root
// with the specified instances.
func CreateSuite(root string, insts []string) (*Suite, error) {
	s, e := setupSuite(root, insts)
	if e != nil {
		return nil, e
	}
	if e := s.copyInsts(insts); e != nil {
		return nil, e
	}
	return s, nil
}

// LinkSuite creates a benchmark suite in which instances are
// symlinks to the origin files.
func LinkSuite(root string, insts []string) (*Suite, error) {
	s, e := setupSuite(root, insts)
	if e != nil {
		return nil, e
	}
	if e := s.linkInsts(insts); e != nil {
		return nil, e
	}
	return s, nil
}

// Run creates a run from a command, an instance timeout, and
// a global timeout.
func (s *Suite) Run(name, cmd string, ito, to time.Duration) (*Run, error) {
	run, e := NewRun(s, name, cmd, ito, to)
	if e != nil {
		return nil, e
	}
	return run, nil
}

// RemoveRun removes a run named name from the suite s.
func (s *Suite) RemoveRun(name string) error {
	p := filepath.Join(suiteRunDir(s.Root), name)
	e := os.RemoveAll(p)
	if e != nil {
		return e
	}
	j := 0
	for _, r := range s.Runs {
		if r.Name == name {
			continue
		}
		s.Runs[j] = r
		j++
	}
	s.Runs = s.Runs[:j]
	return nil
}

// RunSelect selects a subset of runs based on a filtering function filt and
// returns the result in the form of a suite sr which is identical to s except
// that sr only contains runs in s such that filt(r) is true.
func (s *Suite) RunSelect(filt func(*Run) bool) *Suite {
	res := &Suite{}
	*res = *s
	j := 0
	for _, run := range res.Runs {
		if !filt(run) {
			continue
		}
		rr := &Run{}
		*rr = *run
		rr.Suite = res
		res.Runs[j] = rr
		j++
	}
	res.Runs = res.Runs[:j]
	return res
}

// Len returns the number of instances in the suite.
func (s *Suite) Len() int {
	return len(s.Insts)
}

func setupSuite(root string, insts []string) (*Suite, error) {
	_, ste := os.Stat(root)
	if ste == nil {
		return nil, fmt.Errorf("root %s already exists.", root)
	}
	if e := os.MkdirAll(root, 0755); e != nil {
		return nil, e
	}
	if e := os.Mkdir(suiteRunDir(root), 0755); e != nil {
		return nil, e
	}

	s := &Suite{
		Root: root}
	s.Map = make([]string, 0, len(insts))
	for _, inst := range insts {
		s.Map = append(s.Map, inst)
	}
	if e := s.writeMap(); e != nil {
		return nil, e
	}
	s.Hashes = make([]string, 0, len(insts))
	for _, inst := range insts {
		h, e := hash(inst)
		if e != nil {
			return nil, e
		}
		s.Hashes = append(s.Hashes, h)
	}
	if e := s.writeHashes(); e != nil {
		return nil, e
	}
	return s, nil
}

func (s *Suite) copyInsts(insts []string) error {
	iFmt := iFmtFor(len(insts))
	for i, inst := range insts {
		p := filepath.Join(s.Root, fmt.Sprintf(iFmt, i, exts(inst)))
		w, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if e != nil {
			return e
		}
		r, e := os.Open(inst)
		if e != nil {
			return e
		}
		_, e = io.Copy(w, r)
		if e != nil {
			return e
		}
		w.Close()
		r.Close()
	}
	return nil
}

func (s *Suite) linkInsts(insts []string) error {
	iFmt := iFmtFor(len(insts))
	for i, inst := range insts {
		p := fmt.Sprintf(iFmt, i, exts(inst))
		if e := os.Symlink(inst, p); e != nil {
			return e
		}
	}
	return nil
}

func iFmtFor(N int) string {
	n := 1
	w := 1
	for w < N {
		w *= 10
		n++
	}
	return fmt.Sprintf("bench-%%0%dd%%s", n)
}

var knownExts = map[string]bool{
	".cnf":    true,
	".icnf":   true,
	".bz2":    true,
	".gz":     true,
	".inccnf": true}

func exts(p string) string {
	res := make([]string, 0, 2)
	for {
		e := filepath.Ext(p)
		if e == "" {
			break
		}
		_, ok := knownExts[e]
		if ok {
			res = append(res, e)
		}
		i := strings.LastIndex(p, ".")
		p = p[:i]
	}
	b := 0
	e := len(res) - 1
	for b < e {
		res[b], res[e] = res[e], res[b]
		b++
		e--
	}
	return strings.Join(res, "")
}

func hash(p string) (string, error) {
	f, e := os.Open(p)
	if e != nil {
		return "", e
	}
	defer f.Close()
	sha := sha256.New()
	_, e = io.Copy(sha, f)
	if e != nil {
		return "", e
	}
	sum := sha.Sum(nil)
	return hex.EncodeToString(sum), nil
}

func (s *Suite) readInsts() error {
	files, e := ioutil.ReadDir(s.Root)
	if e != nil {
		return e
	}

	iMap := make(map[int]string, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		nm := f.Name()
		if nm == "map" || nm == "hash" {
			continue
		}
		j := -1
		_, e := fmt.Sscanf(nm, "bench-%d.", &j)
		if e != nil {
			log.Printf("unexpected file in benchmark dir %s: %s\n", s.Root, nm)
			continue
		}
		iMap[j] = nm
	}
	s.Insts = make([]string, len(iMap))
	for k, nm := range iMap {
		if k < 0 || k >= len(s.Insts) {
			return fmt.Errorf("error, bad ordering for instance path: %s\n", nm)
		}
		s.Insts[k] = nm
	}
	return nil
}

func (s *Suite) readMap() error {
	mf, e := os.Open(suiteMapPath(s.Root))
	if e != nil {
		return e
	}
	defer mf.Close()
	r := bufio.NewReader(mf)
	for {
		line, e := r.ReadString(byte('\n'))
		if e != nil && e != io.EOF {
			return e
		}
		s.Map = append(s.Map, strings.TrimSpace(line))
		if e == io.EOF {
			return nil
		}
	}
}

func (s *Suite) readHashes() error {
	hf, e := os.Open(suiteHashPath(s.Root))
	if e != nil {
		return e
	}
	defer hf.Close()
	r := bufio.NewReader(hf)
	for {
		line, e := r.ReadString(byte('\n'))
		if e != nil && e != io.EOF {
			return e
		}
		s.Hashes = append(s.Hashes, line)
		if e == io.EOF {
			return nil
		}
	}
}

func (s *Suite) readRuns() error {
	rDir := suiteRunDir(s.Root)
	st, ste := os.Stat(rDir)
	if ste != nil {
		return ste
	}
	if !st.IsDir() {
		return fmt.Errorf("%s not a directory.", rDir)
	}
	files, e := ioutil.ReadDir(rDir)
	if e != nil {
		return e
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		r, re := OpenRun(s, filepath.Join(rDir, f.Name()))
		if re != nil {
			log.Printf("error opening run '%s': %s\n", f.Name(), re)
			continue
		}
		s.Runs = append(s.Runs, r)
	}
	return nil
}

func (s *Suite) writeMap() error {
	mf, e := os.OpenFile(suiteMapPath(s.Root), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer mf.Close()
	for _, e := range s.Map {
		fmt.Fprintln(mf, e)
	}
	return nil
}

func (s *Suite) writeHashes() error {
	hf, e := os.OpenFile(suiteHashPath(s.Root), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer hf.Close()
	for _, e := range s.Hashes {
		fmt.Fprintln(hf, e)
	}
	return nil
}

func suiteMapPath(root string) string {
	return filepath.Join(root, "map")
}

func suiteHashPath(root string) string {
	return filepath.Join(root, "hash")
}

func suiteRunDir(root string) string {
	return filepath.Join(root, "runs")
}
