// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package linkedfiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const linkExtension = ".link"

var (
	_ fs.FS = (*FS)(nil)
	_ fs.FS = (*BlockFS)(nil)
	// ErrUnsupportedLinkFile is returned when a linked file is not supported.
	ErrUnsupportedLinkFile = errors.New("linked files are not supported in this filesystem")
)

// FS is a filesystem that handles linked files.
// It wraps another filesystem and checks for linked files with the ".link" extension.
// If a linked file is found, it reads the link file to determine the target file
// and its checksum. If the target file is up to date, it returns the target file.
// Otherwise, it returns an error.
type FS struct {
	workDir string
	inner   fs.FS
}

// NewFS creates a new FS.
func NewFS(workDir string, inner fs.FS) *FS {
	return &FS{workDir: workDir, inner: inner}
}

// Open opens a file in the filesystem.
func (lfs *FS) Open(name string) (fs.File, error) {
	if filepath.Ext(name) != linkExtension {
		return lfs.inner.Open(name)
	}
	pathName := filepath.Join(lfs.workDir, name)
	l, err := NewLinkedFile(pathName)
	if err != nil {
		return nil, err
	}
	if !l.UpToDate {
		return nil, fmt.Errorf("linked file %s is not up to date", name)
	}
	includedPath := filepath.Join(lfs.workDir, filepath.Dir(name), l.IncludedFilePath)
	return os.Open(includedPath)
}

// BlockFS is a filesystem that blocks use of linked files.
type BlockFS struct {
	inner fs.FS
}

// NewBlockFS creates a new BlockFS.
func NewBlockFS(inner fs.FS) *BlockFS {
	return &BlockFS{inner: inner}
}

// Open opens a file in the filesystem.
func (bfs *BlockFS) Open(name string) (fs.File, error) {
	if filepath.Ext(name) == linkExtension {
		return nil, ErrUnsupportedLinkFile
	}
	return bfs.inner.Open(name)
}
