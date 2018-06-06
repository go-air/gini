// Copyright 2018 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package aiger

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/irifrance/gini/logic"
	"github.com/irifrance/gini/z"
)

// Errors related to IO and formatting
var (
	PrematureEOF       = errors.New("premature EOF")
	ReadError          = errors.New("Read error")
	UnexpectedChar     = errors.New("Unexpected char")
	BadHeader          = errors.New("Bad header")
	BadUInt            = errors.New("malformed literal")
	BinaryMismatch     = errors.New("binary mismatch")
	InvalidLatchInit   = errors.New("invalid latch init value")
	LitOOB             = errors.New("Literal out of bounds")
	BadDeltaEncoding   = errors.New("Bad Delta Encoding")
	InvalidIndex       = errors.New("invalid index")
	InvalidSymbolType  = errors.New("invalid symbol type")
	InvalidName        = errors.New("invalid symbol name")
	SignedInput        = errors.New("input is negated")
	SignedLatch        = errors.New("latch is negated")
	SignedAnd          = errors.New("and gate def is negated")
	CombLoop           = errors.New("combinational logic has a loop")
	AndMultiplyDefined = errors.New("and gate multiply defined")
	UndefinedLit       = errors.New("literal not defined")
)

// Type Aiger contains the information read from or written to
// disk in Aiger format version 1.9
type T struct {
	*logic.S    // The Boolean system backing this Aiger object
	Inputs      []z.Lit
	Outputs     []z.Lit
	Bad         []z.Lit                 // Read/Write List of Bad state literals
	Constraints []z.Lit                 // Read/Write List of Environment Constraints
	Justice     [][]z.Lit               // Read/Write List of Justice Properties
	Fair        []z.Lit                 // Read/Write List of Fairness Constraints
	symbols     map[byte]map[int]string // symbol table
}

// MakeFor makes an Aiger object from a Boolean system.  The system
// is the backing store for the Aiger object, no copy is made
func MakeFor(sys *logic.S, ms ...z.Lit) *T {
	result := &T{
		S:           sys,
		Bad:         make([]z.Lit, 0),
		Constraints: make([]z.Lit, 0),
		Justice:     make([][]z.Lit, 0),
		Fair:        make([]z.Lit, 0),
		symbols:     make(map[byte]map[int]string, 0)}
	result.symbols['i'] = make(map[int]string, 0)
	result.symbols['l'] = make(map[int]string, 0)
	result.symbols['o'] = make(map[int]string, 0)
	result.symbols['b'] = make(map[int]string, 0)
	result.symbols['c'] = make(map[int]string, 0)
	result.symbols['j'] = make(map[int]string, 0)
	result.symbols['f'] = make(map[int]string, 0)
	n := sys.Len()
	for i := 1; i < n; i++ {
		m := sys.At(i)
		ty := sys.Type(m)
		if ty == logic.SInput {
			result.Inputs = append(result.Inputs, m)
		}
	}
	result.Outputs = make([]z.Lit, len(ms))
	copy(result.Outputs, ms)
	return result
}

// Make makes an Aiger object with initial capacity hint c
// for the underlying logic.S object
func Make(c int) *T {
	return MakeFor(logic.NewSCap(c))
}

// Copy makes a copy of an aiger object.
func Copy(a *T) *T {
	if a == nil {
		return nil
	}
	result := &T{
		S:           a.S.Copy(),
		Bad:         make([]z.Lit, len(a.Bad)),
		Constraints: make([]z.Lit, len(a.Constraints)),
		Justice:     make([][]z.Lit, len(a.Justice)),
		Fair:        make([]z.Lit, len(a.Fair)),
		symbols:     make(map[byte]map[int]string, len(a.symbols))}
	copy(result.Bad, a.Bad)
	copy(result.Constraints, a.Constraints)
	copy(result.Fair, a.Fair)
	for i := 0; i < len(a.Justice); i++ {
		result.Justice[i] = make([]z.Lit, len(a.Justice[i]))
		copy(result.Justice[i], a.Justice[i])
	}
	symKeys := [...]byte{'i', 'l', 'o', 'b', 'c', 'j', 'f'}
	for _, k := range symKeys {
		result.symbols[k] = make(map[int]string, len(a.symbols[k]))
		for i, nm := range a.symbols[k] {
			result.symbols[k][i] = nm
		}
	}
	return result
}

// Return the Boolean system backing this Aiger object
func (a *T) Sys() *logic.S {
	return a.S
}

// Name index'th input with name nm
// return a non-nil error if index is out of bounds or nm
// contains a new line
func (a *T) NameInput(index int, nm string) error {
	if index < 0 || index > len(a.Inputs) {
		return InvalidIndex
	}
	if strings.Contains(nm, "\n") {
		return InvalidName
	}
	a.symbols['i'][index] = nm
	return nil
}

// InputName gives the name of the index'th Input in the aiger
// system.  If no such name exists, InputName returns (nil, false).
// Otherwise, InputName returns (name, true).
func (a *T) InputName(index int) (string, bool) {
	nm, found := a.symbols['i'][index]
	return nm, found
}

