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
}

func (o *Options) valid() error {
	switch {
	case o.NumWorkers < 0:
		return fmt.Errorf("Invalid number of workers: %v", o.NumWorkers)
	case o.NumWorkers > maxWorkers:
		return fmt.Errorf("NumWorkers: %v > maximum %v (2 * number of CPUs)",
			o.NumWorkers, maxWorkers)
	case o.NumWorkers == 0:
		o.NumWorkers = maxWorkers / 2
	}

	return nil
}
