package procs

import (
	"github.com/pbtrung/scat"
	"github.com/pbtrung/scat/split"
)

var Split Proc

func init() {
	Split = ChunkIterFunc(func(c *scat.Chunk) scat.ChunkIter {
		return split.NewSplitter(c.Num(), c.Data().Reader())
	})
}

func NewSplitSize(min, max uint) Proc {
	return ChunkIterFunc(func(c *scat.Chunk) scat.ChunkIter {
		return split.NewSplitterSize(c.Num(), c.Data().Reader(), min, max)
	})
}
