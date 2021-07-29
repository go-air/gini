// Copyright 2018 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package aiger

import (
	"bytes"
	"os"
	"testing"

	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"
	//	"fmt"
)

// note this is 1.9 version: we have MILOABCJF
var expectedOutput1 = `aag 4 1 1 2 1 0 0 0 0
2
4 6 0
4
5
6 2 4
c
aiger file version 1.9 created by gini
`
var expectedOutput2 = `aig 4 1 1 2 1 0 0 0 0
6 1
4
5
c
aiger file version 1.9 created by gini
`

func makeExample() *T {
	sys := logic.NewSCap(11)
	in := sys.Lit()
	m := sys.Latch(sys.F)
	sys.SetNext(m, sys.F)
	a := sys.And(in, m)
	sys.SetNext(m, a)
	return MakeFor(sys, m, m.Not())
}

func TestWriteAscii(t *testing.T) {
	sys := makeExample()
	var buf bytes.Buffer
	err := sys.WriteAscii(&buf)
	if err != nil {
		t.Errorf("unexpected error in write ascii")
	}
	if buf.String() != expectedOutput1 {
		t.Errorf("unexpected output: %s\nvs\n%s", buf.String(), expectedOutput1)
	}
}

func TestWriteBinary(t *testing.T) {
	sys := makeExample()
	var buf bytes.Buffer
	err := sys.WriteBinary(&buf)
	if err != nil {
		t.Errorf("WriteBinary gave an error")
	}
	if buf.String() != expectedOutput2 {
		t.Errorf("unexpected output got '%s' vs '%s'\n", buf.String(), expectedOutput2)
	}
}

var binaryExample = `aig 4 1 1 2 1 0 0 2 0
6 1
4
5
2
1
4
5
4
i0 first-input
c
aiger file version 1.9 created by gini
`

type testSig struct {
	Max, Inputs, Latches, Outputs, Ands, Bads, Constraints, Justices, Fairs int
	err                                                                     error
}

var binTestMap = map[string]testSig{
	"test/lmcs06brp0.aig": testSig{
		Max: 902, Inputs: 47, Latches: 89,
		Outputs: 0, Ands: 766, Bads: 0,
		Constraints: 1, Justices: 1},
	"test/lmcs06short1.aig": testSig{
		Max: 71, Inputs: 8, Latches: 10,
		Outputs: 0, Ands: 53, Bads: 0,
		Constraints: 0, Justices: 1},
	"test/sm98tcas16tmulti.aig": testSig{
		Max: 5757, Inputs: 279, Latches: 310,
		Outputs: 0, Ands: 5168, Bads: 6, Constraints: 1},
	"test/visarbiter.aig": testSig{
		Max: 464, Inputs: 3, Latches: 23, Outputs: 1, Ands: 438},
	"test/visemodel.aig": testSig{
		Max: 340, Inputs: 11, Latches: 15, Outputs: 1, Ands: 314}}

func checkAiger(k string, a *T, err error, testSig *testSig, t *testing.T) {
	if err != testSig.err {
		t.Errorf("%s: error mismatch %s != %s", k, err, testSig.err)
	}
	if err != nil || testSig.err != nil {
		return
	}

	if len(a.Inputs) != testSig.Inputs {
		t.Errorf("%s: input mismatch: %d != %d", k,
			len(a.Inputs), testSig.Inputs)
	}
	if len(a.Latches) != testSig.Latches {
		t.Errorf("%s: latch count mismatch %d != %d",
			k, len(a.Latches), testSig.Latches)
	}
	if len(a.Outputs) > testSig.Outputs {
		// hmm Outputs are a set but not a set in aiger
		t.Errorf("%s: too many outputs", k)
	}
	if len(a.Bad) != testSig.Bads {
		t.Errorf("%s: wrong number of bad states: %d != %d",
			k, len(a.Bad), testSig.Bads)
	}
	if len(a.Constraints) != testSig.Constraints {
		t.Errorf("%s:, wrong number of constraints %d != %d",
			k, len(a.Constraints), testSig.Constraints)
	}
	if len(a.Justice) != testSig.Justices {
		t.Errorf("%s: wrong number of justice properties %d != %d",
			k, len(a.Justice), testSig.Justices)
	}
	if len(a.Fair) != testSig.Fairs {
		t.Errorf("%s: wrong number of fairness constraints %d != %d",
			k, len(a.Fair), testSig.Fairs)
	}
}

func TestReadBinary(t *testing.T) {
	buf := bytes.NewBufferString(binaryExample)
	aiger, err := ReadBinary(buf)
	if err != nil {
		t.Errorf("error reading binary: '%s'", err)
		return
	}
	if len(aiger.Justice) != 2 {
		t.Errorf("wrong number of justice properties")
	}
	for k, v := range binTestMap {
		f, err := os.Open(k)
		if err != nil {
			t.Errorf("couldn't open %s", k)
			continue
		}
		aiger, err := ReadBinary(f)
		checkAiger(k, aiger, err, &v, t)
		f.Close()
	}
}

