package argparse_test

import (
	"strconv"
	"testing"

	"github.com/Roman2K/scat/argparse"
	assert "github.com/stretchr/testify/require"
)

func TestArgBytes(t *testing.T) {
	str := "1kib"
	i, n, err := argparse.ArgBytes.Parse(str)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1024), i)
	assert.Equal(t, 4, n)

	str = "1kib "
	i, n, err = argparse.ArgBytes.Parse(str)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1024), i)
	assert.Equal(t, 4, n)

	str = " 1kib"
	_, _, err = argparse.ArgBytes.Parse(str)
	assert.Error(t, err)
	assert.IsType(t, &strconv.NumError{}, err)
}
