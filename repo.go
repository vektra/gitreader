package gitreader

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Loader interface {
	LoadObject(id string) (*Object, error)
	Close() error
}

type Repo struct {
	Base    string
	Loaders []Loader
}

var ErrInvalidRepo = errors.New("invalid repo")

// Open up a repository. Can be either normal or bare.
// Be sure to issue Close() on a repo when you're finished
// with it because that makes sure that any pack files
// used by the repo are properly unmapped.
func OpenRepo(path string) (*Repo, error) {
	tries := []string{filepath.Join(path, ".git"), path}

	var repoPath string

	for _, dir := range tries {
		testDir := filepath.Join(dir, "objects")

		if _, err := os.Stat(testDir); err != nil {
			continue
		}

		repoPath = dir
		break
	}

	if repoPath == "" {
		return nil, ErrInvalidRepo
	}

	repo := &Repo{repoPath, nil}

	err := repo.initLoaders()
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Repo) Close() error {
	for _, loader := range r.Loaders {
		loader.Close()
	}

	return nil
}

func (r *Repo) initLoaders() error {
	loaders := []Loader{&LooseObject{r.Base}}

	packs := filepath.Join(r.Base, "objects/pack")

	files, err := ioutil.ReadDir(packs)
	if err == nil {
		for _, file := range files {
			n := file.Name()
			if filepath.Ext(n) == ".idx" {
				pack, err := LoadPack(filepath.Join(packs, n[:len(n)-4]))
				if err != nil {
					return err
				}

				loaders = append(loaders, pack)
			}
		}
	}

	r.Loaders = loaders

	return nil
}

var refDirs = []string{"heads", "tags"}

var ErrUnknownRef = errors.New("unknown ref")

// Given a reference, return the object id for the commit
func (r *Repo) ResolveRef(ref string) (string, error) {
	if ref == "HEAD" {
		return r.resolveIndirect(filepath.Join(r.Base, "HEAD"))
	}

	for _, dir := range refDirs {
		path := filepath.Join(r.Base, "refs", dir, ref)

		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		return strings.TrimSpace(string(data)), nil
	}

	path := filepath.Join(r.Base, ref)
	data, err := ioutil.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// this might be a raw ref. See if there is a commit there and if so
	// accept it as is.

	obj, err := r.LoadObject(ref)
	if err == nil && obj.Type == "commit" {
		return ref, nil
	}

	return "", ErrUnknownRef
}

func (r *Repo) resolveIndirect(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	id := strings.TrimSpace(string(data))

	if id[0:4] == "ref:" {
		return r.ResolveRef(strings.TrimSpace(id[4:]))
	}

	return id, nil
}

// Lookup an object id
func (r *Repo) LoadObject(id string) (*Object, error) {
	for _, loader := range r.Loaders {
		obj, err := loader.LoadObject(id)
		if err != nil {
			if err == ErrNotExist {
				continue
			}

			return nil, err
		}

		return obj, nil
	}

	return nil, ErrNotExist
}

var ErrNotCommit = errors.New("ref is not a commit")
var ErrNotTree = errors.New("object is not a tree")
var ErrNotBlob = errors.New("object is not a blob")

// Given a ref and a path, return an object id
func (r *Repo) Resolve(ref, path string) (string, error) {
	refId, err := r.ResolveRef(ref)
	if err != nil {
		return "", err
	}

	obj, err := r.LoadObject(refId)
	if err != nil {
		return "", err
	}

	if obj.Type != "commit" {
		return "", ErrNotCommit
	}

	commit, err := obj.Commit()
	if err != nil {
		return "", err
	}

	treeObj, err := r.LoadObject(commit.Tree)
	if err != nil {
		return "", err
	}

	if treeObj.Type != "tree" {
		return "", ErrNotTree
	}

	tree, err := treeObj.Tree()
	if err != nil {
		return "", err
	}

	segments := strings.Split(path, "/")

	for _, seg := range segments[:len(segments)-1] {
		nextId, ok := tree.Entries[seg]
		if !ok {
			return "", ErrNotExist
		}

		obj, err := r.LoadObject(nextId.Id)
		if err != nil {
			return "", err
		}

		if obj.Type != "tree" {
			return "", ErrNotTree
		}

		tree, err = obj.Tree()
		if err != nil {
			return "", err
		}
	}

	final := segments[len(segments)-1]

	nextId, ok := tree.Entries[final]
	if !ok {
		return "", ErrNotExist
	}

	return nextId.Id, nil
}

// Given a ref and a path to a blob, return the blob data
func (r *Repo) CatFile(ref, path string) (*Blob, error) {
	id, err := r.Resolve(ref, path)
	if err != nil {
		return nil, err
	}

	obj, err := r.LoadObject(id)
	if err != nil {
		return nil, err
	}

	if obj.Type != "blob" {
		return nil, ErrNotBlob
	}

	return obj.Blob()
}
