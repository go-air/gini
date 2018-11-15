package xo

import "github.com/irifrance/gini/z"

type Active struct {
	Free     []z.Lit
	Occs     [][]z.C
	IsActive []bool
}

func newActive(vcap int) *Active {
	return &Active{
		Occs:     make([][]z.C, vcap),
		IsActive: make([]bool, vcap)}
}

func (a *Active) Activate(s *S) z.Lit {
	var act z.Lit = z.LitNull
	n := len(a.Free)

	if n != 0 {
		act = a.Free[n-1]
		a.Free = a.Free[:n-1]
	} else {
		act = s.Lit()
	}
	a.IsActive[act.Var()] = true
	s.Add(act.Not())

	loc, u := s.Cdb.Add(0)
	if u != z.LitNull {
		panic("activated empty clause")
	}

	// add occ
	if loc != CInf {
		a.Occs[act.Var()] = []z.C{loc}
	}
	return act
}

func (a *Active) Deactivate(cdb *Cdb, m z.Lit) {
	mv := m.Var()
	m = mv.Pos()
	sl := a.Occs[mv]
	a.Occs[mv] = nil
	cdb.Remove(sl...) // this might trigger CRemap below, so we update occs first.
	a.Free = append(a.Free, m)
	a.IsActive[mv] = false
}

func (a *Active) CRemap(rlm map[z.C]z.C) {
	for i := range a.Occs {
		sl := a.Occs[i]
		j := 0
		for _, c := range sl {
			d, ok := rlm[c]
			if d == CNull {
				continue
			}
			if !ok {
				sl[j] = c
				j++
				continue
			}
			sl[j] = d
			j++
		}
		a.Occs[i] = sl[:j]
	}
}

func (a *Active) growToVar(u z.Var) {
	w := u + 1
	oc := make([][]z.C, w)
	copy(oc, a.Occs)
	a.Occs = oc

	ia := make([]bool, w)
	copy(ia, a.IsActive)
	a.IsActive = ia
}

func (a *Active) Copy() *Active {
	res := &Active{
		Occs:     make([][]z.C, len(a.Occs), cap(a.Occs)),
		IsActive: make([]bool, len(a.IsActive), cap(a.IsActive))}
	copy(res.IsActive, a.IsActive)
	for i, asl := range a.Occs {
		rsl := make([]z.C, len(asl), cap(asl))
		copy(rsl, asl)
		res.Occs[i] = rsl
	}
	return res
}
