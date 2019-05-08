package racewalk

import (
	"os"
	"path/filepath"
	"syscall"
)

// FileNode is a superset of os.FileInfo
type FileNode struct {
	os.FileInfo

	// StatPath is the path that was used to os.Lstat() the file
	StatPath string
}

// complete takes a string and a FileInfo, and returns a FileNode. The string,
// 'top', is a path to the directory containing the FileInfo.
func complete(top string, finfo os.FileInfo) *FileNode {
	fileNode := FileNode{
		FileInfo: finfo,
		StatPath: filepath.Join(top, finfo.Name()),
	}
	return &fileNode
}

// GetStat returns the *syscall.Stat_t buffer
// If a unix Stat_t cannot be given, a nil is returned
func (fileNode FileNode) GetStat() *syscall.Stat_t {
	switch stat := fileNode.Sys().(type) {
	case *syscall.Stat_t:
		return stat
	}

	return nil
}
