package racewalk

import (
	"os"
	"path/filepath"
	"syscall"
)

// FileNode is a superset of os.FileInfo
type FileNode struct {
	os.FileInfo

	statPath string
}

// complete takes a string and a FileInfo, and returns a FileNode. The string,
// 'top', is a path to the directory containing the FileInfo.
func complete(top string, finfo os.FileInfo) *FileNode {
	fileNode := FileNode{
		FileInfo: finfo,
		statPath: filepath.Join(top, finfo.Name()),
	}
	return &fileNode
}

func (fileNode FileNode) Stat() *syscall.Stat_t {
	stat, ok := fileNode.FileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}

	return stat
}
