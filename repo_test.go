package gitreader

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoResolveRef(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj")
	require.NoError(t, err)

	id, err := repo.ResolveRef("master")
	require.NoError(t, err)

	assert.Equal(t, "bdae0e92f4a7ca0ec05b6c2decab9dc18361750b", id)
}

func TestRepoOpenRepoBare(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj/.git")
	require.NoError(t, err)

	id, err := repo.ResolveRef("master")
	require.NoError(t, err)

	assert.Equal(t, "bdae0e92f4a7ca0ec05b6c2decab9dc18361750b", id)
}

func TestRepoResolveRefReadsTags(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj")
	require.NoError(t, err)

	id, err := repo.ResolveRef("before")
	require.NoError(t, err)

	assert.Equal(t, "6fe9de222caf76a787e0df553264d0d9f3bc4ead", id)
}

func TestRepoResolveRefHEAD(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj")
	require.NoError(t, err)

	id, err := repo.ResolveRef("HEAD")
	require.NoError(t, err)

	assert.Equal(t, "bdae0e92f4a7ca0ec05b6c2decab9dc18361750b", id)
}

func TestRepoLoadObject(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj")
	require.NoError(t, err)

	obj, err := repo.LoadObject("467c21715563cbf5bf52ae79616e02914b89e9f1")
	require.NoError(t, err)

	assert.Equal(t, "blob", obj.Type)

	blob, err := obj.Blob()
	require.NoError(t, err)

	all, err := ioutil.ReadAll(blob)
	require.NoError(t, err)

	assert.Equal(t, []byte("web: puma\nworker: sidekiq\n"), all)
}

func TestRepoResolve(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj")
	require.NoError(t, err)

	id, err := repo.Resolve("HEAD", "Procfile")
	require.NoError(t, err)

	assert.Equal(t, "467c21715563cbf5bf52ae79616e02914b89e9f1", id)
}

func TestRepoResolveInSubtree(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj")
	require.NoError(t, err)

	id, err := repo.Resolve("HEAD", "app/config.rb")
	require.NoError(t, err)

	assert.Equal(t, "ce013625030ba8dba906f756967f9e9ca394464a", id)
}

func TestRepoCatFile(t *testing.T) {
	repo, err := OpenRepo("fixtures/proj")
	require.NoError(t, err)

	blob, err := repo.CatFile("HEAD", "Procfile")
	require.NoError(t, err)

	all, err := blob.Bytes()
	require.NoError(t, err)

	assert.Equal(t, "web: puma\nworker: sidekiq\n", string(all))
}
