// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package bench

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Type Run describes a run of a command/solver on a *Suite
type Run struct {
	Root        string
	Name        string
	Suite       *Suite
	Cmd         string
	Desc        string
	Commit      string
	Env         map[string]string
	Arch        string
	Os          string
	NumCPU      int
	Start       time.Time
	Timeout     time.Duration
	InstTimeout time.Duration
	InstRuns    []*InstRun
}

// IsRunDir tests whether or not root looks like a run directory.
func IsRunDir(root string) bool {
	for _, p := range []string{
		root, runCmdPath(root), runDescPath(root), runCommitPath(root),
		runTimeoutPath(root), runTimesPath(root), runResultsPath(root)} {
		_, ste := os.Stat(p)
		if ste != nil {
			return false
		}
	}
	return true
}

// OpenRun opens a Run.
func OpenRun(suite *Suite, root string) (*Run, error) {
	_, fn := filepath.Split(root)
	r := &Run{Root: root, Suite: suite, Name: fn}
	if e := r.readCmd(); e != nil {
		return nil, e
	}
	if e := r.readArch(); e != nil {
		return nil, e
	}
	if e := r.readOs(); e != nil {
		return nil, e
	}
	if e := r.readNumCpu(); e != nil {
		return nil, e
	}
	if e := r.readDesc(); e != nil {
		return nil, e
	}
	if e := r.readCommit(); e != nil {
		return nil, e
	}
	if e := r.readEnv(); e != nil {
		return nil, e
	}
	if e := r.readInstRuns(); e != nil {
		return nil, e
	}
	if e := r.readTimeout(); e != nil {
		return nil, e
	}
	if e := r.readInstTimeout(); e != nil {
		return nil, e
	}
	if e := r.readStart(); e != nil {
		return nil, e
	}
	return r, nil
}

// NewRun creates a new run for the *Suite suite rooted at root with
// instance timeout ito and global timeout to for command cmd.
//
// The resulting Run is (unfortunately) not yet openable since there
// are no results
func NewRun(suite *Suite, name, cmd string, ito, to time.Duration) (*Run, error) {
	d, fn := filepath.Split(name)
	if d != "" {
		return nil, fmt.Errorf("run name should not contain a directory")
	}
	name = fn
	run := &Run{
		Cmd:         cmd,
		Arch:        runtime.GOARCH,
		Os:          runtime.GOOS,
		NumCPU:      runtime.NumCPU(),
		Root:        filepath.Join(suiteRunDir(suite.Root), name),
		Commit:      "00000000",
		Start:       time.Now(),
		Suite:       suite,
		Timeout:     to,
		InstTimeout: ito}

	env := os.Environ()
	run.Env = make(map[string]string, len(env))
	for _, e := range env {
		parts := strings.Split(e, "=")
		run.Env[parts[0]] = strings.Join(parts[1:], "=")
	}
	if e := os.MkdirAll(run.Root, 0755); e != nil {
		return nil, e
	}
	if e := run.writeArch(); e != nil {
		return nil, e
	}
	if e := run.writeCmd(); e != nil {
		return nil, e
	}
	if e := run.writeCommit(); e != nil {
		return nil, e
	}
	if e := run.writeDesc(); e != nil {
		return nil, e
	}
	if e := run.writeEnv(); e != nil {
		return nil, e
	}
	if e := run.writeInstTimeout(); e != nil {
		return nil, e
	}
	if e := run.writeNumCpu(); e != nil {
		return nil, e
	}
	if e := run.writeOs(); e != nil {
		return nil, e
	}
	if e := run.writeTimeout(); e != nil {
		return nil, e
	}
	if e := run.writeStart(); e != nil {
		return nil, e
	}
	return run, nil
}

func (r *Run) Len() int {
	return len(r.Suite.Insts)
}

func (r *Run) Do(i int) (*InstRun, error) {
	return NewInstRun(r, i)
}

func (r *Run) readCmd() error {
	s, e := p2s(runCmdPath(r.Root))
	if e != nil {
		return e
	}
	r.Cmd = s
	return nil
}
func (r *Run) writeCmd() error {
	return s2f(r.Cmd, runCmdPath(r.Root))
}

func (r *Run) readDesc() error {
	s, e := p2s(runDescPath(r.Root))
	if e != nil {
		return e
	}
	r.Desc = s
	return nil
}
func (r *Run) writeDesc() error {
	return s2f(r.Desc, runDescPath(r.Root))
}

func (r *Run) readCommit() error {
	s, e := p2s(runCommitPath(r.Root))
	if e != nil {
		return e
	}
	r.Commit = strings.TrimSpace(s)
	return nil
}
func (r *Run) writeCommit() error {
	return s2f(fmt.Sprintf("%s\n", r.Commit), runCommitPath(r.Root))
}

func (r *Run) readArch() error {
	s, e := p2s(runArchPath(r.Root))
	if e != nil {
		return e
	}
	r.Arch = strings.TrimSpace(s)
	return nil
}
func (r *Run) writeArch() error {
	return s2f(fmt.Sprintf("%s\n", r.Arch), runArchPath(r.Root))
}

func (r *Run) readOs() error {
	s, e := p2s(runOsPath(r.Root))
	if e != nil {
		return e
	}
	r.Os = strings.TrimSpace(s)
	return nil
}
func (r *Run) writeOs() error {
	return s2f(fmt.Sprintf("%s\n", r.Os), runOsPath(r.Root))
}

