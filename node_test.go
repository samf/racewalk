package racewalk

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileNode(t *testing.T) {
	dirpath := setup()
	defer cleanup(dirpath)

	t.Run("happy path", func(t *testing.T) {
		assert := assert.New(t)

		finfo, err := os.Lstat(filepath.Join(dirpath, "top"))
		assert.NoError(err)

		fnode := complete(dirpath, finfo)
		assert.Equal("top", fnode.Name())
		stat := fnode.GetStat()
		assert.NotNil(stat)
		assert.Equal(uint16(syscall.S_IFDIR), stat.Mode&syscall.S_IFMT)
	})
}
