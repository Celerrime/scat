package procs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/klauspost/reedsolomon"
	"github.com/pbtrung/scat"
)

type parity struct {
	enc     reedsolomon.Encoder
	nshards int
}

func NewParity(ndata, nparity int) (p ProcUnprocer, err error) {
	enc, err := reedsolomon.New(ndata, nparity)
	p = &parity{
		enc:     enc,
		nshards: ndata + nparity,
	}
	return
}

func (p *parity) Proc() Proc {
	return ProcFunc(p.process)
}

func (p *parity) process(c *scat.Chunk) <-chan Res {
	sz, ok := c.Data().(scat.Sizer)
	if !ok {
		err := errors.New("sized-data required to determine target size")
		return SingleRes(c, err)
	}
	ch := make(chan Res)
	go func() {
		defer close(ch)
		shards, err := p.split(c)
		if err != nil {
			ch <- Res{Chunk: c, Err: err}
			return
		}
		for i, shard := range shards {
			chunk := scat.NewChunk(c.Num()*p.nshards+i, scat.BytesData(shard))
			chunk.SetTargetSize(sz.Size())
			ch <- Res{Chunk: chunk}
		}
	}()
	return ch
}

func (p *parity) split(c *scat.Chunk) (shards [][]byte, err error) {
	bytes, err := c.Data().Bytes()
	if err != nil {
		return
	}
	shards, err = p.enc.Split(bytes)
	if err != nil {
		return
	}
	err = p.enc.Encode(shards)
	return
}

func (p *parity) Unproc() Proc {
	return ChunkFunc(p.unprocess)
}

func (p *parity) unprocess(c *scat.Chunk) (new *scat.Chunk, err error) {
	data, err := p.join(c)
	new = c.WithData(scat.BytesData(data))
	return
}

func (p *parity) join(c *scat.Chunk) (joined []byte, err error) {
	// Shard chunks
	chunks, err := getGroup(c, p.nshards)
	if err != nil {
		return
	}

	// Shards slice
	shards := make([][]byte, len(chunks))
	mustReconstruct := false
	for i, c := range chunks {
		err, ok := parityRecoverableErr(c)
		if ok {
			mustReconstruct = true
			logParityRecoverableErr(err, c)
			continue
		}
		if err != nil {
			return nil, err
		}
		bytes, err := c.Data().Bytes()
		if err != nil {
			return nil, err
		}
		shards[i] = bytes
	}

	// Reconstruct invalid shards
	if mustReconstruct {
		err = p.enc.Reconstruct(shards)
		if err != nil {
			return
		}
	}

	// Verify integrity
	ok, err := p.enc.Verify(shards)
	if err == nil && !ok {
		err = errors.New("verification failed")
	}
	if err != nil {
		return
	}

	// Join data shards, trim trailing padding
	out := bytes.NewBuffer(make([]byte, 0, c.TargetSize()))
	err = p.enc.Join(out, shards, out.Cap())
	joined = out.Bytes()
	return
}

func getGroup(c *scat.Chunk, size int) (chunks []*scat.Chunk, err error) {
	chunks, ok := GetGroup(c)
	switch {
	case !ok:
		err = errors.New("missing group")
	case len(chunks) != size:
		err = errors.New("invalid group size")
	}
	return
}

func parityRecoverableErr(c *scat.Chunk) (err error, ok bool) {
	err, ok = GetGroupErr(c)
	if !ok {
		return
	}
	if _, isMiss := err.(MissingDataError); isMiss {
		ok = true
		return
	}
	ok = (err == ErrIntegrityCheckFailed)
	return
}

func logParityRecoverableErr(err error, c *scat.Chunk) {
	var stderr []byte
	if exit, ok := err.(*exec.ExitError); ok {
		stderr = exit.Stderr
	}
	fmt.Fprintf(os.Stderr,
		"parity: recovering err: %v (stderr=%q chunk=%x)\n",
		err, stderr, c.Hash(),
	)
}
