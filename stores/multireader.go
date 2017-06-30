package stores

import (
	"errors"
	"fmt"
	"os"

	"github.com/pbtrung/scat"
	"github.com/pbtrung/scat/concur"
	"github.com/pbtrung/scat/procs"
	"github.com/pbtrung/scat/stores/copies"
)

var (
	errMultiReaderNoneAvail = errors.New("no readers available")
)

type mrd struct {
	reg     *copies.Reg
	copiers []Copier
}

func NewMultiReader(copiers []Copier) (proc procs.Proc, err error) {
	ml := make(MultiLister, len(copiers))
	for i, cp := range copiers {
		ml[i] = cp
	}
	reg := copies.NewReg()
	proc = mrd{
		reg:     reg,
		copiers: copiers,
	}
	err = ml.AddEntriesTo([]LsEntryAdder{
		CopiesEntryAdder{Reg: reg},
	})
	return
}

var shuffle = ShuffleCopiers // var for tests

func (mrd mrd) Process(c *scat.Chunk) <-chan procs.Res {
	owners := mrd.reg.List(c.Hash()).Owners()
	copiers := make([]Copier, len(owners))
	for i, o := range owners {
		copiers[i] = o.(Copier)
	}
	copiers = shuffle(copiers)
	casc := make(procs.Cascade, len(copiers))
	for i, cp := range copiers {
		casc[i] = procs.OnEnd{cp, func(err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "multireader: copier error: %v\n", err)
				mrd.reg.RemoveOwner(cp)
			}
		}}
	}
	if len(casc) == 0 {
		return procs.SingleRes(c, procs.MissingDataError{errMultiReaderNoneAvail})
	}
	return casc.Process(c)
}

func (mrd mrd) Finish() error {
	return finishFuncs(mrd.copiers).FirstErr()
}

func finishFuncs(copiers []Copier) (fns concur.Funcs) {
	fns = make(concur.Funcs, len(copiers))
	for i, cp := range copiers {
		fns[i] = cp.Finish
	}
	return
}
