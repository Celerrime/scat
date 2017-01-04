package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"secsplit/aprocs"
	"secsplit/cpprocs"
	"secsplit/cpprocs/mincopies"
	"secsplit/index"
	"secsplit/procs"
	"secsplit/split"
	"secsplit/stats"
)

func main() {
	if err := start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func start() error {
	rand.Seed(time.Now().UnixNano())
	args := os.Args[1:]
	if len(args) != 1 {
		return errors.New("usage: split|join")
	}
	cmd := args[0]
	switch cmd {
	case "split":
		return cmdSplit()
	case "join":
		return cmdJoin()
	}
	return fmt.Errorf("unknown cmd: %s", cmd)
}

const (
	ndata   = 2
	nparity = 1
)

func cmdSplit() (err error) {
	in, out := os.Stdin, os.Stdout

	log := stats.NewLog(os.Stderr, 250*time.Millisecond)
	// log := stats.NewLog(ioutil.Discard, 250*time.Millisecond)

	parity, err := aprocs.NewParity(ndata, nparity)
	if err != nil {
		return
	}

	cats := make([]cpprocs.Copier, 3)
	for i, n := 0, len(cats); i < n; i++ {
		id := fmt.Sprintf("cat%d", i+1)
		cats[i] = cpprocs.NewCopier(id, stats.NewLsProc(log, id,
			cpprocs.NewCommand(cpprocs.NewCat("/Users/roman/tmp/"+id)).LsProc(),
		))
	}
	cats[0].SetQuota(10 * 1024 * 1024)
	minCopies, err := mincopies.New(2, cats)
	if err != nil {
		return
	}

	chain := aprocs.NewBacklog(4, aprocs.NewChain([]aprocs.Proc{
		stats.NewProc(log, "checksum",
			procs.A(procs.Checksum{}.Proc()),
		),
		stats.NewProc(log, "size",
			procs.A(procs.Size),
		),
		stats.NewProc(log, "index",
			aprocs.NewIndex(out),
		),
		stats.NewProc(log, "parity",
			parity.Proc(),
		),
		stats.NewProc(log, "compress",
			procs.A((&procs.Compress{}).Proc()),
		),
		stats.NewProc(log, "checksum2",
			procs.A(procs.Checksum{}.Proc()),
		),
		// stats.NewProc(log, "localstore",
		// 	procs.A((&procs.LocalStore{"out"}).Proc()),
		// ),
		aprocs.NewConcur(2, minCopies),
	}))
	defer chain.Finish()

	splitter := split.NewSplitter(in)
	err = aprocs.Process(chain, splitter)
	if err != nil {
		return
	}
	return chain.Finish()
}

func cmdJoin() (err error) {
	in, out := os.Stdin, os.Stdout

	log := stats.NewLog(os.Stderr, 250*time.Millisecond)

	parity, err := aprocs.NewParity(ndata, nparity)
	if err != nil {
		return
	}

	cats := make([]cpprocs.LsProc, 3)
	for i, n := 0, len(cats); i < n; i++ {
		id := fmt.Sprintf("cat%d", i+1)
		cats[i] = stats.NewLsProc(log, id,
			cpprocs.NewCommand(cpprocs.NewCat("/Users/roman/tmp/"+id)).LsUnproc(),
		)
	}

	chain := aprocs.NewBacklog(2, aprocs.NewChain([]aprocs.Proc{
		stats.NewProc(log, "localstore",
			procs.A((&procs.LocalStore{"out"}).Unproc()),
		),
		stats.NewProc(log, "pool",
			aprocs.NewPool(2, aprocs.NewChain([]aprocs.Proc{
				stats.NewProc(log, "checksum",
					procs.A(procs.Checksum{}.Unproc()),
				),
				stats.NewProc(log, "compress",
					procs.A((&procs.Compress{}).Unproc()),
				),
				stats.NewProc(log, "group",
					aprocs.NewGroup(ndata+nparity),
				),
				stats.NewProc(log, "parity",
					parity.Unproc(),
				),
			})),
		),
		aprocs.NewMutex(aprocs.NewChain([]aprocs.Proc{
			stats.NewProc(log, "sort",
				aprocs.NewSort(),
			),
			stats.NewProc(log, "writerto",
				aprocs.NewWriterTo(out),
			),
		})),
	}))

	scan := index.NewScanner(in)
	err = aprocs.Process(chain, scan)
	if err != nil {
		return
	}
	return chain.Finish()
}
