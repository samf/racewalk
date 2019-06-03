package racewalk

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	dirnodes = []string{
		"top",
		"top/empty",
		"top/non-empty",
		"top/non-empty/bottom",
	}
	unreadable = []string{
		"top/unreadable",
		"top/non-empty/unreadable",
	}
	regnodes = []struct {
		name string
		size int64
	}{
		{
			name: "top/non-empty/empty",
		},
		{
			name: "top/little",
			size: 1,
		},
		{
			name: "top/bigger",
			size: 8192,
		},
		{
			name: "top/non-empty/medium",
			size: 15,
		},
		{
			name: "top/non-empty/bottom/feeder",
			size: 255,
		},
	}
	symlinks = []struct {
		name   string
		target string
	}{
		{
			name:   "top/detached",
			target: "frogs",
		},
		{
			name:   "top/non-empty/bottom/loopy",
			target: "../..",
		},
	}
)

func setup() string {
	dirname := fmt.Sprintf("%v-%v", time.Now().Unix(), os.Getpid())
	err := os.Mkdir(dirname, 0777)
	if err != nil {
		panic(err)
	}
	err = os.Chdir(dirname)
	if err != nil {
		panic(err)
	}

	for _, dir := range dirnodes {
		err := os.Mkdir(dir, 0777)
		if err != nil {
			panic(err)
		}
	}

	for _, reg := range regnodes {
		buffy := make([]byte, reg.size)
		err := ioutil.WriteFile(reg.name, buffy, 0666)
		if err != nil {
			panic(err)
		}
	}

	for _, link := range symlinks {
		err := os.Symlink(link.target, link.name)
		if err != nil {
			panic(err)
		}
	}

	err = os.Chdir("..")
	if err != nil {
		panic(err)
	}

	return dirname
}

func setupUnreadable(dirname string) {
	err := os.Chdir(dirname)
	if err != nil {
		panic(err)
	}

	for _, darkDir := range unreadable {
		err := os.Mkdir(darkDir, 0000)
		if err != nil {
			panic(err)
		}
	}

	err = os.Chdir("..")
	if err != nil {
		panic(err)
	}
}

func cleanupUnreadable(dirname string) {
	err := os.Chdir(dirname)
	if err != nil {
		panic(err)
	}

	for _, dark := range unreadable {
		err := os.Chmod(dark, 0777)
		if err != nil {
			panic(err)
		}
	}

	for _, dark := range unreadable {
		err := os.Remove(dark)
		if err != nil {
			panic(err)
		}
	}

	err = os.Chdir("..")
	if err != nil {
		panic(err)
	}
}

func cleanup(dirname string) {
	err := os.RemoveAll(dirname)
	if err != nil {
		panic(err)
	}
}

func TestWalk(t *testing.T) {
	dirname := setup()
	defer cleanup(dirname)

	t.Run("happy path", func(t *testing.T) {
		assert := assert.New(t)

		err := Walk(dirname, nil, func(path string,
			dirs, others []FileNode) ([]FileNode, error) {
			return dirs, nil
		})
		assert.NoError(err)
	})

	t.Run("get everything", func(t *testing.T) {
		var visited int32
		assert := assert.New(t)

		// add one for dirname itself
		created := int32(1 + len(dirnodes) + len(regnodes) + len(symlinks))

		err := Walk(dirname, nil, func(path string,
			subdirs, entries []FileNode) ([]FileNode, error) {
			atomic.AddInt32(&visited, int32(len(entries)))

			return subdirs, nil
		})

		assert.NoError(err)
		assert.Equal(created, visited)
	})

	t.Run("get only the top dir", func(t *testing.T) {
		var count int32
		assert := assert.New(t)

		err := Walk(dirname, nil, func(path string,
			subdirs, entries []FileNode) ([]FileNode, error) {
			atomic.AddInt32(&count, int32(len(entries)))
			return []FileNode{}, nil
		})

		assert.NoError(err)
		assert.Equal(int32(1), count)
	})

	t.Run("handler errs", func(t *testing.T) {
		errstr := "uh oh"
		assert := assert.New(t)

		err := Walk(dirname, nil, func(path string,
			dirs, others []FileNode) ([]FileNode, error) {
			return dirs, fmt.Errorf(errstr)
		})

		assert.Error(err)
		assert.Equal(err.Error(), errstr)
	})

	t.Run("error handler", func(t *testing.T) {
		var count int

		assert := assert.New(t)

		setupUnreadable(dirname)
		defer cleanupUnreadable(dirname)

		opt := &Options{
			ErrHandler: func(path string, err error) error {
				count++
				return nil
			},
		}
		err := Walk(dirname, opt, func(path string,
			dirs, others []FileNode) ([]FileNode, error) {
			return dirs, nil
		})
		assert.NoError(err)
		assert.Equal(2, count)
	})

	t.Run("good data", func(t *testing.T) {
		var data sync.Map
		assert := assert.New(t)
		require := require.New(t)

		err := Walk(dirname, nil, func(path string,
			subdirs, entries []FileNode) ([]FileNode, error) {
			for _, thing := range entries {
				key := path
				if !thing.IsDir() {
					key = filepath.Join(path, thing.Name())
				}
				slash := strings.IndexRune(key, '/')
				if slash != -1 {
					key = key[slash+1:]
				}
				_, ok := data.LoadOrStore(key, thing)
				assert.False(ok)
			}

			return subdirs, nil
		})

		assert.NoError(err)
		for _, dinfo := range dirnodes {
			inode, ok := data.Load(dinfo)
			require.True(ok, dinfo)
			node, ok := inode.(FileNode)
			require.True(ok, dinfo)
			assert.True(node.IsDir())
		}

		for _, finfo := range regnodes {
			inode, ok := data.Load(finfo.name)
			require.True(ok)
			node, ok := inode.(FileNode)
			require.True(ok)
			assert.False(node.IsDir())
			assert.Equal(finfo.size, node.FileInfo.Size())
		}

		for _, linkinfo := range symlinks {
			inode, ok := data.Load(linkinfo.name)
			require.True(ok)
			node, ok := inode.(FileNode)
			require.True(ok)
			assert.Equal(uint16(syscall.S_IFLNK),
				node.GetStat().Mode&syscall.S_IFMT)
		}
	})
}
