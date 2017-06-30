package split

import (
	"fmt"
	"io"

	"github.com/pbtrung/scat"
	"github.com/restic/chunker"
)

const (
	pol        = chunker.Pol(0x3DA3358B4DC173)
	minMin     = 512 * 1024 // chunker.chunkerBufSize
	defaultMin = chunker.MinSize
	defaultMax = chunker.MaxSize
)

type splitter struct {
	chunker *chunker.Chunker
	buf     []byte
	num     int // int for use as slice index
	chunk   *scat.Chunk
	err     error
}

func NewSplitter(num int, r io.Reader) scat.ChunkIter {
	return NewSplitterSize(num, r, defaultMin, defaultMax)
}

func NewSplitterSize(num int, r io.Reader, min, max uint) scat.ChunkIter {
	if min < minMin {
		panic(fmt.Errorf("min size must be >= %d bytes", minMin))
	}
	chunker := chunker.NewWithBoundaries(r, pol, min, max)
	return &splitter{
		chunker: chunker,
		buf:     make([]byte, max),
		num:     num,
	}
}

func (s *splitter) Next() bool {
	c, err := s.chunker.Next(s.buf)
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		s.err = err
		return false
	}
	data := make(scat.BytesData, len(c.Data))
	copy(data, c.Data)
	s.chunk = scat.NewChunk(s.num, data)
	s.chunk.SetTargetSize(len(data))
	s.num++
	// Check for overflow: uint resets to 0, int resets to -minInt
	if s.num <= 0 {
		panic("overflow")
	}
	return true
}

func (s *splitter) Chunk() *scat.Chunk {
	return s.chunk
}

func (s *splitter) Err() error {
	return s.err
}
