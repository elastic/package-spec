// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/package-spec/code/go/internal/spectypes"
)

//go:embed testdata/limits/manifest.yml
var manifestYml string

//go:embed testdata/limits/changelog.yml
var changelogYml string

func TestLimitsValidation(t *testing.T) {
	cases := []struct {
		title string
		fsys  fs.FS
		valid bool
	}{
		{
			title: "all good",
			fsys:  newMockFS().Good(),
			valid: true,
		},
		{
			title: "configurationSizeLimit exceeded",
			fsys: newMockFS().Good().
				Override("manifest.yml", OverrideSize(10*spectypes.MegaByte)),
			valid: false,
		},
		{
			title: "sizeLimit exceeded",
			fsys: newMockFS().Good().
				Override("docs/README.md", OverrideSize(200*spectypes.MegaByte)),
			valid: false,
		},
		{
			title: "totalContentsLimit exceeded",
			fsys: newMockFS().Good().
				Override("docs", OverrideGenerateFiles(70000, ".md", 512*spectypes.Byte)),
			valid: false,
		},
		{
			title: "relativePathSizeLimit exceeded",
			fsys: newMockFS().Good().
				Override("img/kibana-system.png", OverrideSize(10*spectypes.MegaByte)),
			valid: false,
		},
		{
			title: "ignore developer files",
			fsys: newMockFS().Good().
				Override("_dev/deploy/docker", OverrideAddFiles(
					newMockFile("entrypoint.sh").WithSize(2048*spectypes.MegaByte))),
			valid: true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			err := ValidateFromFS("test-package", c.fsys)
			if c.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

var _ fs.FS = &mockFS{}

type mockFS struct {
	root *mockFile
}

func newMockFS() *mockFS {
	return &mockFS{root: newMockDir(".")}
}

func (fs *mockFS) WithFiles(files ...*mockFile) *mockFS {
	fs.root.WithFiles(files...)
	return fs
}

func (fs *mockFS) Good() *mockFS {
	return fs.WithFiles(
		newMockFile("manifest.yml").WithContent(manifestYml),
		newMockFile("changelog.yml").WithContent(changelogYml),
		newMockDir("docs").WithFiles(
			newMockFile("README.md").WithContent("## README"),
		),
		newMockDir("img").WithFiles(
			newMockFile("kibana-system.png"),
			newMockFile("system.svg"),
		),
		newMockDir("_dev").WithFiles(
			newMockDir("deploy").WithFiles(
				newMockDir("docker").WithFiles(
					newMockFile("docker-compose.yml").WithContent("version: 2.3"),
				),
			),
		),
	)
}

func (fs *mockFS) Override(name string, override func(*mockFile)) *mockFS {
	f, err := fs.root.findFile(name)
	if err != nil {
		panic("overriding " + name + ": " + err.Error())
	}
	override(f)
	return fs
}

func OverrideSize(size spectypes.FileSize) func(*mockFile) {
	return func(f *mockFile) {
		f.WithSize(size)
	}
}

func OverrideAddFiles(files ...*mockFile) func(*mockFile) {
	return func(f *mockFile) {
		f.WithFiles(files...)
	}
}

func OverrideGenerateFiles(n int, suffix string, size spectypes.FileSize) func(*mockFile) {
	return func(f *mockFile) {
		var files []*mockFile
		for i := 0; i < n; i++ {
			files = append(files,
				newMockFile(fmt.Sprintf("tmp%d%s", i, suffix)).WithSize(size))
		}
		f.WithFiles(files...)
	}
}

func (fs *mockFS) Open(name string) (fs.File, error) {
	f, err := fs.root.findFile(name)
	if err != nil {
		return nil, err
	}
	return f.open(), nil
}

var _ fs.File = &mockFile{}
var _ fs.ReadDirFile = &mockFile{}

type mockFile struct {
	stat    mockFileInfo
	content string
	reader  io.Reader
	files   []*mockFile
}

func newMockFile(name string) *mockFile {
	if strings.Contains(name, string(os.PathSeparator)) {
		panic(name + " contains a path separator " + string(os.PathSeparator))
	}
	return &mockFile{
		stat: mockFileInfo{
			name:  name,
			mode:  0644,
			isDir: false,
		},
	}
}

func newMockDir(name string) *mockFile {
	f := newMockFile(name)
	f.stat.mode = 0755
	f.stat.isDir = true
	return f
}

func (f *mockFile) WithContent(content string) *mockFile {
	if f.stat.isDir {
		panic("directory cannot have content")
	}
	f.content = content
	f.stat.size = int64(len(content))
	return f
}

func (f *mockFile) WithSize(size spectypes.FileSize) *mockFile {
	f.stat.size = int64(size)
	return f
}

func (f *mockFile) WithFiles(files ...*mockFile) *mockFile {
	if !f.stat.isDir {
		panic("regular file cannot contain files")
	}
	f.files = append(f.files, files...)
	return f
}

func (f *mockFile) findFile(name string) (*mockFile, error) {
	if name == "." {
		return f, nil
	}
	name = filepath.Clean(name)
	parts := strings.SplitN(name, string(os.PathSeparator), 2)

	if len(parts) == 0 {
		panic("path should not be empty here")
	}

	var file *mockFile
	for _, candidate := range f.files {
		if candidate.stat.name == parts[0] {
			file = candidate
			break
		}
	}

	if file == nil {
		return nil, os.ErrNotExist
	}

	if len(parts) == 2 {
		return file.findFile(parts[1])
	}
	return file, nil
}

func (f *mockFile) open() *mockFile {
	var descriptor mockFile
	descriptor = *f
	if f.content != "" {
		descriptor.reader = strings.NewReader(f.content)
	}
	return &descriptor
}

func (f *mockFile) Stat() (fs.FileInfo, error) { return &f.stat, nil }
func (f *mockFile) Read(d []byte) (int, error) {
	if f.reader == nil {
		return 0, io.EOF
	}
	return f.reader.Read(d)
}
func (f *mockFile) Close() error { return nil }

func (f *mockFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.stat.isDir {
		return nil, os.ErrInvalid
	}
	var result []fs.DirEntry
	for i, entry := range f.files {
		if n > 0 && i >= n {
			break
		}
		result = append(result, &entry.stat)
	}
	return result, nil
}

var _ fs.FileInfo = &mockFileInfo{}
var _ fs.DirEntry = &mockFileInfo{}

type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *mockFileInfo) Info() (fs.FileInfo, error) { return fi, nil }
func (fi *mockFileInfo) IsDir() bool                { return fi.isDir }
func (fi *mockFileInfo) Mode() fs.FileMode          { return fi.mode }
func (fi *mockFileInfo) ModTime() time.Time         { return fi.modTime }
func (fi *mockFileInfo) Name() string               { return fi.name }
func (fi *mockFileInfo) Size() int64                { return fi.size }
func (fi *mockFileInfo) Sys() interface{}           { return nil }
func (fi *mockFileInfo) Type() fs.FileMode          { return fi.mode.Type() }
