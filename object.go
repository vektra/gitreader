package gitreader

import (
	"bufio"
	"compress/zlib"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

var ErrNotExist = errors.New("object does not exist")

type Object struct {
	Type string
	Size uint64

	input io.ReadCloser
	body  *bufio.Reader
}

// Cleanup the object's resources
func (o *Object) Close() error {
	return o.input.Close()
}

// Read the data and construct a new Object
func ParseObject(input io.Reader) (*Object, error) {
	plain, err := zlib.NewReader(input)
	if err != nil {
		return nil, err
	}

	buf := bufio.NewReader(plain)

	typ, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	szstr, err := buf.ReadString(0)
	if err != nil {
		return nil, err
	}

	sz, err := strconv.ParseUint(szstr[:len(szstr)-1], 10, 0)
	if err != nil {
		return nil, err
	}

	obj := &Object{
		Type:  typ[:len(typ)-1],
		Size:  sz,
		input: plain,
		body:  buf,
	}

	return obj, nil
}

func (o *Object) readValue() (string, string, error) {
	line, err := o.body.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	line = strings.TrimSpace(line)

	if line == "" {
		return "", "", nil
	}

	parts := strings.SplitN(line, " ", 2)

	return parts[0], parts[1], nil
}

type Commit struct {
	Parent, Tree, Author, Committer, Message string
}

// Return the Object as a Commit
func (o *Object) Commit() (*Commit, error) {
	com := &Commit{}

	for {
		kind, data, err := o.readValue()
		if err != nil {
			return nil, err
		}

		if kind == "" {
			break
		}

		switch kind {
		case "parent":
			com.Parent = data
		case "tree":
			com.Tree = data
		case "author":
			com.Author = data
		case "committer":
			com.Committer = data
		default:
			return nil, fmt.Errorf("Unknown value: %s", kind)
		}
	}

	data, err := ioutil.ReadAll(o.body)
	if err != nil {
		return nil, err
	}

	com.Message = string(data)

	return com, nil
}

type Tree struct {
	Entries map[string]*Entry
}

type Entry struct {
	Permissions, Name, Id string
}

// Return the Object as a Tree
func (o *Object) Tree() (*Tree, error) {
	tree := &Tree{
		Entries: make(map[string]*Entry),
	}

	idbytes := make([]byte, 20)

	for {
		name, err := o.body.ReadString(0)
		if err != nil {
			if err == io.EOF {
				return tree, nil
			}

			return nil, err
		}

		parts := strings.SplitN(name[:len(name)-1], " ", 2)

		_, err = o.body.Read(idbytes)
		if err != nil {
			return nil, err
		}

		entry := &Entry{
			Permissions: parts[0],
			Name:        parts[1],
			Id:          hex.EncodeToString(idbytes),
		}

		tree.Entries[entry.Name] = entry
	}

	return nil, nil
}

type Blob struct {
	io.Reader
	all []byte
}

// Read all the data in the blob and return it.
// WARNING: use Blob as an io.Reader instead if you can!
func (b *Blob) Bytes() ([]byte, error) {
	if b.all != nil {
		return b.all, nil
	}

	all, err := ioutil.ReadAll(b)
	if err != nil {
		return nil, err
	}

	b.all = all

	return all, nil
}

// Return the Object as a Blob
func (o *Object) Blob() (*Blob, error) {
	return &Blob{o.body, nil}, nil
}