// Name index'th Latch with name nm
// return a non-nil error if index is out of bounds or nm
// contains a new line
func (a *T) NameLatch(index int, nm string) error {
	if index < 0 || index > len(a.Latches) {
		return InvalidIndex
	}
	if strings.Contains(nm, "\n") {
		return InvalidName
	}
	a.symbols['l'][index] = nm
	return nil
}

// LatchName gives the name of the index'th Latch in the aiger
// system.  If no such name exists, LatchName returns nil, false.
// Otherwise, LatchName returns name, true.
func (a *T) LatchName(index int) (string, bool) {
	nm, found := a.symbols['l'][index]
	return nm, found
}

func (a *T) SetOutput(m z.Lit) {
	a.Outputs = append(a.Outputs, m)
}

func (a *T) NewIn() z.Lit {
	m := a.S.Lit()
	a.Inputs = append(a.Inputs, m)
	return m
}

// Name index'th output with name nm
// return a non-nil error if index is out of bounds or nm
// contains a new line
func (a *T) NameOutput(index int, nm string) error {
	if index < 0 || index > len(a.Outputs) {
		return InvalidIndex
	}
	if strings.Contains(nm, "\n") {
		return InvalidName
	}
	a.symbols['o'][index] = nm
	return nil
}

// OutputName gives the name of the index'th Output in the aiger
// system.  If no such name exists, OutputName returns nil, false.
// Otherwise, OutputName returns name, true.
func (a *T) OutputName(index int) (string, bool) {
	nm, found := a.symbols['o'][index]
	return nm, found
}

// Name index'th Bad State property with name nm
// return a non-nil error if index is out of bounds or nm
// contains a new line
func (a *T) NameBad(index int, nm string) error {
	if index < 0 || index > len(a.Bad) {
		return InvalidIndex
	}
	if strings.Contains(nm, "\n") {
		return InvalidName
	}
	a.symbols['b'][index] = nm
	return nil
}

// BadName gives the name of the index'th Bad state in the aiger
// system.  If no such name exists, BadName returns nil, false.
// Otherwise, BadName returns name, true.
func (a *T) BadName(index int) (string, bool) {
	nm, found := a.symbols['b'][index]
	return nm, found
}

// Name index'th Constraint with name nm
// return a non-nil error if index is out of bounds or nm
// contains a new line
func (a *T) NameConstraint(index int, nm string) error {
	if index < 0 || index > len(a.Constraints) {
		return InvalidIndex
	}
	if strings.Contains(nm, "\n") {
		return InvalidName
	}
	a.symbols['c'][index] = nm
	return nil
}

// ConstraintName gives the name of the index'th Constraint in the aiger
// system.  If no such name exists, ConstraintName returns nil, false.
// Otherwise, ConstraintName returns name, true.
func (a *T) ConstraintName(index int) (string, bool) {
	nm, found := a.symbols['c'][index]
	return nm, found
}

// Name index'th justice property with name nm
// return a non-nil error if index is out of bounds or nm
// contains a new line
func (a *T) NameJustice(index int, nm string) error {
	if index < 0 || index > len(a.Justice) {
		return InvalidIndex
	}
	if strings.Contains(nm, "\n") {
		return InvalidName
	}
	a.symbols['j'][index] = nm
	return nil
}

// JusticeName gives the name of the index'th Justice in the aiger
// system.  If no such name exists, JusticeName returns nil, false.
// Otherwise, JusticeName returns name, true.
func (a *T) JusticeName(index int) (string, bool) {
	nm, found := a.symbols['j'][index]
	return nm, found
}

// Name the index'th fairness constraint with name nm
// return a non-nil error if index is out of bounds or nm
// contains a new line
func (a *T) NameFair(index int, nm string) error {
	if index < 0 || index > len(a.Fair) {
		return InvalidIndex
	}
	if strings.Contains(nm, "\n") {
		return InvalidName
	}
	a.symbols['f'][index] = nm
	return nil
}

// FairName gives the name of the index'th Fair in the aiger
// system.  If no such name exists, FairName returns nil, false.
// Otherwise, FairName returns name, true.
func (a *T) FairName(index int) (string, bool) {
	nm, found := a.symbols['f'][index]
	return nm, found
}

// WriteAscii writes an ASCII version of AIGER format
// for the object a to the writer w.  WriteAscii returns
// a non-nil error if there was an io error while writing.
func (a *T) WriteAscii(w io.Writer) error {
	hdr := makeHeader(a, false)
	bw := bufio.NewWriter(w)
	hdr.write(bw)
	for _, m := range a.Inputs {
		writeLit(bw, m, a.S.T)
		bw.WriteString("\n")
	}
	for _, m := range a.Latches {
		writeLit(bw, m, a.S.T)
		bw.WriteString(" ")
		writeLit(bw, a.Next(m), a.S.T)
		bw.WriteString(" ")
		ini := a.Init(m)
		switch ini {
		case a.S.F:
			bw.WriteString("1\n")
		case a.S.T:
			bw.WriteString("0\n")
		case z.LitNull:
			writeLit(bw, m, a.S.T)
			bw.WriteString("\n")
		default:
			panic("invalid initial value")
		}
	}
	for _, m := range a.Outputs {
		writeLit(bw, m, a.S.T)
		bw.WriteString("\n")
	}
	for _, m := range a.Bad {
		writeLit(bw, m, a.S.T)
		bw.WriteString("\n")
	}
	for _, m := range a.Constraints {
		writeLit(bw, m, a.S.T)
		bw.WriteString("\n")
	}
	for _, ma := range a.Justice {
		bw.WriteString(fmt.Sprintf("%d\n", len(ma)))
	}
	for _, ma := range a.Justice {
		for _, m := range ma {
			writeLit(bw, m, a.S.T)
			bw.WriteString("\n")
		}
	}
	for _, m := range a.Fair {
		writeLit(bw, m, a.S.T)
		bw.WriteString("\n")
	}
	a.writeAsciiAnds(bw)
	a.writeSymtab(bw)
	writeComment(bw)
	return bw.Flush()
}

