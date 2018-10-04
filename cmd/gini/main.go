// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package main

import (
	"compress/bzip2"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	proto "github.com/irifrance/gini/crisp"
	"github.com/irifrance/gini/internal/xo"
	"github.com/irifrance/gini/z"
)

var pprofAddr = flag.String("pprof", "", "address to serve http profile (eg :6060)")
var timeout = flag.Duration("timeout", time.Second*30, "timeout")
var model = flag.Bool("model", false, "output model (default false)")
var satcomp = flag.Bool("satcomp", false, "if true, exit 10 sat, 20 unsat and output dimacs (default false)")
var stats = flag.Bool("stats", false, "if true, print some statistics after solving (default false)")
var mon = flag.Duration("mon", 0*time.Second, "if non-zero, print statistics during solving (default 0, implies -stats)")
var crisp = flag.String("crisp", "", "address of crisp server to use")
var failed = flag.Bool("failed", false, "output failed assumptions")

type assumes []z.Lit

func (a *assumes) String() string {
	return fmt.Sprintf("%+v", *a)
}
func (a *assumes) Set(val string) error {
	parts := strings.Split(val, ",")
	for _, val := range parts {
		i, e := strconv.Atoi(val)
		if e != nil {
			return e
		}
		if i == 0 {
			return fmt.Errorf("zero assumption")
		}
		*a = append(*a, z.Dimacs2Lit(i))
	}
	return nil
}
func (a *assumes) Get() interface{} {
	return []z.Lit(*a)
}

var assumptions = assumes([]z.Lit{})

