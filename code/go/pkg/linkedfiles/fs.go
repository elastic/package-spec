// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package linkedfiles

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

var _ fs.FS = (*LinksFS)(nil)

// LinksFS is a filesystem that handles linked files.
// It wraps another filesystem and checks for linked files with the ".link" extension.
// If a linked file is found, it reads the link file to determine the target file
// and its checksum. If the target file is up to date, it returns the target file.
// Otherwise, it returns an error.
type LinksFS struct {
	workDir string
	inner   fs.FS
}

// NewLinksFS creates a new LinksFS.
func NewLinksFS(workDir string, inner fs.FS) *LinksFS {
	return &LinksFS{workDir: workDir, inner: inner}
}

// Open opens a file in the filesystem.
func (lfs *LinksFS) Open(name string) (fs.File, error) {
	if filepath.Ext(name) != LinkExtension {
		return lfs.inner.Open(name)
	}
	pathName := filepath.Join(lfs.workDir, name)
	l, err := newLinkedFile(pathName)
	if err != nil {
		return nil, err
	}
	if !l.UpToDate {
		return nil, fmt.Errorf("linked file %s is not up to date", name)
	}
	return lfs.inner.Open(filepath.Join(filepath.Dir(name), l.IncludedFilePath))
}
