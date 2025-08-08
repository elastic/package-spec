// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fspath

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FS implements the fs interface and can also show a path where the fs is located.
// This is useful to report error messages relative to the location of the file system.
type FS interface {
	fs.FS

	Path(name ...string) string
}

type fsDir struct {
	fs.FS

	path string
}

// Path returns a path for the given names, based on the location of the file system.
func (fs *fsDir) Path(names ...string) string {
	return filepath.Join(append([]string{fs.path}, namesFromSlash(names)...)...)
}

func namesFromSlash(names []string) []string {
	result := make([]string, len(names))
	for i, name := range names {
		result[i] = filepath.FromSlash(name)
	}
	return result
}

// DirFS returns a file system for a directory, it keeps the path to implement the FS interface.
func DirFS(path string) FS {
	return &fsDir{
		FS:   os.DirFS(path),
		path: path,
	}
}