// WriteBinary writes the Boolean system sys in binary
// AIGER format (version 1.9) to the writer w.  WriterAigerBinary
// returns an error if there was an io error while writing.
func (a *T) WriteBinary(w io.Writer) error {
	hdr := makeHeader(a, true)
	bw := bufio.NewWriter(w)
	hdr.write(bw)
	abw := &aigerBinWriter{
		trueLit:   a.S.T,
		firstPass: true,
		w:         bw,
		id:        0,
		idMap:     make([]uint, a.Len())}

	// Stage1: create a mapping that matches binary aiger
	// identifier packing requirements (
	// const ids < all input ids < all latch ids < all and ids)
	// we map constant, then input, then latches,
	// finally ands (ands with a DFS traversal)
	abw.mapLit(a.S.T)
	for _, m := range a.Inputs {
		abw.mapLit(m)
	}
	for _, m := range a.Latches {
		abw.mapLit(m)
	}
	// create mapping for and gates
	nexts := make([]z.Lit, 0, len(a.Latches))
	for _, m := range a.Latches {
		nexts = append(nexts, a.Next(m))
	}
	dfs := newsDfs(a.S, func(s *logic.S, m z.Lit) {
		if s.Type(m) == logic.SAnd {
			abw.mapLit(m)
		}
	})
	dfs.post(a.Outputs...)
	dfs.post(nexts...)
	dfs.post(a.Bad...)
	dfs.post(a.Constraints...)
	for _, ma := range a.Justice {
		dfs.post(ma...)
	}
	dfs.post(a.Fair...)
	dfs.reset()

	// Stage2: write the remaining data.  Latches
	for _, m := range a.Latches {
		var init uint
		ini := a.Init(m)
		if ini == 0 {
			init = abw.forLit(m)
		} else if ini == a.S.F {
			init = 1
		} else if ini == a.S.T {
			init = 0
		} else {
			panic("invalid init state")
		}
		bw.WriteString(fmt.Sprintf("%d %d\n", abw.forLit(a.Next(m)), init))
	}
	// followed by outputs
	for _, m := range a.Outputs {
		bw.WriteString(fmt.Sprintf("%d\n", abw.forLit(m)))
	}
	for _, m := range a.Bad {
		bw.WriteString(fmt.Sprintf("%d\n", abw.forLit(m)))
	}
	for _, m := range a.Constraints {
		bw.WriteString(fmt.Sprintf("%d\n", abw.forLit(m)))
	}
	for _, ma := range a.Justice {
		bw.WriteString(fmt.Sprintf("%d\n", len(ma)))
	}
	for _, ma := range a.Justice {
		for _, m := range ma {
			bw.WriteString(fmt.Sprintf("%d\n", abw.forLit(m)))
		}
	}
	for _, m := range a.Fair {
		bw.WriteString(fmt.Sprintf("%d\n", abw.forLit(m)))
	}
	// second pass writes the ands in binary format.
	dfs.fn = abw.writeBinAnd
	dfs.post(a.Outputs...)
	dfs.post(nexts...)
	dfs.post(a.Bad...)
	for _, ma := range a.Justice {
		dfs.post(ma...)
	}
	dfs.post(a.Fair...)
	a.writeSymtab(bw)
	// finally write comment
	writeComment(bw)
	return bw.Flush()
}

