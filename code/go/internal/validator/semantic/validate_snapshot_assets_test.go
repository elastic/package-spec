// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/elastic/package-spec/code/go/internal/pkgpath"
)

func TestReadMigrationVersionField(t *testing.T) {
	cases := []struct {
		title     string
		undefined bool
		version   string
		expected  string
	}{
		{
			title:    "migration field exists",
			version:  "8.1.0",
			expected: "8.1.0",
		},
		{
			title:    "migration field exists",
			version:  "",
			expected: "",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			assetContents := "{}"
			if c.version != "" {
				assetContents = fmt.Sprintf(`{"migrationVersion": { "asset": "%s"} }`, c.version)
			}

			objectFile, err := createObjectFile(t, assetContents, "asset.json")
			require.NoError(t, err)

			version, err := readMigrationVersionField(*objectFile)
			require.NoError(t, err)
			assert.Equal(t, c.expected, version)
		})
	}
}

func createObjectFile(t *testing.T, contents, filename string) (*pkgpath.File, error) {
	dir := t.TempDir()
	fsysDir := fspath.DirFS(dir)
	assetFile := filepath.Join(dir, filename)
	f, err := os.Create(assetFile)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	if _, err = f.WriteString(contents); err != nil {
		return nil, nil
	}

	objectFiles, err := pkgpath.Files(fsysDir, filename)
	if err != nil {
		return nil, err
	}

	return &objectFiles[0], nil
}

func TestUsingSnapshotVersion(t *testing.T) {
	cases := []struct {
		title    string
		versions []string
		expected bool
	}{
		{
			title:    "stable release",
			versions: []string{"8.1.0"},
			expected: false,
		},
		{
			title:    "empty version",
			versions: []string{},
			expected: false,
		},
		{
			title:    "snapshot version",
			versions: []string{"8.1.0-SNAPSHOT"},
			expected: true,
		},
		{
			title:    "different versions",
			versions: []string{"8.1.0-SNAPSHOT", "8.2.0"},
			expected: true,
		},
		{
			title:    "other prerelease",
			versions: []string{"8.1.0-rc1"},
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			snapshot, err := usingSnapshotVersion(c.versions)
			require.NoError(t, err)
			assert.Equal(t, c.expected, snapshot)
		})
	}
}
