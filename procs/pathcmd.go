package procs

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/pbtrung/scat"
	"github.com/pbtrung/scat/tmpdedup"
)

type pathCmdIn struct {
	newCmd PathCmdInFn
	tmp    *tmpdedup.Dir
}

type PathCmdInFn func(*scat.Chunk, string) (*exec.Cmd, error)

func NewPathCmdIn(newCmd PathCmdInFn, tmp *tmpdedup.Dir) Proc {
	return InplaceFunc(pathCmdIn{newCmd, tmp}.process)
}

func (cmdp pathCmdIn) process(c *scat.Chunk) error {
	filename := fmt.Sprintf("%x", c.Hash())
	path, wg, err := cmdp.tmp.Get(filename, func(path string) (err error) {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		_, err = io.Copy(f, c.Data().Reader())
		return
	})
	if err != nil {
		return err
	}
	defer wg.Done()
	cmd, err := cmdp.newCmd(c, path)
	if err != nil {
		return err
	}
	return cmd.Run()
}
