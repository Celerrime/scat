package cpprocs

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"secsplit/checksum"
)

type cat struct {
	dir string
}

func NewCat(dir string) CmdSpawner {
	return cat{dir: dir}
}

func (cat cat) NewProcCmd(hash checksum.Hash) (cmd *exec.Cmd, err error) {
	path := cat.filePath(hash)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	cmd = exec.Command("cat")
	cmd.Stdout = f
	return
}

func (cat cat) NewUnprocCmd(hash checksum.Hash) (*exec.Cmd, error) {
	path := cat.filePath(hash)
	return exec.Command("cat", path), nil
}

func (cat cat) filePath(hash checksum.Hash) string {
	return filepath.Join(cat.dir, fmt.Sprintf("%x", hash))
}

func (cat cat) Ls() (entries []LsEntry, err error) {
	files, err := ioutil.ReadDir(cat.dir)
	if err != nil {
		return
	}
	var (
		buf   []byte
		entry LsEntry
	)
	for _, f := range files {
		n, err := fmt.Sscanf(f.Name(), "%x", &buf)
		if err != nil || n != 1 {
			continue
		}
		err = entry.Hash.LoadSlice(buf)
		if err != nil {
			continue
		}
		fi, err := os.Stat(filepath.Join(cat.dir, f.Name()))
		if err != nil {
			return nil, err
		}
		entry.Size = fi.Size()
		entries = append(entries, entry)
	}
	return
}
