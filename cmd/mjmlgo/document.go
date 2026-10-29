package main

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"time"
)

// documentFS serves one document from memory and resolves every other path
// against a directory. It lets a document that is not in the include root - read
// from standard input, or rendered with -c filePath pointing elsewhere - still
// have its mj-include paths resolved, which is what the reference CLI's filePath
// option is for.
type documentFS struct {
	// name is the path the document answers to inside the filesystem, and dir
	// the directory everything else is resolved against.
	name string
	src  []byte
	dir  string
}

func (f documentFS) Open(name string) (fs.File, error) {
	if name == f.name {
		return &memoryFile{name: name, reader: bytes.NewReader(f.src), size: int64(len(f.src))}, nil
	}

	return os.DirFS(f.dir).Open(name)
}

// memoryFile is the in-memory document as an fs.File.
type memoryFile struct {
	name   string
	reader *bytes.Reader
	size   int64
}

func (f *memoryFile) Stat() (fs.FileInfo, error) { return memoryInfo{name: f.name, size: f.size}, nil }
func (f *memoryFile) Read(p []byte) (int, error) { return f.reader.Read(p) }
func (f *memoryFile) Close() error               { return nil }

// memoryInfo is the metadata of the in-memory document. Nothing in the render
// pipeline looks at modification times, so a zero time is the honest answer.
type memoryInfo struct {
	name string
	size int64
}

func (i memoryInfo) Name() string       { return path.Base(i.name) }
func (i memoryInfo) Size() int64        { return i.size }
func (i memoryInfo) Mode() fs.FileMode  { return 0o444 }
func (i memoryInfo) ModTime() time.Time { return time.Time{} }
func (i memoryInfo) IsDir() bool        { return false }
func (i memoryInfo) Sys() any           { return nil }
