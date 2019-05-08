package racewalk

import (
	"fmt"
	"runtime"
)

var (
	maxWorkers        = runtime.NumCPU() * 2
	maxTaskBufferSize = 1024
)

// Options is a structure that can be used to set options on the
// Walk() function.
// NumWorkers sets the size of the worker pool. If it's zero, then the number
// of workers gets set automatically.
// Debug enables some debugging features.
type Options struct {
	NumWorkers     int
	TaskBufferSize int
	Debug          bool

	pending *int32
}

func (opt *Options) valid() error {
	switch {
	case opt.NumWorkers < 0:
		return fmt.Errorf("Invalid number of workers: %v", opt.NumWorkers)
	case opt.NumWorkers > maxWorkers:
		return fmt.Errorf("NumWorkers: %v > maximum %v (2 * number of CPUs)",
			opt.NumWorkers, maxWorkers)
	case opt.NumWorkers == 0:
		opt.NumWorkers = maxWorkers / 2
	}

	switch {
	case opt.TaskBufferSize < 0:
		return fmt.Errorf("Invalid task buffer size: %v", opt.TaskBufferSize)
	case opt.TaskBufferSize > maxTaskBufferSize:
		return fmt.Errorf("TaskBufferSize: %v > maximum %v", opt.TaskBufferSize,
			maxTaskBufferSize)
	case opt.TaskBufferSize == 0:
		opt.TaskBufferSize = maxTaskBufferSize / 2
	}

	opt.pending = new(int32)

	return nil
}
