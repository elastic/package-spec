// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func makeEsqlViewFS(t *testing.T, files map[string]string) fspath.FS {
	t.Helper()
	tmpDir := t.TempDir()
	viewDir := filepath.Join(tmpDir, "elasticsearch", "esql_view")
	require.NoError(t, os.MkdirAll(viewDir, 0o755))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(viewDir, name), []byte(content), 0o644))
	}
	return fspath.DirFS(tmpDir)
}

func TestValidateEsqlViews(t *testing.T) {
	t.Run("no esql_view folder", func(t *testing.T) {
		fsys := fspath.DirFS(t.TempDir())
		errs := ValidateEsqlViews(fsys)
		assert.Empty(t, errs)
	})

	t.Run("valid view", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"mypkg-access_logs.yml": "name: mypkg-access_logs\nquery: FROM logs-*\n",
		})
		errs := ValidateEsqlViews(fsys)
		assert.Empty(t, errs)
	})

	t.Run("name does not match filename stem", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"mypkg-access_logs.yml": "name: other_name\nquery: FROM logs-*\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), `name field "other_name" must equal the filename stem "mypkg-access_logs"`)
		assert.Contains(t, errs[0].Error(), "SVR00012")
	})

	t.Run("duplicate name across files", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"mypkg-view_a.yml": "name: mypkg-view_a\nquery: FROM logs-*\n",
			"mypkg-view_b.yml": "name: mypkg-view_a\nquery: FROM metrics-*\n",
		})
		errs := ValidateEsqlViews(fsys)
		// view_b has two errors: name != stem and duplicate
		require.GreaterOrEqual(t, len(errs), 1)
		errMessages := make([]string, len(errs))
		for i, e := range errs {
			errMessages[i] = e.Error()
		}
		hasDuplicate := false
		for _, msg := range errMessages {
			if strings.Contains(msg, "duplicate view name") {
				hasDuplicate = true
			}
		}
		assert.True(t, hasDuplicate, "expected a duplicate name error")
	})

	t.Run("reserved names", func(t *testing.T) {
		// Use filenames whose stems equal the reserved name so the stem-mismatch
		// check does not fire — only the reserved-name check should.
		cases := []struct {
			file string
			name string
		}{
			{"..yml", "."},
			{"...yml", ".."},
		}
		for _, tc := range cases {
			fsys := makeEsqlViewFS(t, map[string]string{
				tc.file: "name: " + tc.name + "\nquery: FROM logs-*\n",
			})
			errs := ValidateEsqlViews(fsys)
			hasReserved := false
			for _, e := range errs {
				if strings.Contains(e.Error(), "is not allowed") {
					hasReserved = true
				}
			}
			assert.True(t, hasReserved, "expected a reserved-name error for name %q", tc.name)
		}
	})


	t.Run("whitespace-only query", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"mypkg-view.yml": "name: mypkg-view\nquery: \"   \"\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "query must not be empty or whitespace-only")
		assert.Contains(t, errs[0].Error(), "SVR00012")
	})

	t.Run("empty query", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"mypkg-view.yml": "name: mypkg-view\nquery: \"\"\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "query must not be empty or whitespace-only")
	})

	t.Run("non-yml files are ignored", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"mypkg-view.yml":  "name: mypkg-view\nquery: FROM logs-*\n",
			"mypkg-other.txt": "not a view",
		})
		errs := ValidateEsqlViews(fsys)
		assert.Empty(t, errs)
	})
}
