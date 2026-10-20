// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func makeEsqlViewFS(t *testing.T, files map[string]string) fspath.FS {
	t.Helper()
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "manifest.yml"), []byte("name: mypkg\n"), 0o644))
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
			"access_logs.yml": "name: $.integrations.mypkg.access_logs\nquery: FROM logs-*\n",
		})
		errs := ValidateEsqlViews(fsys)
		assert.Empty(t, errs)
	})

	t.Run("name does not match filename stem", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"access_logs.yml": "name: $.integrations.mypkg.other_name\nquery: FROM logs-*\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), `name field "$.integrations.mypkg.other_name" must be "$.integrations.mypkg.access_logs"`)
		assert.Contains(t, errs[0].Error(), "SVR00013")
	})

	t.Run("name without the namespace prefix", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"access_logs.yml": "name: access_logs\nquery: FROM logs-*\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), `must be "$.integrations.mypkg.access_logs"`)
	})

	t.Run("name with another package name", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"access_logs.yml": "name: $.integrations.otherpkg.access_logs\nquery: FROM logs-*\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), `must be "$.integrations.mypkg.access_logs"`)
	})

	t.Run("missing manifest", func(t *testing.T) {
		tmpDir := t.TempDir()
		viewDir := filepath.Join(tmpDir, "elasticsearch", "esql_view")
		require.NoError(t, os.MkdirAll(viewDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(viewDir, "view.yml"), []byte("name: x\nquery: FROM logs-*\n"), 0o644))
		errs := ValidateEsqlViews(fspath.DirFS(tmpDir))
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "manifest")
	})

	t.Run("whitespace-only query", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"view.yml": "name: $.integrations.mypkg.view\nquery: \"   \"\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "query must not be empty or whitespace-only")
		assert.Contains(t, errs[0].Error(), "SVR00013")
	})

	t.Run("empty query", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"view.yml": "name: $.integrations.mypkg.view\nquery: \"\"\n",
		})
		errs := ValidateEsqlViews(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "query must not be empty or whitespace-only")
	})

	t.Run("non-yml files are ignored", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"view.yml":  "name: $.integrations.mypkg.view\nquery: FROM logs-*\n",
			"other.txt": "not a view",
		})
		errs := ValidateEsqlViews(fsys)
		assert.Empty(t, errs)
	})
}
