package gitreader

// This is adapted from https://github.com/edsrzf/go-git/blob/master/pack.go

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/edsrzf/mmap-go"
)

// Load the pack data from the given path
func LoadPack(path string) (*Pack, error) {
	pack := &Pack{
		idxPath:  path + ".idx",
		dataPath: path + ".pack",
	}

	err := pack.loadIndex()
	if err != nil {
		return nil, err
	}

	err = pack.loadData()
	if err != nil {
		return nil, err
	}

	return pack, nil
}

var order = binary.BigEndian

// Implements LoadObject for a pack file
type Pack struct {
	idxPath   string
	indexFile *os.File
	index     mmap.MMap

	dataPath string
	dataFile *os.File
	data     mmap.MMap
}

func (p *Pack) Close() error {
	p.index.Unmap()
	p.indexFile.Close()
	p.data.Unmap()
	return p.dataFile.Close()
}

var ErrBadIndex = errors.New("bad index format")
var ErrBadPack = errors.New("bad pack format")

const indexHeader = "\xFF\x74\x4F\x63\x00\x00\x00\x02"

func (p *Pack) loadIndex() error {
	var err error
	p.indexFile, err = os.Open(p.idxPath)
	if err != nil {
		return err
	}

	p.index, err = mmap.Map(p.indexFile, mmap.RDONLY, 0)
	if err != nil {
		return err
	}

	if string([]byte(p.index[:8])) != indexHeader {
		return ErrBadIndex
	}

	return nil
}

const packHeader = "PACK\x00\x00\x00\x02"

func (p *Pack) loadData() error {
	var err error
	p.dataFile, err = os.Open(p.dataPath)
	if err != nil {
		return err
	}

	p.data, err = mmap.Map(p.dataFile, mmap.RDONLY, 0)
	if err != nil {
		return err
	}

	if string([]byte(p.data[:8])) != packHeader {
		return ErrBadPack
	}

	return nil
}

const (
	_OBJ_COMMIT = iota + 1
	_OBJ_TREE
	_OBJ_BLOB
	_OBJ_TAG
	_
	_OBJ_OFS_DELTA
	_OBJ_REF_DELTA
)

var ErrNotFound = errors.New("object not found")

func (p *Pack) FindOffset(id string) (uint32, error) {
	idBytes, err := hex.DecodeString(id)
	if err != nil {
		return 0, err
	}

	// 255 uint32s
	fan := p.index[8:1032]
	size := order.Uint32(fan[1020:])

	// TODO: Make sure fan[id[0]] > fan[id[0] - 1]
	//       Otherwise our object's definitely not here
	cnt := order.Uint32(fan[4*id[0]:])

	loc := 8 + 1024 + cnt*20
	suspect := p.index[loc : loc+20]
	cmp := bytes.Compare(idBytes, suspect)

	for lo, hi := uint32(0), size; cmp != 0; cmp = bytes.Compare(idBytes, suspect) {
		if cmp < 0 {
			hi = cnt
		} else {
			lo = cnt + 1
		}
		if lo >= hi {
			return 0, ErrNotExist
		}
		cnt = (lo + hi) / 2
		loc = 8 + 1024 + cnt*20
		suspect = p.index[loc : loc+20]
	}

	// TODO: check for 64-bit offset
	// calculate which sha1 we looked at
	n := (loc - 1032) / 20
	offsetBase := 1032 + 20*size + 4*size
	offset := order.Uint32(p.index[offsetBase+4*n:])
	return offset, nil
}

func (p *Pack) LoadObject(id string) (*Object, error) {
	offset, err := p.FindOffset(id)
	if err != nil {
		return nil, err
	}

	return p.readObject(offset)
}

var ErrUnknownType = errors.New("unknown type")

func (p *Pack) readObject(offset uint32) (*Object, error) {
	objType, objSize, rdr, err := p.readRaw(offset)
	if err != nil {
		return nil, err
	}

	obj := &Object{
		Size:  objSize,
		input: rdr,
		body:  bufio.NewReader(rdr),
	}

	switch objType {
	case _OBJ_COMMIT:
		obj.Type = "commit"
	case _OBJ_TREE:
		obj.Type = "tree"
	case _OBJ_BLOB:
		obj.Type = "blob"
	default:
		return nil, ErrUnknownType
	}

	return obj, nil
}

