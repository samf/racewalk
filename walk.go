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
//
// It receives a slice of subdirs and entries, and returns a slice of dirs. The
// returned slice of subdirs may be the same as was passed in, or it may remove
// elements from the original, signifying that we will skip that directory.
//
// The 'entries' argument will contain the directory corresponding to 'root'
// first, followed by other non-directory entries.
type WalkHandler func(root string, subdirs []FileNode,
	entries []FileNode) ([]FileNode, error)

type workItem struct {
	top     string
	subdirs []FileNode
	entries []FileNode
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
	work := make(chan *workItem, opt.TaskBufferSize)
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
				opt.printf("Signal: %v\n", s)
				opt.printf("Pending: %v\n", atomic.LoadInt32(opt.pending))
				opt.printf("Work channel length: %v\n", len(work))
				opt.printf("Goroutines remaining: %v\n", runtime.NumGoroutine())
			case <-done:
			}
			signal.Stop(sigs)
		}()
	}

	select {
	case err := <-errs:
		close(done)
		return err
	case <-done:
	}

	if len(work) > 0 {
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
				opt.println(os.Stderr, "skipped a nil workItem")
				continue
			}

			dirs, err := handler(wi.top, wi.subdirs, wi.entries)
			if err != nil {
				opt.printf("sending error from handler: %v", err)
				errs <- err
				return
			}

			leftovers := []*workItem{}
			for _, dir := range dirs {
				dirpath := filepath.Join(wi.top, dir.Name())
				wi, err := dirToWorkItem(dirpath)
				if err != nil {
					opt.printf("sending error from dwi: %v", err)
					errs <- err
					return
				}
				select {
				case work <- wi:
				case <-done:
					return
				default:
					leftovers = append(leftovers, wi)
				}
				atomic.AddInt32(opt.pending, 1)
			}

			if len(leftovers) > 0 {
				go func() {
					for _, wi := range leftovers {
						select {
						case work <- wi:
						case <-done:
							break
						}
					}
				}()
			}

			if atomic.AddInt32(opt.pending, -1) == 0 {
				opt.println("goal reached")
				close(done)
				return
			}
		case <-done:
			opt.println("got a 'done' notice")
			return
		}
	}
}

func dirToWorkItem(dirpath string) (*workItem, error) {
	dirnode, err := MakeFileNode(dirpath)
	if err != nil {
		return nil, err
	}

	workItem := &workItem{
		top:     dirpath,
		entries: []FileNode{*dirnode},
	}

	finfos, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	for _, finfo := range finfos {
		fnode := complete(dirpath, finfo)

		if fnode.IsDir() {
			workItem.subdirs = append(workItem.subdirs, *fnode)
		} else {
			workItem.entries = append(workItem.entries, *fnode)
		}
	}

	return workItem, nil
}
