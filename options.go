package racewalk

import (
	"fmt"
	"os"
	"runtime"
)

var (
	maxWorkers        = runtime.NumCPU() * 2
	maxTaskBufferSize = 1024
)

// Options is a structure that can be used to set options on the
// Walk() function.
// ErrHandler is an optional hook that is called when we are unable to traverse
// NumWorkers sets the size of the worker pool. If it's zero, then the number
// of workers gets set automatically.
// Debug enables some debugging features.
type Options struct {
	ErrHandler     ErrHandler
	NumWorkers     int
	TaskBufferSize int
	Debug          bool

	pending *int32
}

// ErrHandler is a handler for errors encountered when Walk() is traversing
// path is the path that we tried to traverse, and err is the error encountered
//
// If the hook returns an error, processing will stop and Walk() will return
// the given error.
type ErrHandler func(path string, err error) error

// DefaultErrHandler is used in Walk() when no ErrHandler is set in Options
// it simply prints the error to stderr and doesn't return the error
func DefaultErrHandler(path string, err error) error {
	fmt.Fprintf(os.Stderr, "%v: %v\n", path, err)
	return nil
}

func (opt *Options) valid() error {
	if opt.ErrHandler == nil {
		opt.ErrHandler = DefaultErrHandler
	}

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

func (opt Options) printf(format string, rest ...interface{}) {
	if !opt.Debug {
		return
	}

	fmt.Fprintf(os.Stderr, "DEBUG: "+format, rest...)
}

func (opt Options) println(rest ...interface{}) {
	if !opt.Debug {
		return
	}

	rest = append([]interface{}{"DEBUG:"}, rest...)

	fmt.Fprintln(os.Stderr, rest...)
}
