package racewalk

import (
	"os"

	"golang.org/x/sys/unix"
)

// FileNode is a superset of os.FileInfo
type FileNode struct {
	os.FileInfo
	unix.Stat_t
}
