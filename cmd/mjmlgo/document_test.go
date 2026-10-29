package main

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentFSServesDocumentAndDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "partial.mjml"), "from disk")

	fsys := documentFS{name: stdinName, src: []byte("from memory"), dir: dir}

	content, err := fs.ReadFile(fsys, stdinName)
	require.NoError(t, err)
	require.Equal(t, "from memory", string(content))

	content, err = fs.ReadFile(fsys, "partial.mjml")
	require.NoError(t, err)
	require.Equal(t, "from disk", string(content))

	_, err = fs.ReadFile(fsys, "missing.mjml")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestDocumentFSReportsDocumentMetadata(t *testing.T) {
	t.Parallel()

	fsys := documentFS{name: stdinName, src: []byte("abc"), dir: t.TempDir()}

	file, err := fsys.Open(stdinName)
	require.NoError(t, err)
	defer file.Close()

	info, err := file.Stat()
	require.NoError(t, err)
	require.Equal(t, stdinName, info.Name())
	require.EqualValues(t, 3, info.Size())
	require.False(t, info.IsDir())
	require.Equal(t, fs.FileMode(0o444), info.Mode())

	// The file reads from where it stands and can be closed more than once
	// without a complaint: the renderer reads it once.
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, "abc", string(content))
	require.NoError(t, file.Close())
}

func TestDocumentFSDelegatesMissingDirectory(t *testing.T) {
	t.Parallel()

	fsys := documentFS{name: stdinName, src: []byte("x"), dir: os.TempDir()}

	_, err := fsys.Open("no-such-file.mjml")
	require.Error(t, err)
}
