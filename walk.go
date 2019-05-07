package racewalk

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
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

// NumWorkers is the number of goroutines walking your tree
var NumWorkers = 7

// Walk calls the 'handle' function for every directory under top. The handle
// function may be called from a go routine.
func Walk(top string, handler WalkHandler) error {
	var pending int32

	// read the top directory
	first, err := dirToWorkItem(top)
	if err != nil {
		return err
	}

	// we now have one workItem
	pending = 1
	work := make(chan *workItem, 1)
	defer close(work)
	work <- first

	errs := make(chan error)
	defer close(errs)
	done := make(chan struct{})

	for i := 0; i < NumWorkers; i++ {
		go walker(work, errs, done, handler, &pending)
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
	handler WalkHandler, pending *int32) {
	for {
		select {
		case wi := <-work:
			if wi == nil {
				continue
			}

			dirs, err := handler(wi.top, wi.dirs, wi.others)
			if err != nil {
				errs <- err
				return
			}

			for _, dir := range dirs {
				dirpath := filepath.Join(wi.top, dir.Name())
				wi, err := dirToWorkItem(dirpath)
				if err != nil {
					errs <- err
					return
				}
				work <- wi
				atomic.AddInt32(pending, 1)
			}

			if atomic.AddInt32(pending, -1) == 0 {
				close(done)
				return
			}
		case <-done:
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
