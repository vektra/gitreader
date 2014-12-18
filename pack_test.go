package gitreader

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackFindOffset(t *testing.T) {
	pack, err := LoadPack("fixtures/pack-e59dc469beaf63d356b7ca488ca065536cb224f8")
	require.NoError(t, err)

	id := "3e15650095622b50da9e805b2d0550b5961512c9"

	offset, err := pack.FindOffset(id)
	require.NoError(t, err)

	assert.True(t, offset > 0)
}

func TestPackLoadObject(t *testing.T) {
	pack, err := LoadPack("fixtures/pack-e59dc469beaf63d356b7ca488ca065536cb224f8")
	require.NoError(t, err)

	id := "3e15650095622b50da9e805b2d0550b5961512c9"

	object, err := pack.LoadObject(id)
	require.NoError(t, err)

	assert.Equal(t, "commit", object.Type)

	commit, err := object.Commit()
	require.NoError(t, err)

	assert.Equal(t, "b28f66668670da36a8618360d1f16f3415dfaa3f", commit.Tree)
	assert.Equal(t, "Evan Phoenix <evan@phx.io> 1418539320 -0800", commit.Author)
}

func TestPackLoadDeltaObject(t *testing.T) {
	pack, err := LoadPack("fixtures/pack-053ba600409ce6dbe6d211b6d34f9ef86a447ef0")
	require.NoError(t, err)

	id := "4be557ed63be643afaf898197f7dcbabb37630f1"

	object, err := pack.LoadObject(id)
	require.NoError(t, err)

	assert.Equal(t, "blob", object.Type)

	blob, err := object.Blob()
	require.NoError(t, err)

	sha := sha1.New()

	n, err := io.Copy(sha, blob)
	require.NoError(t, err)

	assert.Equal(t, object.Size, n)

	sum := sha.Sum(nil)

	hexSum := hex.EncodeToString(sum)

	assert.Equal(t, "a62edf8685920f7d5a95113020631cdebd18a185", hexSum)
}
