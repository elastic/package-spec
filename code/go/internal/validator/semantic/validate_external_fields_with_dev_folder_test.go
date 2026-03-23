// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
)

func TestReadDevBuildDependenciesKeys(t *testing.T) {
	tempDir := t.TempDir()
	tests := []struct {
		title    string
		contents string
		expected []string
	}{
		{
			"keys defined",
			"dependencies:\n  ecs: \"a\"\n  foo: 4\n",
			[]string{"ecs", "foo"},
		},
		{
			"empty",
			"dependencies: {}\n",
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			fsys := fspath.DirFS(tempDir)
			buildFilePath := filepath.Join(tempDir, "build.yml")
			err := os.WriteFile(buildFilePath, []byte(test.contents), 0644)
			require.NoError(t, err)

			f, err := pkgpath.Files(fsys, "build.yml")
			require.Len(t, f, 1)
			require.NoError(t, err)

			list, err := readDevBuildDependenciesKeys(f[0])
			require.NoError(t, err)

			sort.Strings(list)
			assert.Equal(t, test.expected, list)
		})
	}

}
