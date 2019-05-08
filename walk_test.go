package racewalk

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

	t.Run("good data", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)
		data := make(map[string]FileNode)

		err := Walk(dirname, nil, func(path string,
			dirs, others []FileNode) ([]FileNode, error) {
			strip := len(dirname) + 1
			if len(path) <= strip {
				path = "."
			} else {
				path = path[strip:]
			}
			for _, dir := range dirs {
				key := filepath.Join(path, dir.Name())
				data[key] = dir
			}
			for _, thing := range others {
				key := filepath.Join(path, thing.Name())
				data[key] = thing
			}

			return dirs, nil
		})

		assert.NoError(err)
		for _, dinfo := range dirnodes {
			node, ok := data[dinfo]
			require.True(ok, dinfo)
			assert.True(node.IsDir())
		}

		for _, finfo := range regnodes {
			node, ok := data[finfo.name]
			require.True(ok)
			assert.False(node.IsDir())
			assert.Equal(finfo.size, node.FileInfo.Size())
		}

		for _, linkinfo := range symlinks {
			node, ok := data[linkinfo.name]
			require.True(ok)
			assert.Equal(uint16(syscall.S_IFLNK),
				node.Stat().Mode&syscall.S_IFMT)
		}
	})
}
