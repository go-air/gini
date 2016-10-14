// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package bench

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
)

type walk struct {
	pattern string
	Collect []string
}

func (w *walk) Walk(p string, st os.FileInfo, e error) error {
	if e != nil {
		return e
	}
	if st.IsDir() {
		return nil
	}
	if w.pattern == "" {
		w.Collect = append(w.Collect, p)
		return nil
	}

	_, f := filepath.Split(p)
	matched, e := filepath.Match(w.pattern, f)
	if e != nil {
		fmt.Printf("error match: %s\n", e)
		return e
	}
	if matched {
		w.Collect = append(w.Collect, p)
	}
	return nil
}

// Select walks all the directories listed in dirs and
// selects up to N files randomly from all files in
// any directory in p.
//
// Select logs errors.
func Select(N int, dirs ...string) []string {
	return MatchSelect("", N, dirs...)
}

// MatchSelect is like Select but filters files with filepath.Match
// using pattern.
func MatchSelect(pattern string, N int, dirs ...string) []string {
	w := &walk{pattern: pattern}
	for _, p := range dirs {
		if e := filepath.Walk(p, w.Walk); e != nil {
			log.Printf("couldn't walk %s: %s, skipping.\n", p, e)
		}
	}
	n := 0
	orgLen := len(w.Collect)
	insts := make([]string, 0, N)
	for n < N {
		if len(w.Collect) == 0 {
			log.Printf("couldn't select %d, only %d choices.\n", N, orgLen)
			break
		}
		e := len(w.Collect)
		c := rand.Intn(e)
		e--
		w.Collect[c], w.Collect[e] = w.Collect[e], w.Collect[c]
		insts = append(insts, w.Collect[e])
		n++
		w.Collect = w.Collect[:e]
	}
	return insts
}
