package gitreader

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommitObject(t *testing.T) {
	plain := []byte("commit 165\x00parent abcd\ntree b28f66668670da36a8618360d1f16f3415dfaa3f\nauthor Evan Phoenix <evan@phx.io> 1418539320 -0800\ncommitter Evan Phoenix <evan@phx.io> 1418539320 -0800\n\nadd Procfile\n")

	var compress bytes.Buffer

	zw := zlib.NewWriter(&compress)
	zw.Write(plain)
	zw.Close()

	obj, err := ParseObject(&compress)
	require.NoError(t, err)

	assert.Equal(t, "commit", obj.Type)
	assert.Equal(t, 165, obj.Size)

	commit, err := obj.Commit()
	require.NoError(t, err)

	assert.Equal(t, "abcd", commit.Parent)
	assert.Equal(t, "b28f66668670da36a8618360d1f16f3415dfaa3f", commit.Tree)
	assert.Equal(t, "Evan Phoenix <evan@phx.io> 1418539320 -0800", commit.Author)
	assert.Equal(t, "Evan Phoenix <evan@phx.io> 1418539320 -0800", commit.Committer)
	assert.Equal(t, "add Procfile\n", commit.Message)
}

func TestParseTreeObject(t *testing.T) {
	plain := []byte("tree 36\x00100644 Procfile\x00^\x7FE{\xB1s/C\x15\xF3\xB6\x19>\xE8^\xFD\xF7s]P")

	var compress bytes.Buffer

	zw := zlib.NewWriter(&compress)
	zw.Write(plain)
	zw.Close()

	obj, err := ParseObject(&compress)
	require.NoError(t, err)

	assert.Equal(t, "tree", obj.Type)
	assert.Equal(t, 36, obj.Size)

	tree, err := obj.Tree()
	require.NoError(t, err)

	entry, ok := tree.Entries["Procfile"]
	require.True(t, ok)

	assert.Equal(t, "100644", entry.Permissions)
	assert.Equal(t, "Procfile", entry.Name)
	assert.Equal(t, "5e7f457bb1732f4315f3b6193ee85efdf7735d50", entry.Id)
}

func TestParseBlobObject(t *testing.T) {
	plain := []byte("blob 10\x00web: puma\n")

	var compress bytes.Buffer

	zw := zlib.NewWriter(&compress)
	zw.Write(plain)
	zw.Close()

	obj, err := ParseObject(&compress)
	require.NoError(t, err)

	assert.Equal(t, "blob", obj.Type)
	assert.Equal(t, 10, obj.Size)

	rdr, err := obj.Blob()
	require.NoError(t, err)

	blob, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)

	assert.Equal(t, []byte("web: puma\n"), blob)
}
