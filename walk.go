package racewalk

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
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
	var wg sync.WaitGroup

	// read the top directory
	first, err := dirToWorkItem(top)
	if err != nil {
		return err
	}

	// we now have one workItem
	wg.Add(1)
	work := make(chan *workItem, 1)
	work <- first

	errs := make(chan error)
	done := make(chan struct{})

	for i := 0; i < NumWorkers; i++ {
		go walker(work, errs, done, handler, &wg)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errs:
		close(done)
		return err
	case <-done:
	}
	return nil
}

func walker(work chan *workItem, errs chan<- error, done <-chan struct{},
	handler WalkHandler, wg *sync.WaitGroup) {
	for {
		select {
		case wi := <-work:
			dirs, err := handler(wi.top, wi.dirs, wi.others)
			if err != nil {
				errs <- err
				return // return without wg.Done() because we're giving up
			}

			for _, dir := range dirs {
				dirpath := filepath.Join(wi.top, dir.Name())
				wi, err := dirToWorkItem(dirpath)
				if err != nil {
					errs <- err
					return
				}
				work <- wi
				wg.Add(1)
			}

			wg.Done()
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
		fnode, err := complete(dir, finfo)
		if err != nil {
			return nil, err
		}

		if fnode.IsDir() {
			workItem.dirs = append(workItem.dirs, *fnode)
		} else {
			workItem.others = append(workItem.others, *fnode)
		}
	}

	return workItem, nil
}

// complete takes a string and a FileInfo, and returns a FileNode. The string,
// 'top', is a path to the directory containing the FileInfo.
func complete(top string, finfo os.FileInfo) (*FileNode, error) {
	fileNode := FileNode{
		FileInfo: finfo,
	}
	err := unix.Lstat(filepath.Join(top, finfo.Name()), &fileNode.Stat_t)
	if err != nil {
		return nil, err
	}

	return &fileNode, err
}
