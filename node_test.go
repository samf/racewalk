package racewalk

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileNode(t *testing.T) {
	dirpath := setup()
	defer cleanup(dirpath)

	t.Run("happy path", func(t *testing.T) {
		assert := assert.New(t)

		fnode, err := MakeFileNode(dirpath)
		assert.NoError(err)
		assert.Equal(dirpath, fnode.Name())
		stat := fnode.GetStat()
		assert.NotNil(stat)
		assert.Equal(uint16(syscall.S_IFDIR), stat.Mode&syscall.S_IFMT)
	})
	t.Run("MakeFileNode", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		dirchild := filepath.Join(dirpath, "little-dir")
		err := os.Mkdir(dirchild, 0777)
		require.NoError(err)

		fileNode, err := MakeFileNode(dirchild)
		assert.NoError(err)
		assert.True(fileNode.IsDir())
	})
}
