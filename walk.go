package racewalk

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync/atomic"
)

// WalkHandler is a function that is called for every directory visited by Walk.
// It receives a slice of dirs and others, and returns a slice of dirs. The
// returned slice of dirs may be the same as was passed in, or it may remove
// elements from the original, signifying that we will skip that directory.
type WalkHandler func(root string, dirs []FileNode,
	others []FileNode) ([]FileNode, error)

type workItem struct {
	top    string
	dirs   []FileNode
	others []FileNode
}

// Walk calls the 'handle' function for every directory under top. The handle
// function may be called from a go routine.
func Walk(top string, opt *Options, handler WalkHandler) error {
	// check/initialize options
	if opt == nil {
		opt = new(Options)
	}
	err := opt.valid()
	if err != nil {
		return err
	}

	// read the top directory
	first, err := dirToWorkItem(top)
	if err != nil {
		return err
	}

	// we now have one workItem
	atomic.StoreInt32(opt.pending, 1)
	work := make(chan *workItem, 1)
	work <- first

	errs := make(chan error)
	done := make(chan struct{})

	for i := 0; i < opt.NumWorkers; i++ {
		go walker(work, errs, done, handler, opt)
	}

	if opt.Debug {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs)
		go func() {
			select {
			case s := <-sigs:
				fmt.Printf("Signal: %v\n", s)
				fmt.Printf("Pending: %v\n", atomic.LoadInt32(opt.pending))
				fmt.Printf("Work channel length: %v\n", len(work))
				fmt.Printf("Goroutines remaining: %v\n", runtime.NumGoroutine())
			case <-done:
			}
			signal.Stop(sigs)
		}()
	}

	select {
	case err := <-errs:
		return err
	case <-done:
	}

	if len(work) > 1 {
		return fmt.Errorf("unfinished work remaining: %v items", len(work))
	}

	return nil
}

func walker(work chan *workItem, errs chan<- error, done chan struct{},
	handler WalkHandler, opt *Options) {
	for {
		select {
		case wi := <-work:
			if wi == nil {
				if opt.Debug {
					fmt.Println("DEBUG: skipped a nil workItem")
				}
				continue
			}

			dirs, err := handler(wi.top, wi.dirs, wi.others)
			if err != nil {
				if opt.Debug {
					fmt.Printf("DEBUG: sending error from handler: %v", err)
				}
				errs <- err
				return
			}

			for _, dir := range dirs {
				dirpath := filepath.Join(wi.top, dir.Name())
				wi, err := dirToWorkItem(dirpath)
				if err != nil {
					if opt.Debug {
						fmt.Printf("DEBUG: sending error from dwi: %v", err)
					}
					errs <- err
					return
				}
				work <- wi
				atomic.AddInt32(opt.pending, 1)
			}

			if atomic.AddInt32(opt.pending, -1) == 0 {
				if opt.Debug {
					fmt.Println("DEBUG: goal reached")
				}
				close(done)
				return
			}
		case <-done:
			if opt.Debug {
				fmt.Println("DEBUG: got a 'done' notice")
			}
			return
		}
	}
}

func dirToWorkItem(dir string) (*workItem, error) {
	workItem := &workItem{
		top: dir,
	}

	finfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, finfo := range finfos {
		fnode := complete(dir, finfo)

		if fnode.IsDir() {
			workItem.dirs = append(workItem.dirs, *fnode)
		} else {
			workItem.others = append(workItem.others, *fnode)
		}
	}

	return workItem, nil
}