func path2Reader(p string) (io.Reader, error) {
	if p == "-" {
		return os.Stdin, nil
	}
	st, stErr := os.Stat(p)
	if stErr != nil {
		return nil, stErr
	}
	if st.Mode()&os.ModeSymlink != 0 {
		q, e := os.Readlink(p)
		if e != nil {
			return nil, e
		}
		p = q
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	if strings.HasSuffix(p, ".gz") {
		r, e := gzip.NewReader(f)
		if e != nil {
			return nil, e
		}
		return r, nil
	}
	if strings.HasSuffix(p, ".bz2") {
		return bzip2.NewReader(f), nil
	}
	return f, nil
}

type format int

const (
	cnf format = 1 + iota
	icnf
)

func path2Format(p string) (format, error) {
	q := p
	if strings.HasSuffix(p, ".gz") {
		q = p[:len(p)-3]
	}
	if strings.HasSuffix(p, ".bz2") {
		q = p[:len(p)-4]
	}
	if strings.HasSuffix(q, ".cnf") {
		return cnf, nil
	}
	if strings.HasSuffix(q, ".icnf") {
		return icnf, nil
	}
	if p == "-" {
		return cnf, nil
	}
	return 0, fmt.Errorf("path extension doesn't have .cnf or .icnf")
}

func (f format) runReader(r io.Reader) (int, error) {
	switch f {
	case cnf:
		return runCnfReader(r)
	case icnf:
		return runICnfReader(r)
	default:
		panic(fmt.Sprintf("invalid format %d\n", f))
	}
}

func handleExit(res int) {
	if !*satcomp {
		return
	}
	log.SetPrefix("s ")
	if res == 1 {
		os.Exit(10)
	}
	if res == 0 {
		os.Exit(0)
	}
	if res == -1 {
		os.Exit(20)
	}
	panic(fmt.Sprintf("unknown result, %d", res))
}

func handleResultOutput(res int) {
	if res == 1 {
		fmt.Printf("s SATISFIABLE\n")
		return
	}
	if res == 0 {
		fmt.Printf("s UNKNOWN\n")
		return
	}
	if res == -1 {
		fmt.Printf("s UNSATISFIABLE\n")
		return
	}
	panic(fmt.Sprintf("unknown result, %d", res))
}

func runSatComp() {
	var r io.Reader
	var e error
	var res int
	if flag.NArg() == 0 {
		res, e = runCnfReader(os.Stdin)
	} else {
		r, e = path2Reader(flag.Arg(0))
		if e != nil {
			log.Printf("%s", e)
			os.Exit(1)
		}
		format, fe := path2Format(flag.Arg(0))
		if fe != nil {
			log.Printf("%s", fe)
			os.Exit(1)
		}
		res, e = format.runReader(r)
	}
	if e != nil {
		log.Println(e)
	}
	handleExit(res)
}

func runCnfReader(r io.Reader) (int, error) {
	if *crisp != "" {
		if *mon != 0 {
			log.Printf("cannot monitor stats over CRISP, try setting XO_STATS=yes\n")
		}
		return runCrispReader(r)
	}
	return runXoReader(r)
}

type values interface {
	value(m z.Lit) bool
}

type xoModel struct {
	x *xo.S
}

func (x xoModel) value(m z.Lit) bool {
	return x.x.Value(z.Lit(m))
}

type crispValues []bool

func (vs crispValues) value(m z.Lit) bool {
	return vs[m.Var()]
}

func outputModel(v z.Var, m values) {
	var col = 2
	fmt.Printf("v ")
	for i := z.Var(1); i <= v; i++ {
		n := 0
		for j := i; j > 0; j = j / 10 {
			n++
		}
		t := m.value(i.Pos())
		if !t {
			n++
		}
		if col+n > 78 {
			fmt.Printf("\nv")
			col = 2
		}
		if t {
			fmt.Printf(" %s", i.Pos())
		} else {
			fmt.Printf(" %s", i.Neg())
		}
		col++
		col += n
	}
	fmt.Printf(" 0\n")
}

func outputFailed(fs []z.Lit) {
	var col = 2
	fmt.Printf("f ")
	for _, f := range fs {
		n := 0
		for j := f.Var(); j > 0; j = j / 10 {
			n++
		}
		fd := f.Dimacs()
		if fd < 0 {
			n++
		}
		if col+n > 78 {
			fmt.Printf("\nf")
			col = 2
		}
		fmt.Printf(" %d", fd)
		col++
		col += n
	}
	fmt.Printf("\n")
}

func runXoReader(r io.Reader) (int, error) {
	start := time.Now()
	x, de := xo.NewSDimacs(r)
	if de != nil {
		return 0, fmt.Errorf("error reading dimacs: %s\n", de)
	}
	dur := time.Since(start)
	log.Printf("parsed dimacs in %s\n", dur)

	for _, a := range assumptions {
		x.Assume(z.Lit(a))
	}
	if *mon != 0 {
		return runXoMonitored(x)
	}
	st := xo.NewStats()
	conn := x.GoSolve()

	res := conn.Try(*timeout)
	if *stats {
		x.ReadStats(st)
		st.Dur = time.Since(st.Start)
		log.Println(st)
	}
	handleResultOutput(res)
	if res == 1 && *model {
		outputModel(z.Var(x.Vars.Max), xoModel{x})
	}
	if res == -1 && *failed {
		xls := x.Why(nil)
		fs := make([]z.Lit, len(xls))
		for i, m := range xls {
			fs[i] = z.Lit(m)
		}
		outputFailed(fs)
	}
	return res, nil
}

func runXoMonitored(x *xo.S) (int, error) {
	c := x.GoSolve().(*xo.Ctl)
	stChan := c.TryStats(*timeout, *mon)
	result := 0
	for stRes := range stChan {
		if stRes.Stats != nil {
			log.Println(stRes.Stats)
		}
		result = stRes.Result
	}
	handleResultOutput(result)
	if result == 1 && *model {
		outputModel(z.Var(x.Vars.Max), xoModel{x})
	}
	if result == -1 && *failed {
		xls := x.Why(nil)
		fs := make([]z.Lit, len(xls))
		for i, m := range xls {
			fs[i] = z.Lit(m)
		}
		outputFailed(fs)
	}
	return result, nil
}

func runCrispReader(r io.Reader) (int, error) {
	c, e := proto.Dial(*crisp)
	if e != nil {
		log.Printf("error dialing crisp: %s\n", e)
		return 0, e
	}
	defer func() { c.Quit(); c.Close() }()
	if e := c.Dimacs(r); e != nil {
		log.Printf("error loading dimacs: %s\n", e)
		return 0, e
	}
	if e := c.Assume(assumptions...); e != nil {
		log.Printf("error making assumptions: %s", e)
		return 0, e
	}
	res, e := c.GoSolve().Try(*timeout)
	if e != nil {
		log.Printf("crisp client error: %s\n", e)
		return 0, e
	}
	handleResultOutput(res)
	if res == 1 && *model {
		m, e := c.Model(nil)
		if e != nil {
			return res, e
		}
		outputModel(c.MaxVar(), crispValues(m))
	}
	if res == -1 && *failed {
		fs, e := c.Why(nil)
		if e != nil {
			return res, e
		}
		outputFailed(fs)
	}
	return res, nil
}

func main() {
	flag.Usage = func() {
		p := os.Args[0]
		_, p = filepath.Split(p)
		fmt.Fprintf(os.Stderr, usage, p, p, p, p, p, p, p)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}

	log.SetPrefix("c [gini] ")
	flag.Var(&assumptions, "assume", "add an assumption")
	flag.Parse()
	if flag.NArg() > 1 && *satcomp {
		fmt.Fprintf(os.Stderr, "can't use -satcomp with more than one input.\n")
		os.Exit(1)
	}
	// wanna see profiling info?  no recompile necessary, no performance impact
	// when its not in use.
	if *pprofAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(*pprofAddr, nil))
		}()
	}
	if *satcomp {
		runSatComp()
		return
	}
	for i := 0; i < flag.NArg(); i++ {
		fName := flag.Arg(i)
		r, e := path2Reader(fName)
		if e != nil {
			log.Println(e)
			continue
		}
		format, e := path2Format(fName)
		log.Print(fName)
		_, e = format.runReader(r)
		if e != nil {
			log.Printf("error: %s\n", e)
		}
	}
}
