// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package bench

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type InstRun struct {
	Run    *Run
	Inst   int
	Result int
	Start  time.Time
	Dur    time.Duration
	UDur   time.Duration
	SDur   time.Duration
	Out    io.WriteCloser
	Err    io.WriteCloser
	Error  string
}

func OpenInstRun(run *Run, inst int) (*InstRun, error) {
	p := iRunPath(run.Root, inst)
	ir := &InstRun{
		Run:  run,
		Inst: inst}
	d, e := p2d(iRunDurPath(p))
	if e != nil {
		return nil, e
	}
	ir.Dur = d
	d, e = p2d(iRunUDurPath(p))
	if e != nil {
		return nil, e
	}
	ir.UDur = d
	d, e = p2d(iRunSDurPath(p))
	if e != nil {
		return nil, e
	}
	ir.SDur = d
	t, e := p2t(iRunStartPath(p))
	if e != nil {
		return nil, e
	}
	ir.Start = *t
	f, e := os.Open(iRunResPath(p))
	if e != nil {
		return nil, e
	}
	defer f.Close()
	_, e = fmt.Fscanf(f, "%d", &ir.Result)
	if e != nil {
		return nil, e
	}
	ie, e := p2s(iRunErrPath(p))
	if e != nil {
		return nil, e
	}
	ir.Error = ie
	return ir, nil
}

func NewInstRun(run *Run, inst int) (*InstRun, error) {
	ir := &InstRun{
		Run:  run,
		Inst: inst}

	orgDir, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	if e := ir.setup(); e != nil {
		return nil, e
	}
	ir.chDirAndDo()
	if e := os.Chdir(orgDir); e != nil {
		return nil, e
	}
	if e := ir.save(); e != nil {
		return nil, e
	}
	return ir, nil
}

func (ir *InstRun) chDirAndDo() {
	d := iRunPath(ir.Run.Root, ir.Inst)
	orgDir, e := os.Getwd()
	if e != nil {
		log.Printf("error: %s\n", e)
		return
	}
	orgDir, e = filepath.Abs(orgDir)
	if e != nil {
		log.Printf("error: %s\n", e)
		return
	}
	if e := os.Chdir(d); e != nil {
		log.Printf("error: %s\n", e)
		return
	}
	defer func() {
		if e := os.Chdir(orgDir); e != nil {
			log.Printf("error: %s\n", e)
		}
	}()
	p := ir.Run.Suite.Insts[ir.Inst]
	if e := os.Symlink(filepath.Join("../../../", p), p); e != nil {
		log.Printf("error: %s\n", e)
		return
	}
	ir.do()
}

func (ir *InstRun) do() {
	parts := strings.Split(ir.Run.Cmd, " ")
	parts = append(parts, ir.Run.Suite.Insts[ir.Inst])
	cmd := exec.Command(parts[0], parts[1:]...)
	out, e := cmd.StdoutPipe()
	if e != nil {
		log.Printf("error out pipe: %s\n", e)
		return
	}
	err, e := cmd.StderrPipe()
	if e != nil {
		log.Printf("error err pipe: %s\n", e)
		return
	}

	done := make(chan error)
	ir.Start = time.Now()
	ialarm := time.After(ir.dur())
	if e := cmd.Start(); e != nil {
		log.Printf("error starting command '%s': %s\n", ir.Run.Cmd, e)
		return
	}
	var wg sync.WaitGroup
	go func() {
		ir.capture(cmd, done, ialarm)
	}()
	wg.Add(1)
	go func() {
		_, e := io.Copy(ir.Out, out)
		if e != nil {
			log.Printf("error piping output: %s\n", e)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		_, e := io.Copy(ir.Err, err)
		if e != nil {
			log.Printf("error piping stderr: %s\n", e)
		}
		wg.Done()
	}()
	wg.Wait()
	done <- cmd.Wait()
	<-done
}

func (ir *InstRun) capture(cmd *exec.Cmd, done chan error, ialarm <-chan time.Time) {
	for {
		select {
		case <-ialarm:
			cmd.Process.Kill()
			ialarm = nil
		case e := <-done:
			if e != nil {
				ir.Error = e.Error()
			}
			ir.Dur = time.Since(ir.Start)
			ir.Result = 0
			if exitErr, ok := e.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					x := status.ExitStatus()
					if x == 10 {
						ir.Result = 1
					} else if x == 20 {
						ir.Result = -1
					} else {
						ir.Result = 0
					}
				}
			}
			ir.UDur = cmd.ProcessState.UserTime()
			ir.SDur = cmd.ProcessState.SystemTime()
			ir.Out.Close()
			ir.Err.Close()
			done <- nil
			return
		}
	}
}

func (ir *InstRun) dur() time.Duration {
	runStart := ir.Run.Start
	runEnd := runStart.Add(ir.Run.Timeout)
	instEnd := time.Now().Add(ir.Run.InstTimeout)
	if runEnd.Before(instEnd) {
		return runEnd.Sub(time.Now())
	}
	return ir.Run.InstTimeout
}

func (ir *InstRun) setup() error {
	d := iRunPath(ir.Run.Root, ir.Inst)
	if e := os.Mkdir(d, 0755); e != nil {
		ir.Error = e.Error()
		return e
	}
	outPath := filepath.Join(d, "out")
	out, e := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		ir.Error = e.Error()
		return e
	}
	ir.Out = out
	errPath := filepath.Join(d, "err")
	err, e := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		ir.Error = e.Error()
		return e
	}
	ir.Err = err
	ir.Start = time.Now()
	if e := t2f(iRunStartPath(d), ir.Start); e != nil {
		ir.Error = e.Error()
		return e
	}
	return nil
}

func (ir *InstRun) save() error {
	p := iRunPath(ir.Run.Root, ir.Inst)
	f, e := os.OpenFile(iRunResPath(p), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	fmt.Fprintf(f, "%d\n", ir.Result)
	f.Close()
	if e := d2f(iRunDurPath(p), ir.Dur); e != nil {
		return e
	}
	if e := d2f(iRunUDurPath(p), ir.UDur); e != nil {
		return e
	}
	if e := d2f(iRunSDurPath(p), ir.SDur); e != nil {
		return e
	}
	if e := s2f(fmt.Sprintf("%s\n", ir.Error), iRunErrPath(p)); e != nil {
		return e
	}
	return nil
}

func iRunPath(root string, i int) string {
	return filepath.Join(root, fmt.Sprintf("inst-%d.run", i))
}
func iRunResPath(root string) string {
	return filepath.Join(root, "result")
}
func iRunDurPath(root string) string {
	return filepath.Join(root, "dur")
}
func iRunUDurPath(root string) string {
	return filepath.Join(root, "udur")
}
func iRunSDurPath(root string) string {
	return filepath.Join(root, "sdur")
}
func iRunErrPath(root string) string {
	return filepath.Join(root, "error")
}
func iRunStartPath(root string) string {
	return filepath.Join(root, "start")
}