func (r *Run) readNumCpu() error {
	p := runNumCpuPath(r.Root)
	f, e := os.Open(p)
	if e != nil {
		return e
	}
	defer f.Close()
	n := 0
	i, e := fmt.Fscanf(f, "%d", &n)
	if i != 1 {
		return e
	}
	r.NumCPU = n
	return nil
}
func (r *Run) writeNumCpu() error {
	f, e := os.OpenFile(runNumCpuPath(r.Root), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	fmt.Fprintf(f, "%d\n", r.NumCPU)
	return nil
}

func (r *Run) readEnv() error {
	env := os.Environ()
	r.Env = make(map[string]string, len(env))
	for _, kv := range env {
		parts := strings.Split(kv, "=")
		if len(parts) != 2 {
			return fmt.Errorf("environ %s\n", kv)
		}
		r.Env[parts[0]] = parts[1]
	}
	return nil
}
func (r *Run) writeEnv() error {
	p := runEnvPath(r.Root)
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	for k, v := range r.Env {
		fmt.Fprintf(f, "%s=%s\n", k, v)
	}
	return nil
}

func (r *Run) readTimeout() error {
	p := runTimeoutPath(r.Root)
	d, e := p2d(p)
	if e != nil {
		return e
	}
	r.Timeout = d
	return nil
}
func (r *Run) writeTimeout() error {
	return d2f(runTimeoutPath(r.Root), r.Timeout)
}

func (r *Run) readInstTimeout() error {
	p := runInstTimeoutPath(r.Root)
	d, e := p2d(p)
	if e != nil {
		return e
	}
	r.InstTimeout = d
	return nil
}
func (r *Run) writeInstTimeout() error {
	return d2f(runInstTimeoutPath(r.Root), r.InstTimeout)
}

func (r *Run) readStart() error {
	p := runStartPath(r.Root)
	t, e := p2t(p)
	if e != nil {
		return e
	}
	r.Start = *t
	return nil
}
func (r *Run) writeStart() error {
	p := runStartPath(r.Root)
	return t2f(p, r.Start)
}

func (r *Run) readInstRuns() error {
	suite := r.Suite
	for i, _ := range suite.Insts {
		ir, e := OpenInstRun(r, i)
		if e != nil {
			log.Printf("error opening inst run %d in suite %s: %s\n", i, suite.Root, e)
			continue
		}
		r.InstRuns = append(r.InstRuns, ir)
	}
	return nil
}

func p2s(p string) (string, error) {
	f, e := os.Open(p)
	if e != nil {
		return "", e
	}
	defer f.Close()
	st, ste := f.Stat()
	if ste != nil {
		return "", ste
	}
	sz := int(st.Size())
	buf := make([]byte, sz)
	if _, e := io.ReadFull(f, buf); e != nil {
		return "", e
	}
	return string(buf), nil
}

func s2f(s, p string) error {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	n := 0
	buf := []byte(s)
	N := len(buf)
	var w int
	for n < N {
		w, e = f.Write(buf[n:])
		if e != nil {
			return e
		}
		n += w
	}
	return nil
}

func p2d(p string) (time.Duration, error) {
	var d time.Duration
	f, e := os.Open(p)
	if e != nil {
		return d, e
	}
	defer f.Close()
	i := int64(0)
	n, e := fmt.Fscanf(f, "%d", &i)
	if e != nil || n != 1 {
		return d, e
	}
	return time.Duration(i), nil
}
func d2f(p string, d time.Duration) error {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	i := int64(d)
	_, e = fmt.Fprintf(f, "%d\n", i)
	return e
}
func p2t(p string) (*time.Time, error) {
	s, e := p2s(p)
	if e != nil {
		return nil, e
	}
	var t time.Time
	pt := &t
	if e := pt.UnmarshalText([]byte(s)); e != nil {
		return nil, e
	}
	return pt, nil
}
func t2f(p string, t time.Time) error {
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	b, e := t.MarshalText()
	if e != nil {
		return e
	}
	n := 0
	N := len(b)
	for n < N {
		w, e := f.Write(b[n:])
		if e != nil {
			return e
		}
		n += w
	}
	return nil
}
func runSuitePath(root string) string {
	return filepath.Join(root, "../../")
}
func runEnvPath(root string) string {
	return filepath.Join(root, "env")
}
func runDescPath(root string) string {
	return filepath.Join(root, "desc")
}
func runTimesPath(root string) string {
	return filepath.Join(root, "times")
}
func runCmdPath(root string) string {
	return filepath.Join(root, "cmd")
}
func runCommitPath(root string) string {
	return filepath.Join(root, "commit")
}
func runStartPath(root string) string {
	return filepath.Join(root, "start")
}
func runArchPath(root string) string {
	return filepath.Join(root, "arch")
}
func runOsPath(root string) string {
	return filepath.Join(root, "os")
}
func runNumCpuPath(root string) string {
	return filepath.Join(root, "ncpu")
}
func runTimeoutPath(root string) string {
	return filepath.Join(root, "timeout")
}
func runInstTimeoutPath(root string) string {
	return filepath.Join(root, "inst-timeout")
}
func runResultsPath(root string) string {
	return filepath.Join(root, "results")
}
