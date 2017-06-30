package stats_test

import (
	"testing"

	"github.com/pbtrung/scat/procs"
	"github.com/pbtrung/scat/stats"
	"github.com/pbtrung/scat/testutil"
)

func TestProcFinish(t *testing.T) {
	statsd := stats.New()
	testutil.TestFinishErrForward(t, func(proc procs.Proc) testutil.Finisher {
		return stats.Proc{statsd, nil, proc}
	})
}
