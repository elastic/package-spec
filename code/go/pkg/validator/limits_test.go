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
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/package-spec/code/go/internal/spectypes"
)

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
			fsys: newMockFS().Good().Override(func(o *overrideFS) {
				o.File("manifest.yml").WithSize(10 * spectypes.MegaByte)
			}),
			valid: false,
		},
		{
			title: "sizeLimit exceeded",
			fsys: newMockFS().Good().Override(func(o *overrideFS) {
				o.File("docs/README.md").WithSize(200 * spectypes.MegaByte)
			}),
			valid: false,
		},
		{
			title: "totalSizeLimit exceeded",
			fsys: newMockFS().Good().WithFiles(
				newMockFile("docs/other.md").WithSize(140*spectypes.MegaByte),
				newMockFile("docs/someother.md").WithSize(140*spectypes.MegaByte),
			),
			valid: false,
		},
		{
			title: "totalContentsLimit exceeded",
			fsys: newMockFS().Good().Override(func(o *overrideFS) {
				o.File("docs").WithGeneratedFiles(70000, ".md", 512*spectypes.Byte)
			}),
			valid: false,
		},
		/* FIXME:
		{
			title: "relativePathSizeLimit exceeded",
			fsys: newMockFS().Good().Override(func(o *overrideFS) {
				o.File("img/kibana-system.png").WithSize(10 * spectypes.MegaByte)
			}),
			valid: false,
		},
		*/
		{
			title: "data streams limit exceeded",
			fsys: newMockFS().Good().Override(func(o *overrideFS) {
				o.MultiplyFile("data_stream", "foo", 1000)
			}),
			valid: false,
		},
		{
			title: "fieldsPerDataStreamLimit exceeded",
			fsys: newMockFS().Good().WithFiles(
				newMockFile("data_stream/foo/fields/many-fields.yml").WithContent(generateFields(2500)),
			),
			valid: false,
		},
		{
			title: "config template sizeLimit exceeded",
			fsys: newMockFS().Good().WithFiles(
				newMockFile("agent/input/stream.yml.hbs").WithSize(6 * spectypes.MegaByte),
			),
			valid: false,
		},
		{
			title: "ingest pipeline sizeLimit exceeded",
			fsys: newMockFS().Good().WithFiles(
				newMockFile("elasticsearch/ingest_pipeline/pipeline.yml").WithContent("---\n").WithSize(4 * spectypes.MegaByte),
			),
			valid: false,
		},
		{
			title: "ignore developer files",
			fsys: newMockFS().Good().WithFiles(
				newMockFile("_dev/deploy/docker/entrypoint.sh").WithSize(2048 * spectypes.MegaByte),
			),
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

//go:embed testdata/limits/manifest.yml
var manifestYml string

//go:embed testdata/limits/changelog.yml
var changelogYml string

//go:embed testdata/limits/data_stream/foo/manifest.yml
var datastreamManifestYml string

//go:embed testdata/limits/data_stream/foo/fields/base-fields.yml
var fieldsYml string

func generateFields(n int) string {
	var buf strings.Builder

	for i := 0; i < n; i++ {
		buf.WriteString(fmt.Sprintf("- name: generated.foo%d\n", i))
		buf.WriteString("  type: keyword\n")
	}
	return buf.String()
}

func (fs *mockFS) Good() *mockFS {
	return fs.WithFiles(
		newMockFile("manifest.yml").WithContent(manifestYml),
		newMockFile("changelog.yml").WithContent(changelogYml),
		newMockFile("docs/README.md").WithContent("## README"),
		newMockFile("img/kibana-system.png"),
		newMockFile("img/system.svg"),
		newMockFile("_dev/deploy/docker/docker-compose.yml").WithContent("version: 2.3"),
		newMockFile("data_stream/foo/manifest.yml").WithContent(datastreamManifestYml),
		newMockFile("data_stream/foo/fields/base-fields.yml").WithContent(fieldsYml),
	)
}

func (fs *mockFS) Override(overrider func(*overrideFS)) *mockFS {
	overrider(&overrideFS{fs})
	return fs
}

type overrideFS struct {
	fs *mockFS
}

func (o *overrideFS) File(name string) *mockFile {
	f, err := o.fs.root.findFile(name)
	if err != nil {
		panic(err)
	}
	return f
}

func (o *overrideFS) MultiplyFile(dir string, name string, times int) {
	d, err := o.fs.root.findFile(dir)
	if err != nil {
		panic(err)
	}

	f, err := d.findFile(name)
	if err != nil {
		panic(err)
	}

	for i := 0; i < times; i++ {
		cp := f.Copy()
		cp.stat.name = fmt.Sprintf("%s%d", f.stat.name, i)
		d.files = append(d.files, cp)
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

func (f *mockFile) Copy() *mockFile {
	cp := mockFile{}
	cp.stat = f.stat
	cp.content = f.content
	for _, src := range f.files {
		cp.files = append(cp.files, src.Copy())
	}
	return &cp
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
	for _, file := range files {
		f.addFileWithDirs(file)
	}
	return f
}

func (f *mockFile) WithGeneratedFiles(n int, suffix string, size spectypes.FileSize) *mockFile {
	var files []*mockFile
	for i := 0; i < n; i++ {
		files = append(files,
			newMockFile(fmt.Sprintf("tmp%d%s", i, suffix)).WithSize(size))
	}
	f.WithFiles(files...)
	return f
}

func (f *mockFile) addFileWithDirs(file *mockFile) {
	parts := strings.Split(file.stat.name, "/")
	dir := f
	for i, part := range parts[:len(parts)-1] {
		d, err := dir.findFile(part)
		if err == nil {
			if !d.stat.isDir {
				panic(path.Join(parts[:i]...) + " is not a directory")
			}
			dir = d
		} else {
			d = newMockDir(part)
			dir.files = append(dir.files, d)
			dir = d
		}
	}
	file.stat.name = parts[len(parts)-1]
	dir.files = append(dir.files, file)
}

func (f *mockFile) findFile(name string) (*mockFile, error) {
	if name == "." {
		return f, nil
	}
	name = path.Clean(name)
	parts := strings.SplitN(name, "/", 2)

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
