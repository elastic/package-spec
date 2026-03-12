// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkgpath

import (
	"sync"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

type filesEntry struct {
	files []File
	err   error
}

type computedEntry struct {
	value any
	err   error
}

// CachedFS wraps an fspath.FS with cached file access.
// Validators receive this type instead of raw fspath.FS, ensuring all file
// access goes through the Files() method which caches results by glob pattern.
type CachedFS struct {
	fs fspath.FS

	filesMu    sync.Mutex
	filesCache map[string]filesEntry

	computeMu    sync.Mutex
	computeCache map[string]computedEntry
}

// NewCachedFS creates a CachedFS wrapping the given filesystem.
func NewCachedFS(fsys fspath.FS) *CachedFS {
	return &CachedFS{
		fs:           fsys,
		filesCache:   make(map[string]filesEntry),
		computeCache: make(map[string]computedEntry),
	}
}

// Files finds files matching the glob pattern. Results are cached: repeated
// calls with the same pattern return the same File instances, sharing their
// parsed content caches.
func (c *CachedFS) Files(glob string) ([]File, error) {
	c.filesMu.Lock()
	entry, ok := c.filesCache[glob]
	c.filesMu.Unlock()
	if ok {
		return entry.files, entry.err
	}

	files, err := Files(c.fs, glob)

	c.filesMu.Lock()
	c.filesCache[glob] = filesEntry{files, err}
	c.filesMu.Unlock()

	return files, err
}

// Path returns a path for the given names, based on the location of the
// underlying filesystem. Used for error messages and linked file resolution.
func (c *CachedFS) Path(names ...string) string {
	return c.fs.Path(names...)
}

// RawFS returns the underlying filesystem for special cases that need
// direct fs.FS access.
func (c *CachedFS) RawFS() fspath.FS {
	return c.fs
}

// LoadOrStore returns the cached value for key if present. Otherwise it calls
// compute, stores the result, and returns it. This is useful for caching
// derived data (e.g. parsed YAML into custom structs) that cannot use the
// generic File.Values() cache.
func (c *CachedFS) LoadOrStore(key string, compute func() (any, error)) (any, error) {
	c.computeMu.Lock()
	entry, ok := c.computeCache[key]
	c.computeMu.Unlock()
	if ok {
		return entry.value, entry.err
	}

	value, err := compute()

	c.computeMu.Lock()
	c.computeCache[key] = computedEntry{value, err}
	c.computeMu.Unlock()

	return value, err
}