var asciiTestMap = map[string]testSig{
	"test/combloop.aag": testSig{
		err: CombLoop},
	"test/multidef.aag": testSig{
		err: AndMultiplyDefined},
	"test/nextundef.aag": testSig{
		err: UndefinedLit},
	"test/badinit.aag": testSig{
		err: InvalidLatchInit},
	"test/resetenable.aag": testSig{
		Max: 7, Inputs: 2, Latches: 1,
		Outputs: 2, Ands: 4},
	"test/empty.aag": testSig{},
	"test/true.aag":  testSig{Outputs: 1},
	"test/halfadder.aag": testSig{
		Max: 7, Inputs: 2, Latches: 0,
		Outputs: 2, Ands: 3},
	"test/toggle-re.aag": testSig{
		Max: 7, Inputs: 2, Latches: 1,
		Outputs: 2, Ands: 4}}

func TestReadAscii(t *testing.T) {
	for k, v := range asciiTestMap {
		f, err := os.Open(k)
		if err != nil {
			t.Errorf("couldn't open %s", k)
			continue
		}
		aiger, err := ReadAscii(f)
		checkAiger(k, aiger, err, &v, t)
		f.Close()
	}
}

func TestFoo(t *testing.T) {
	g := Make(10)
	g.NewIn()
	if err := g.NameInput(0, "i"); err != nil {
		t.Errorf("couldn't name input 0 'i'")
	}
	nm, ok := g.InputName(0)
	if nm != "i" {
		t.Errorf("name didn't work.")
	}
	if !ok {
		t.Errorf("not ok")
	}
}

func TestWriteRead(t *testing.T) {
	g := Make(10)
	i := g.NewIn()
	ii := g.NewIn()
	g.Latch(g.S.F)
	g.Latch(g.S.F)
	a := g.And(i, ii)
	g.SetOutput(a)
	if err := g.NameInput(0, "i"); err != nil {
		t.Errorf("couldn't name input 0: '%s'", err)
	}
	if err := g.NameLatch(0, "l"); err != nil {
		t.Errorf("couldn't name latch 0: '%s'", err)
	}
	if err := g.NameOutput(0, "o"); err != nil {
		t.Errorf("couldn't name output 0: '%s'", err)
	}

	g.Bad = append(g.Bad, a)
	if err := g.NameBad(0, "b"); err != nil {
		t.Errorf("couldn't name bad 0: '%s'", err)
	}

	g.Constraints = append(g.Constraints, a)

	if err := g.NameConstraint(0, "c"); err != nil {
		t.Errorf("couldn't name constraint 0: '%s'", err)
	}

	j := make([]z.Lit, 1)
	j[0] = a
	g.Justice = append(g.Justice, j)
	if err := g.NameJustice(0, "j"); err != nil {
		t.Errorf("couldn't name justic 0: '%s'", err)
	}

	g.Fair = append(g.Fair, a)
	if err := g.NameFair(0, "f"); err != nil {
		t.Errorf("couldn't name fair 0: '%s'", err)
	}

	if _, found := g.InputName(1); found {
		t.Errorf("input 1 has name, shouldn't")
	}
	if nm, found := g.InputName(0); nm != "i" || !found {
		t.Errorf("wrong name for input 0: '%s' or not found: '%v'", nm, found)
	}

	buf := new(bytes.Buffer)
	g.WriteBinary(buf)

	gg, err := ReadBinary(buf)
	if err != nil {
		t.Errorf("couldn't read written binary: '%s'", err)
		return
	}
	if len(gg.Inputs) != len(g.Inputs) {
		t.Errorf("wrong #inputs after write->read")
	}
	if len(gg.Latches) != len(g.Latches) {
		t.Errorf("wrong #latches after write->read")
	}
	if len(gg.Outputs) != len(g.Outputs) {
		t.Errorf("wrong #outputs after write->read")
	}
	if len(gg.Bad) != len(g.Bad) {
		t.Errorf("wrong #bad after write->read")
	}
	if len(gg.Constraints) != len(g.Constraints) {
		t.Errorf("wrong #constraints after write->read")
	}
	if len(gg.Justice) != len(g.Justice) {
		t.Errorf("wrong #justice after write->read")
	}
	if len(gg.Fair) != len(g.Fair) {
		t.Errorf("wrong #fair after write->read")
	}
	if nm, _ := gg.InputName(0); nm != "i" {
		t.Errorf("after write/read, input 0 wrong: '%s'", nm)
	}
	if nm, _ := gg.LatchName(0); nm != "l" {
		t.Errorf("after write/read, latch 0 wrong: '%s'", nm)
	}
	if nm, _ := gg.OutputName(0); nm != "o" {
		t.Errorf("after write/read, output 0 wrong: '%s'", nm)
	}
	if nm, _ := gg.BadName(0); nm != "b" {
		t.Errorf("after write/read, bad 0 wrong: '%s'", nm)
	}
	if nm, _ := gg.ConstraintName(0); nm != "c" {
		t.Errorf("after write/read constraint 0 wrong: '%s'", nm)
	}
	if nm, _ := gg.JusticeName(0); nm != "j" {
		t.Errorf("after write/read, justice 0 wrong: '%s'", nm)
	}
	if nm, _ := gg.FairName(0); nm != "f" {
		t.Errorf("after write/read, fair 0 wrong: '%s'", nm)
	}
}
