package argproc_test

import (
	"testing"

	"github.com/pbtrung/scat/argproc"
)

func TestNew(t *testing.T) {
	// just test that it compiles
	argproc.New(nil, nil)
}
