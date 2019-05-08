package racewalk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		assert := assert.New(t)

		o := new(Options)
		err := o.valid()
		assert.NoError(err)
		assert.True(o.NumWorkers > 0)
	})
	t.Run("negative workers", func(t *testing.T) {
		assert := assert.New(t)

		o := &Options{
			NumWorkers: -6,
		}
		err := o.valid()
		assert.Error(err)
	})
	t.Run("too many workers", func(t *testing.T) {
		assert := assert.New(t)

		o := &Options{
			NumWorkers: maxWorkers + 7,
		}
		err := o.valid()
		assert.Error(err)
	})
}
