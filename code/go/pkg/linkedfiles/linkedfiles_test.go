// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package linkedfiles

import (
	"bytes"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkUpdateChecksum(t *testing.T) {
	root, err := FindRepositoryRoot()
	assert.NoError(t, err)

	outdatedFile, err := newLinkedFile(root, "code/go/pkg/linkedfiles/testdata/links/outdated.yml.link")
	t.Cleanup(func() {
		_ = WriteFile(outdatedFile.LinkFilePath, []byte(outdatedFile.IncludedFilePath))
	})
	assert.NoError(t, err)
	assert.False(t, outdatedFile.UpToDate)
	assert.Empty(t, outdatedFile.LinkChecksum)
	updated, err := outdatedFile.UpdateChecksum()
	assert.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, "d709feed45b708c9548a18ca48f3ad4f41be8d3f691f83d7417ca902a20e6c1e", outdatedFile.LinkChecksum)
	assert.True(t, outdatedFile.UpToDate)

	uptodateFile, err := newLinkedFile(root, "code/go/pkg/linkedfiles/testdata/links/uptodate.yml.link")
	assert.NoError(t, err)
	assert.True(t, uptodateFile.UpToDate)
	updated, err = uptodateFile.UpdateChecksum()
	assert.NoError(t, err)
	assert.False(t, updated)
}

func TestLinkReplaceTargetFilePathDirectory(t *testing.T) {
	root, err := FindRepositoryRoot()
	assert.NoError(t, err)

	linkedFile, err := newLinkedFile(root, "code/go/pkg/linkedfiles/testdata/links/uptodate.yml.link")
	assert.NoError(t, err)
	assert.Equal(t, "code/go/pkg/linkedfiles/testdata/links/uptodate.yml", linkedFile.TargetFilePath)

	linkedFile.ReplaceTargetFilePathDirectory("code/go/pkg/linkedfiles/testdata/links", "code/go/pkg/linkedfiles/build/testdata/links")
	assert.Equal(t, "code/go/pkg/linkedfiles/build/testdata/links/uptodate.yml", linkedFile.TargetFilePath)
}

func TestListLinkedFiles(t *testing.T) {
	root, err := FindRepositoryRoot()
	assert.NoError(t, err)
	linkedFiles, err := ListLinkedFilesInRoot(root, "code/go/pkg/linkedfiles/testdata/links")
	assert.NoError(t, err)
	assert.NotEmpty(t, linkedFiles)
	assert.Len(t, linkedFiles, 2)
	assert.Equal(t, "code/go/pkg/linkedfiles/testdata/links/outdated.yml.link", linkedFiles[0].LinkFilePath)
	assert.Empty(t, linkedFiles[0].LinkChecksum)
	assert.Equal(t, "code/go/pkg/linkedfiles/testdata/links/outdated.yml", linkedFiles[0].TargetFilePath)
	assert.Equal(t, "code/go/pkg/linkedfiles/testdata/links/included.yml", linkedFiles[0].IncludedFilePath)
	assert.Equal(t, "d709feed45b708c9548a18ca48f3ad4f41be8d3f691f83d7417ca902a20e6c1e", linkedFiles[0].IncludedFileContentsChecksum)
	assert.False(t, linkedFiles[0].UpToDate)
	assert.Equal(t, "code/go/pkg/linkedfiles/testdata/links/uptodate.yml.link", linkedFiles[1].LinkFilePath)
	assert.Equal(t, "d709feed45b708c9548a18ca48f3ad4f41be8d3f691f83d7417ca902a20e6c1e", linkedFiles[1].LinkChecksum)
	assert.Equal(t, "code/go/pkg/linkedfiles/testdata/links/uptodate.yml", linkedFiles[1].TargetFilePath)
	assert.Equal(t, "code/go/pkg/linkedfiles/testdata/links/included.yml", linkedFiles[1].IncludedFilePath)
	assert.Equal(t, "d709feed45b708c9548a18ca48f3ad4f41be8d3f691f83d7417ca902a20e6c1e", linkedFiles[1].IncludedFileContentsChecksum)
	assert.True(t, linkedFiles[1].UpToDate)
}

func TestCopyFileFromRoot(t *testing.T) {
	root, err := FindRepositoryRoot()
	assert.NoError(t, err)
	fileA := "fileA.txt"
	fileB := "fileB.txt"
	t.Cleanup(func() { _ = root.Remove(fileA) })
	t.Cleanup(func() { _ = root.Remove(fileB) })

	createDummyFile(t, root, fileA, "This is the content of the file.")

	assert.NoError(t, CopyFileFromRoot(root, fileA, fileB))

	equal, err := filesEqual(root, fileA, fileB)
	assert.NoError(t, err)
	assert.True(t, equal, "files should be equal after copying")
}

func createDummyFile(t *testing.T, root *os.Root, filename, content string) {
	file, err := root.Create(filename)
	assert.NoError(t, err)
	defer file.Close()
	_, err = file.WriteString(content)
	assert.NoError(t, err)
}

func filesEqual(root *os.Root, file1, file2 string) (bool, error) {
	f1, err := root.FS().(fs.ReadFileFS).ReadFile(file1)
	if err != nil {
		return false, err
	}

	f2, err := root.FS().(fs.ReadFileFS).ReadFile(file2)
	if err != nil {
		return false, err
	}

	return bytes.Equal(f1, f2), nil
}
