package gitreader

import (
	"os"
	"path/filepath"
)

// Implements reading objects out of the .git/objects directory
type LooseObject struct {
	Base string
}

func (l *LooseObject) LoadObject(id string) (*Object, error) {
	path := filepath.Join(l.Base, "objects", id[:2], id[2:])

	f, err := os.Open(path)
	if err != nil {
		if _, isPe := err.(*os.PathError); isPe {
			return nil, ErrNotExist
		}
		return nil, err
	}

	return ParseObject(f)
}

func (l *LooseObject) Close() error {
	return nil
}
