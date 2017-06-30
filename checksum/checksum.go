package checksum

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/lhecker/argon2"
)

const Size = sha512.Size

type Hash [Size]byte

func (h *Hash) LoadSlice(s []byte) error {
	if len(s) != Size {
		return errors.New("invalid hash length")
	}
	copy(h[:], s)
	return nil
}

func Sum(rd io.Reader) (cks Hash, err error) {
	hash := sha512.New()
	_, err = io.Copy(hash, rd)
	if err != nil {
		return
	}
	hash.Sum(cks[:0])

	cfg := argon2.DefaultConfig()
	cfg.HashLength = Size
	cfg.TimeCost = 4
	cfg.MemoryCost = 1 << 16
	cfg.Parallelism = 2
	raw, err := cfg.Hash(cks[0:Size*3/4], cks[Size*3/4:])
	if err != nil {
		fmt.Printf("Error during hashing: %s\n", err.Error())
		os.Exit(1)
	}
	copy(cks[:], raw.Hash)

	return
}

func SumBytes(b []byte) Hash {
	return sha512.Sum512(b)
}
