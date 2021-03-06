package integration_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"math/rand"
	"runtime"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
	"github.com/pbtrung/scat"
	"github.com/pbtrung/scat/procs"
	"github.com/pbtrung/scat/stores"
	"github.com/pbtrung/scat/stores/quota"
	storestripe "github.com/pbtrung/scat/stores/stripe"
	"github.com/pbtrung/scat/stripe"
	"github.com/pbtrung/scat/testutil"
)

const mb = 1024 * 1024

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestRecoveryRaid5(t *testing.T) {
	const (
		ndata    = 3
		nparity  = 1
		nshards  = ndata + nparity
		splitMin = 1 * mb
		splitMax = splitMin
		dataSize = splitMin * 10
	)

	parity, err := procs.NewParity(ndata, nparity)
	assert.NoError(t, err)

	hashIn := sha256.New()
	data := make(scat.BytesData, dataSize)
	_, err = rand.Read(data)
	assert.NoError(t, err)
	_, err = hashIn.Write(data)
	assert.NoError(t, err)

	copiers := make([]stores.Copier, nshards)
	readers := make([]stores.Copier, nshards)
	resetStores := func() {
		for i := range copiers {
			store := stores.NewMem()
			copiers[i] = stores.Copier{i, store, store.Proc()}
			readers[i] = stores.Copier{i, store, store.Unproc()}
		}
	}

	write := func(cfg stripe.Config) []byte {
		indexBuf := &bytes.Buffer{}
		qman := quota.NewMan()
		for _, cp := range copiers {
			qman.AddRes(cp)
		}
		stripep, err := storestripe.New(cfg, qman)
		assert.NoError(t, err)
		proc := procs.Chain{
			procs.NewSplitSize(splitMin, splitMax),
			procs.NewBacklog(runtime.NumCPU(), procs.Chain{
				procs.ChecksumProc,
				procs.NewIndexProc(indexBuf),
				parity.Proc(),
				procs.ChecksumProc,
				procs.NewGroup(nshards),
				procs.NewConcur(runtime.NumCPU(), stripep),
			}),
		}
		seed := scat.NewChunk(0, data)
		ch := proc.Process(seed)
		_, err = testutil.ReadChunks(ch)
		assert.NoError(t, err)
		return indexBuf.Bytes()
	}
	read := func(index []byte) hash.Hash {
		mrd, err := stores.NewMultiReader(readers)
		assert.NoError(t, err)
		hashOut := sha256.New()
		proc := procs.Chain{
			procs.IndexUnproc,
			procs.NewBacklog(runtime.NumCPU(), procs.Chain{
				mrd,
				procs.ChecksumUnproc,
				procs.NewGroup(nshards),
				parity.Unproc(),
				procs.NewJoin(hashOut),
			}),
		}
		seed := scat.NewChunk(0, scat.BytesData(index))
		ch := proc.Process(seed)
		_, err = testutil.ReadChunks(ch)
		assert.NoError(t, err)
		return hashOut
	}
	hex := func(h hash.Hash) string {
		sum := [sha256.Size]byte{}
		h.Sum(sum[:0])
		return fmt.Sprintf("%x", sum)
	}
	empty := func(cp stores.Copier) {
		store := cp.Lister.(*stores.Mem)
		for _, h := range store.Hashes() {
			store.Delete(h)
		}
	}

	resetStores()

	// sanity check
	index := write(stripe.Config{Min: 1, Excl: 0})
	assert.Equal(t, hex(hashIn), hex(read(index)))

	resetStores()

	// RAID 5: delete 1 whole store
	for _, i := range rand.Perm(len(copiers)) {
		index = write(stripe.Config{Min: 1, Excl: ndata})
		assert.Equal(t, hex(hashIn), hex(read(index)))
		empty(copiers[i])
		assert.Equal(t, hex(hashIn), hex(read(index)))
	}

	resetStores()

	// RAID 1+5: delete 2 whole stores
	for _, i := range rand.Perm(len(copiers)) {
		index = write(stripe.Config{Min: 2, Excl: nshards})
		assert.Equal(t, hex(hashIn), hex(read(index)))
		empty(copiers[i])
		i = (i + 1) % len(copiers)
		empty(copiers[i])
		assert.Equal(t, hex(hashIn), hex(read(index)))
	}
}