var ErrBadDelta = errors.New("bad delta")

func (p *Pack) readRaw(offset uint32) (int, uint64, io.ReadCloser, error) {
	objHeader := p.data[offset]
	objType := int(objHeader & 0x71 >> 4)

	// size when uncompressed
	objSize := uint64(objHeader & 0x0F)
	i := uint32(0)
	shift := uint32(4)
	for objHeader&0x80 != 0 {
		i++
		objHeader = p.data[offset+i]
		objSize |= uint64(objHeader&0x7F) << shift
		shift += 7
	}

	var err error
	var rawBase io.ReadCloser

	if objType == _OBJ_OFS_DELTA {
		i++
		b := p.data[offset+i]
		baseOffset := uint32(b & 0x7F)
		for b&0x80 != 0 {
			i++
			b = p.data[offset+i]
			baseOffset = ((baseOffset + 1) << 7) | uint32(b&0x7F)
		}

		if baseOffset > uint32(len(p.data)) || baseOffset > offset {
			return 0, 0, nil, ErrBadDelta
		}

		objType, objSize, rawBase, err = p.readRaw(offset - baseOffset)
		if err != nil {
			return 0, 0, nil, err
		}

	} else if objType == _OBJ_REF_DELTA {
		baseId := hex.EncodeToString(p.data[offset+i+1 : offset+i+21])
		i += 20
		baseOffset, err := p.FindOffset(baseId)
		if err != nil {
			return 0, 0, nil, err
		}

		objType, objSize, rawBase, err = p.readRaw(baseOffset)
		if err != nil {
			return 0, 0, nil, err
		}
	}

	buf := bytes.NewReader(p.data[offset+i+1:])
	r, err := zlib.NewReader(buf)
	if err != nil {
		return 0, 0, nil, err
	}

	if rawBase != nil {
		// apply delta to base
		r, objSize, err = applyDelta(rawBase, r)
		if err != nil {
			return 0, 0, nil, err
		}
	}

	return objType, objSize, r, nil
}

type closableReader struct {
	*bytes.Reader
}

func (c closableReader) Close() error {
	return nil
}

func applyDelta(base_r, patch_r io.ReadCloser) (io.ReadCloser, uint64, error) {
	patch, err := ioutil.ReadAll(patch_r)
	if err != nil {
		return nil, 0, err
	}

	base, err := ioutil.ReadAll(base_r)
	if err != nil {
		return nil, 0, err
	}

	// base length; TODO: use for bounds checking
	baseLength, n := decodeVarint(patch)
	if baseLength != uint64(len(base)) {
		return nil, 0, ErrBadDelta
	}

	patch = patch[n:]
	resultLength, n := decodeVarint(patch)
	patch = patch[n:]

	result := make([]byte, resultLength)
	loc := uint(0)
	for len(patch) > 0 {
		i := uint(1)

		op := patch[0]
		if op == 0 {
			return nil, 0, ErrBadDelta
		} else if op&0x80 == 0 {
			// insert
			n := uint(op)
			copy(result[loc:], patch[i:i+n])
			loc += n
			patch = patch[i+n:]
			continue
		}

		copyOffset := uint(0)
		for j := uint(0); j < 4; j++ {
			if op&(1<<j) != 0 {
				x := patch[i]
				i++
				copyOffset |= uint(x) << (j * 8)
			}
		}

		copyLength := uint(0)
		for j := uint(0); j < 3; j++ {
			if op&(1<<(4+j)) != 0 {
				x := patch[i]
				i++
				copyLength |= uint(x) << (j * 8)
			}
		}

		if copyLength == 0 {
			copyLength = 1 << 16
		}

		if copyOffset+copyLength > uint(len(base)) || copyLength > uint(len(result[loc:])) {
			return nil, 0, ErrBadDelta
		}

		copy(result[loc:], base[copyOffset:copyOffset+copyLength])
		loc += copyLength
		patch = patch[i:]
	}

	return closableReader{bytes.NewReader(result)}, resultLength, nil
}

func decodeVarint(buf []byte) (x uint64, n int) {
	shift := uint64(0)
	for {
		b := buf[n]
		n++
		x |= uint64(b&0x7F) << shift
		shift += 7
		if b&0x80 == 0 {
			return
		}
	}
	return
}
