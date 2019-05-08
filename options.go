package racewalk

import (
	"fmt"
	"runtime"
)

var (
	maxWorkers = runtime.NumCPU() * 2
)

// Options is a structure that can be used to set options on the
// Walk() function.
// NumWorkers sets the size of the worker pool. If it's zero, then the number
// of workers gets set automatically.
// Debug enables some debugging features.
type Options struct {
	NumWorkers int
	Debug      bool

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

	opt.pending = new(int32)

	return nil
}
