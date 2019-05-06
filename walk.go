package racewalk

// WalkHandler is a function that is called for every directory visited by Walk.
// It receives a slice of dirs and others, and returns a slice of dirs. The
// returned slice of dirs may be the same as was passed in, or it may remove
// elements from the original, signifying that we will skip that directory.
type WalkHandler func(root string, dirs []FileNode,
	others []FileNode) ([]FileNode, error)

// Walk calls the 'handle' function for every directory under top. The handle
// function may be called from a go routine.
func Walk(top string, handle WalkHandler) error {
	return nil
}