// ReadAscii reads an ascii coded Aiger file (version 1.9)
// ReadAscii returns a possibly nil Aiger object system paired with
// a possibly nil error.  If the Aiger object is nil, the error
// is non-nil and indicates the underlying problem.
func ReadAscii(r io.Reader) (*T, error) {
	br := bufio.NewReader(r)
	hdr, err := readHeader(br)
	if err != nil {
		return nil, err
	}
	if hdr.Binary {
		return nil, BinaryMismatch
	}
	aiger := Make(int(hdr.Max + 1))
	aigrdr := makeAigerReader(aiger, hdr)
	if err := aigrdr.readAsciiInputs(hdr, br); err != nil {
		return nil, err
	}
	if err := aigrdr.readLatches(hdr, br, true); err != nil {
		return nil, err
	}
	if err := aigrdr.readOutputs(hdr.Out, hdr.Max, br); err != nil {
		return nil, err
	}
	if err := aigrdr.readBad(br, hdr.Bad, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readConstraints(br, hdr.Constraint, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readJustice(br, hdr.Justice, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readFair(br, hdr.Fair, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readAsciiAnds(hdr, br); err != nil {
		return nil, err
	}
	if err := aigrdr.readSymsAndComments(br); err != nil {
		return nil, err
	}
	if err := aigrdr.commit(true); err != nil {
		return nil, err
	}
	return aigrdr.T, nil
}

// ReadBinary reads a binary Aiger file (version 1.9)
// Readbinary returns a possibly nil Aiger object paired with
// a possibly nil error.  The error will be non-nil and describe
// the underlyng problem if the Aiger object is nil.
func ReadBinary(r io.Reader) (*T, error) {
	br := bufio.NewReader(r)
	hdr, err := readHeader(br)
	if err != nil {
		return nil, err
	}
	if !hdr.Binary {
		return nil, BinaryMismatch
	}
	aiger := Make(int(hdr.Max + 1))
	aigrdr := makeAigerReader(aiger, hdr)
	var i uint
	for i = 0; i < hdr.In; i++ {
		m := aigrdr.S.Lit()
		aigrdr.mapLit((i+1)*2, m)
		aigrdr.Inputs = append(aigrdr.Inputs, m)
	}
	if err := aigrdr.readLatches(hdr, br, false); err != nil {
		return nil, err
	}
	if err := aigrdr.readOutputs(hdr.Out, hdr.Max, br); err != nil {
		return nil, err
	}
	if err := aigrdr.readBad(br, hdr.Bad, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readConstraints(br, hdr.Constraint, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readJustice(br, hdr.Justice, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readFair(br, hdr.Fair, hdr.Max); err != nil {
		return nil, err
	}
	if err := aigrdr.readBinaryAnds(hdr, br); err != nil {
		return nil, err
	}
	if err := aigrdr.readSymsAndComments(br); err != nil {
		return nil, err
	}
	if err := aigrdr.commit(false); err != nil {
		return nil, err
	}
	return aigrdr.T, nil
}

// write the symbol table
func (a *T) writeSymtab(w *bufio.Writer) error {
	for k, _ := range a.symbols {
		for i, nm := range a.symbols[k] {
			if _, err := w.WriteString(fmt.Sprintf("%c%d %s\n", k, i, nm)); err != nil {
				return err
			}
		}
	}
	return w.Flush()
}

// writes a trailing comment saying that gini wrote the file
func writeComment(w *bufio.Writer) {
	w.WriteString("c\naiger file version 1.9 created by gini\n")
}

func (a *T) writeAsciiAnds(w *bufio.Writer) {
	// be nice and put them in topologic order
	dfs := newsDfs(a.S, func(s *logic.S, m z.Lit) {
		if s.Type(m) != logic.SAnd {
			return
		}
		writeLit(w, m, a.S.T)
		w.WriteString(" ")
		c0, c1 := s.Ins(m)
		writeLit(w, c0, a.S.T)
		w.WriteString(" ")
		writeLit(w, c1, a.S.T)
		w.WriteString("\n")
	})
	nexts := make([]z.Lit, 0, len(a.Latches))
	for _, m := range a.Latches {
		nexts = append(nexts, a.Next(m))
	}
	dfs.post(a.Outputs...)
	dfs.post(a.Bad...)
	dfs.post(a.Constraints...)
	for _, ma := range a.Justice {
		dfs.post(ma...)
	}
	dfs.post(a.Fair...)
	dfs.post(nexts...)
}

// state information for binary writer
type aigerBinWriter struct {
	trueLit   z.Lit
	w         *bufio.Writer
	firstPass bool
	id        uint
	idMap     []uint
}

// map literals from gini to aiger encoding for writer
func (abw *aigerBinWriter) mapLit(m z.Lit) {
	abw.idMap[int(m.Var())] = abw.id
	abw.id += 2
}

// get an aiger literal for a gini literal for writer
func (abw *aigerBinWriter) forLit(m z.Lit) uint {
	v := m.Var()
	a := abw.idMap[v]
	if a == 0 || m.IsPos() {
		return a
	}
	return a | 1
}

// implment DfsVis (2 passes, first pass maps and gates
// 2nd pass writes binary encoding using the mapping
// from the 1st pass)
func (abw *aigerBinWriter) writeBinAnd(s *logic.S, m z.Lit) {
	if s.Type(m) != logic.SAnd {
		return
	}
	// *logic.S stores c0 < c1, aiger
	// wants c0 > c1, so we swap
	// the assignment to c1, c0 :=
	c1, c0 := s.Ins(m)
	mc0 := abw.forLit(c0)
	mc1 := abw.forLit(c1)
	me := abw.forLit(m)
	delta0 := me - mc0
	delta1 := mc0 - mc1
	if delta0 <= 0 || delta1 <= 0 {
		panic(fmt.Sprintf("incorrect delta computation %s(%s,%s) d0 %d d1 %d mc0 %d mc1 %d\n", m, c1, c0, delta0, delta1, mc0, mc1))
	}
	write7(abw.w, delta0)
	write7(abw.w, delta1)
}

// data for aiger ands -- we need to keep a copy
// of this info to verify comb loops/ multiple defs
// etc.
type aigAnd struct {
	children [2]uint
	defined  bool
	mapped   bool
	dfsColor uint8
}

type aigerReader struct {
	*T
	// store the on-disk uints for translation
	// only after the ands have been translated,
	// since the *logic.S can optimize away some ands
	// resulting in a smaller *logic.S and a need to
	// store the mapping from aiger literals to
	// logic.S Literals explicitly
	AigInputs      []uint // only used in ascii reading
	AigLatches     []uint // only used in ascii reading
	AigLatchNexts  []uint
	AigOutputs     []uint
	AigBad         []uint
	AigConstraints []uint
	AigJustice     [][]uint
	AigFair        []uint
	varMap         []z.Var
	AigAnds        []aigAnd
}

func makeAigerReader(a *T, hdr *aigerHeader) *aigerReader {
	abr := &aigerReader{
		T:              a,
		AigInputs:      make([]uint, 0, hdr.In),
		AigLatches:     make([]uint, 0, hdr.Latch),
		AigLatchNexts:  make([]uint, 0, hdr.Latch),
		AigOutputs:     make([]uint, 0, hdr.Out),
		AigBad:         make([]uint, 0, hdr.Bad),
		AigConstraints: make([]uint, 0, hdr.Constraint),
		AigJustice:     make([][]uint, 0, hdr.Justice),
		AigFair:        make([]uint, 0, hdr.Fair),
		varMap:         make([]z.Var, hdr.Max+1),
		AigAnds:        nil}
	abr.varMap[0] = a.S.F.Var()
	return abr
}

func (abr *aigerReader) mapLit(aigerLit uint, m z.Lit) {
	abr.varMap[int(aigerLit>>1)] = m.Var()
}

func (abr *aigerReader) litFor(aigerLit uint) z.Lit {
	v := aigerLit >> 1
	rv := abr.varMap[v]
	if rv == 0 {
		return z.LitNull
	}
	if aigerLit&1 != 0 {
		return rv.Pos().Not()
	}
	return rv.Pos()
}

// once everything is read, we can use the aiglit to gini Literal
// mapping to add to the real Aiger object.  This method does that.
func (aigrdr *aigerReader) commit(ascii bool) error {
	offset := len(aigrdr.Inputs) + 1 // only used in binary (ascii=false)
	for i, u := range aigrdr.AigLatchNexts {
		var id uint
		if ascii {
			id = aigrdr.AigLatches[i]
		} else {
			id = uint(offset+i) * 2
		}
		m := aigrdr.litFor(id)
		n := aigrdr.litFor(u)
		if m == 0 || n == 0 {
			return UndefinedLit
		}
		aigrdr.SetNext(m, n)
	}
	for _, u := range aigrdr.AigOutputs {
		m := aigrdr.litFor(u)
		if m == z.LitNull {
			return UndefinedLit
		}
		aigrdr.T.Outputs = append(aigrdr.T.Outputs, m)
	}
	for _, u := range aigrdr.AigBad {
		m := aigrdr.litFor(u)
		if m == 0 {
			return UndefinedLit
		}
		aigrdr.Bad = append(aigrdr.Bad, m)
	}
	for _, u := range aigrdr.AigConstraints {
		m := aigrdr.litFor(u)
		if m == 0 {
			return UndefinedLit
		}
		aigrdr.Constraints = append(aigrdr.Constraints, m)
	}
	for i, ua := range aigrdr.AigJustice {
		for j, u := range ua {
			m := aigrdr.litFor(u)
			if m == 0 {
				return UndefinedLit
			}
			aigrdr.Justice[i][j] = m
		}
	}
	for _, u := range aigrdr.AigFair {
		m := aigrdr.litFor(u)
		if m == 0 {
			return UndefinedLit
		}
		aigrdr.Fair = append(aigrdr.Fair, m)
	}
	return nil
}

// each latch may optionally contain reset info on
// each latch line.  version 1.9 requires it, previous
// versions just zero latches initially, but it seems
// there are some examples which don't specify reset
// values
func (abr *aigerReader) readLatches(hdr *aigerHeader, br *bufio.Reader, ascii bool) error {
	var i uint
	for i = 0; i < hdr.Latch; i++ {
		var m z.Lit
		if ascii {
			latch, latchErr := readUint(br)
			if latchErr != nil {
				return latchErr
			}
			if latch&1 != 0 {
				return SignedLatch
			}
			abr.AigLatches = append(abr.AigLatches, latch)
			m = abr.S.Latch(abr.S.F)
			abr.mapLit(latch, m)
			b, e := br.ReadByte()
			if e == io.EOF {
				return PrematureEOF
			}
			if e != nil {
				return e
			}
			if b != ' ' {
				return UnexpectedChar
			}
		} else {
			m = abr.S.Latch(abr.S.F)
			abr.mapLit((hdr.In+i+1)*2, m)
		}
		nxt, errNxt := readUint(br)
		if errNxt != nil {
			return errNxt
		}
		if nxt > hdr.Max*2+1 {
			return LitOOB
		}
		abr.AigLatchNexts = append(abr.AigLatchNexts, nxt)
		b, e := br.ReadByte()
		if e == io.EOF {
			return PrematureEOF
		}
		if e != nil {
			return ReadError
		}
		if b == '\n' {
			abr.S.SetInit(m, abr.S.F)
			continue
		}
		if b == ' ' {
			ini, errIni := readUint(br)
			if errIni != nil {
				return errIni
			}
			if ini == 0 {
				abr.SetInit(m, abr.S.F)
			} else if ini == 1 {
				abr.SetInit(m, abr.S.T)
			} else if ini == (i+hdr.In+1)*2 {
				abr.SetInit(m, 0)
			} else {
				return InvalidLatchInit
			}
			if err := readNL(br); err != nil {
				return err
			}
			continue
		}
		return UnexpectedChar
	}
	return nil
}

func (aigrdr *aigerReader) readAsciiInputs(hdr *aigerHeader, r *bufio.Reader) error {
	var i uint
	for i = 0; i < hdr.In; i++ {
		in, err := readUint(r)
		if err != nil {
			return err
		}
		if in > hdr.Max*2+1 {
			return LitOOB
		}
		if in&1 != 0 {
			return SignedInput
		}
		m := aigrdr.S.Lit()
		aigrdr.Inputs = append(aigrdr.Inputs, m)
		aigrdr.mapLit(in, m)
		aigrdr.AigInputs = append(aigrdr.AigInputs, in)
		if err := readNL(r); err != nil {
			return err
		}
	}
	return nil
}

func (abr *aigerReader) readOutputs(nOut, max uint, r *bufio.Reader) error {
	abr.AigOutputs = make([]uint, 0, nOut)
	var i uint
	for i = 0; i < nOut; i++ {
		u, e := readUint(r)
		if e != nil {
			return e
		}
		if u > 2*max+1 {
			return LitOOB
		}
		abr.AigOutputs = append(abr.AigOutputs, u)
		if err := readNL(r); err != nil {
			return err
		}
	}
	return nil
}

func (abr *aigerReader) readBad(r *bufio.Reader, count, max uint) error {
	abr.AigBad = make([]uint, 0, int(count))
	abr.Bad = make([]z.Lit, 0, int(count))
	var i uint
	for i = 0; i < count; i++ {
		v, err := readUint(r)
		if err != nil {
			return err
		}
		if v > 2*max+1 {
			return LitOOB
		}
		abr.AigBad = append(abr.AigBad, v)
		if err := readNL(r); err != nil {
			return err
		}
	}
	return nil
}

func (a *aigerReader) readConstraints(r *bufio.Reader, count, max uint) error {
	a.AigConstraints = make([]uint, 0, int(count))
	a.Constraints = make([]z.Lit, 0, int(count))
	var i uint
	for i = 0; i < count; i++ {
		v, err := readUint(r)
		if err != nil {
			return err
		}
		if v > 2*max+1 {
			return LitOOB
		}
		a.AigConstraints = append(a.AigConstraints, v)
		if err := readNL(r); err != nil {
			return err
		}
	}
	return nil
}

func (a *aigerReader) readJustice(r *bufio.Reader, count, max uint) error {
	a.Justice = make([][]z.Lit, int(count), int(count))
	a.AigJustice = make([][]uint, 0, int(count))
	counts := make([]int, 0, int(count))
	var i uint
	for i = 0; i < count; i++ {
		v, err := readUint(r)
		if err != nil {
			return err
		}
		if v > 2*max+1 {
			return LitOOB
		}
		counts = append(counts, int(v))
		if err := readNL(r); err != nil {
			return err
		}
	}
	for i, c := range counts {
		a.AigJustice = append(a.AigJustice, make([]uint, c, c))
		a.Justice[i] = make([]z.Lit, c, c)
		for j := 0; j < c; j++ {
			v, err := readUint(r)
			if err != nil {
				return err
			}
			if v > 2*max+1 {
				return LitOOB
			}
			a.AigJustice[i][j] = v
			if err := readNL(r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *aigerReader) readFair(r *bufio.Reader, count, max uint) error {
	a.AigFair = make([]uint, 0, int(count))
	a.Fair = make([]z.Lit, 0, int(count))
	var i uint
	for i = 0; i < count; i++ {
		v, err := readUint(r)
		if err != nil {
			return err
		}
		if v > 2*max+1 {
			return LitOOB
		}
		a.AigFair = append(a.AigFair, v)
		if err := readNL(r); err != nil {
			return err
		}
	}
	return nil
}

func (aigrdr *aigerReader) readBinaryAnds(hdr *aigerHeader, r *bufio.Reader) error {
	id := (hdr.In + hdr.Latch + 1) * 2 // inputs, latches and constant
	var i uint
	for i = 0; i < hdr.And; i++ {
		delta0, err0 := read7(r)
		if err0 != nil {
			return err0
		}
		if delta0 > id {
			return BadDeltaEncoding
		}
		c0 := id - delta0
		delta1, err1 := read7(r)
		if err1 != nil {
			return err1
		}
		if delta1 > c0 {
			return BadDeltaEncoding
		}
		c1 := c0 - delta1
		m := aigrdr.And(aigrdr.litFor(c1), aigrdr.litFor(c0))
		aigrdr.mapLit(id, m)
		id += 2
	}
	return nil
}

func (aigrdr *aigerReader) readAsciiAnds(hdr *aigerHeader, r *bufio.Reader) error {
	aigrdr.AigAnds = make([]aigAnd, hdr.Max+1, hdr.Max+1)
	var i uint
	for i = 0; i < hdr.And; i++ {
		g, gErr := readUint(r)
		if gErr != nil {
			return gErr
		}
		if g > hdr.Max*2+1 {
			return LitOOB
		}
		if g&1 != 0 {
			return SignedAnd
		}
		// read ' '
		b, e := r.ReadByte()
		if e == io.EOF {
			return PrematureEOF
		}
		if e != nil {
			return e
		}
		if b != ' ' {
			return UnexpectedChar
		}

		// read first child
		c0, c0Err := readUint(r)
		if c0Err != nil {
			return c0Err
		}
		if c0 > hdr.Max*2+1 {
			return LitOOB
		}
		// read ' '
		b, e = r.ReadByte()
		if e == io.EOF {
			return PrematureEOF
		}
		if e != nil {
			return e
		}
		if b != ' ' {
			return UnexpectedChar
		}
		// read second child
		c1, c1Err := readUint(r)
		if c1Err != nil {
			return c1Err
		}
		if c1 > hdr.Max*2+1 {
			return LitOOB
		}
		// read end-of-line
		if err := readNL(r); err != nil {
			return err
		}
		// define the gate in terms of AigAnds
		aa := &aigrdr.AigAnds[int(g>>1)]
		if aa.defined {
			return AndMultiplyDefined
		}
		aa.defined = true
		aa.children[0] = c0
		aa.children[1] = c1
	}
	if err := aigrdr.mapAnds(); err != nil {
		return err
	}
	return nil
}

func (aigrdr *aigerReader) mapAnds() error {
	for _, m := range aigrdr.AigInputs {
		ag := &aigrdr.AigAnds[int(m>>1)]
		ag.defined = true
		ag.mapped = true
	}
	for _, m := range aigrdr.AigLatches {
		ag := &aigrdr.AigAnds[int(m>>1)]
		ag.defined = true
		ag.mapped = true
	}
	for i := 0; i < len(aigrdr.AigAnds); i++ {
		ag := &aigrdr.AigAnds[i]
		if ag.defined && !ag.mapped {
			if err := aigrdr.mapAndsRec(ag, uint(i*2)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (aigrdr *aigerReader) mapAndsRec(ag *aigAnd, aig uint) error {
	switch ag.dfsColor {
	case 0:
		ag.dfsColor = 1
		c0, c1 := ag.children[0], ag.children[1]
		ag0 := &aigrdr.AigAnds[int(c0>>1)]
		if !ag0.defined {
			return UndefinedLit
		}
		if !ag0.mapped {
			if err := aigrdr.mapAndsRec(ag0, c0); err != nil {
				return err
			}
		}
		m := aigrdr.litFor(c0)

		ag1 := &aigrdr.AigAnds[int(c1>>1)]
		if !ag1.defined {
			return UndefinedLit
		}
		if !ag1.mapped {
			if err := aigrdr.mapAndsRec(ag1, c1); err != nil {
				return err
			}
		}
		n := aigrdr.litFor(c1)
		aigrdr.mapLit(aig, aigrdr.And(m, n))
		ag.dfsColor = 2
		ag.mapped = true
	case 1:
		return CombLoop
	case 2:
	default:
		panic("unknown dfs color")
	}
	return nil
}

func (aigrdr *aigerReader) readSymsAndComments(r *bufio.Reader) error {
	for {
		b, e := r.ReadByte()
		if e == io.EOF {
			return nil
		}
		if b == 'i' || b == 'l' || b == 'o' || b == 'b' || b == 'c' || b == 'j' || b == 'f' {
			// symtab must precede comments
			if b == 'c' {
				bn, e := r.ReadByte()
				if e == io.EOF {
					return PrematureEOF
				}
				if e != nil {
					return e
				}
				if bn == '\n' {
					return aigrdr.readComments(r)
				}
				// if anything, it's a constraint symtab entry, not a comment.
				r.UnreadByte()
			}
			index, err := readUint(r)
			if err != nil {
				return err
			}
			bs, e := r.ReadByte()
			if e == io.EOF {
				return PrematureEOF
			}
			if e != nil {
				return e
			}
			if bs != ' ' {
				return UnexpectedChar
			}
			bytes, err := r.ReadBytes('\n')
			if err == io.EOF {
				return PrematureEOF
			}
			if err != nil {
				return err
			}
			aigrdr.symbols[b][int(index)] = string(bytes[0 : len(bytes)-1])
		}
	}
	panic("unreachable")
}

func (ar *aigerReader) readComments(r *bufio.Reader) error {
	for {
		comment, err := r.ReadString('\n')
		// TBD: add comments api
		if len(comment) > 0 {
			if err == io.EOF {
				return PrematureEOF
			}
			if err != nil {
				return err
			}
		} else if err == io.EOF {
			return nil
		}
	}
	panic("unreachable")
}

// header for aiger v 1.9
type aigerHeader struct {
	Binary     bool
	Max        uint
	In         uint
	Latch      uint
	Out        uint
	And        uint
	Bad        uint
	Constraint uint
	Justice    uint
	Fair       uint
}

// creates a header object from a system and an indication
// of whether or not the binary version is desired.
func makeHeader(a *T, binary bool) *aigerHeader {
	s := a.S
	N := s.Len()
	nAnd := uint(0)
	for i := 0; i < N; i++ {
		if s.Type(s.At(i)) == logic.SAnd {
			nAnd++
		}
	}
	return &aigerHeader{
		Binary:     binary,
		Max:        uint(a.Len() - 1),
		In:         uint(len(a.Inputs)),
		Latch:      uint(len(a.Latches)),
		Out:        uint(len(a.Outputs)),
		And:        nAnd,
		Bad:        uint(len(a.Bad)),
		Constraint: uint(len(a.Constraints)),
		Justice:    uint(len(a.Justice)),
		Fair:       uint(len(a.Fair))}
}

// write the header
func (h *aigerHeader) write(w *bufio.Writer) {
	if h.Binary {
		w.WriteString("aig ")
	} else {
		w.WriteString("aag ")
	}
	w.WriteString(fmt.Sprintf("%d %d %d %d %d %d %d %d %d\n",
		h.Max, h.In, h.Latch, h.Out, h.And, h.Bad, h.Constraint,
		h.Justice, h.Fair))
}

// read the header, possibly allowing version 1 style AIGER
// files (without B,C,J,F)
func readHeader(r *bufio.Reader) (*aigerHeader, error) {
	result := &aigerHeader{}
	buf := make([]byte, 0, 3)
	buf, err := readNonWS(r, buf)
	if err != nil {
		return nil, err
	}
	tok := string(buf)
	if tok == "aag" {
		result.Binary = false
	} else if tok == "aig" {
		result.Binary = true
	} else {
		return nil, BadHeader
	}
	wantSpace := true
	i := 0
	var counts [9]uint
	for {
		if !wantSpace {
			if i > 8 {
				return nil, BadHeader
			}
			counts[i], err = readUint(r)
			i++
			if err != nil {
				return nil, err
			}
			wantSpace = true
			continue
		}
		b, e := r.ReadByte()
		if e == io.EOF {
			return nil, PrematureEOF
		}
		if b == '\n' {
			if i < 5 {
				return nil, BadHeader
			}
			break
		}
		if b != ' ' {
			return nil, BadHeader
		}
		wantSpace = false
	}
	result.Max = counts[0]
	result.In = counts[1]
	result.Latch = counts[2]
	result.Out = counts[3]
	result.And = counts[4]
	result.Bad = counts[5]
	result.Constraint = counts[6]
	result.Justice = counts[7]
	result.Fair = counts[8]
	return result, nil
}

// read white space from the reader, discarding
// the read bytes.  Include newlines if newLine is true
func readWS(r *bufio.Reader, newLine bool) {
	for {
		b, e := r.ReadByte()
		if e == io.EOF {
			return
		}
		if b == ' ' || b == '\t' || b == '\r' {
			continue
		}
		if newLine && b == '\n' {
			continue
		}
		r.UnreadByte()
		return
	}
	panic("unreachable")
}

// reads a new line character and returns nil
// unless there was no new line character
func readNL(r *bufio.Reader) error {
	b, e := r.ReadByte()
	if e == io.EOF {
		return PrematureEOF
	}
	if e != nil {
		return e
	}
	if b == '\n' {
		return nil
	}
	os.Stdout.Sync()
	return UnexpectedChar
}

// reads non-white space and puts the result in buf
func readNonWS(r *bufio.Reader, buf []byte) ([]byte, error) {
	buf = buf[:0]
	for {
		b, e := r.ReadByte()
		if e == io.EOF {
			break
		}
		if e != nil {
			return buf, e
		}
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			r.UnreadByte()
			break
		}
		buf = append(buf, b)
	}
	return buf, nil
}

// reads a uint
func readUint(r *bufio.Reader) (uint, error) {
	var result uint = 0
	first := true
	for {
		b, e := r.ReadByte()
		if e == io.EOF {
			if first {
				return 0, PrematureEOF
			}
		}
		if e != nil {
			return 0, e
		}
		if b >= '0' && b <= '9' {
			result *= 10
			result += uint(b - '0')
			first = false
			continue
		}
		r.UnreadByte()
		break
	}
	if first {
		return 0, UnexpectedChar
	}
	return result, nil
}

// write a literal in AIGER style (modulo 2 gives pos/neg)
func writeLit(w *bufio.Writer, m, t z.Lit) error {
	if m == t {
		_, err := w.WriteString("0")
		return err
	}
	if m == t.Not() {
		_, err := w.WriteString("1")
		return err
	}
	u := m - 2
	_, err := w.WriteString(fmt.Sprintf("%d", uint(u)))
	return err
}

// for binary aiger coding of and deltas
func write7(w *bufio.Writer, val uint) error {
	for val != 0 {
		b := byte(val & 0x7f)
		val = val >> 7
		if val != 0 {
			b |= 0x80
		}
		err := w.WriteByte(b)
		if err != nil {
			return err
		}
	}
	return nil
}

// for binary aiger coding of and deltas
func read7(r *bufio.Reader) (result uint, err error) {
	var i int = 0
	for {
		b, e := r.ReadByte()
		if e == io.EOF {
			return 0, PrematureEOF
		}
		result |= (uint(b) & 0x7f) << uint8(7*i)
		i++
		if b&0x80 == 0 {
			break
		}
	}
	return
}
