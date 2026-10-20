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
	viewDir := filepath.Join(tmpDir, "elasticsearch", "esql_view")
	require.NoError(t, os.MkdirAll(viewDir, 0o755))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(viewDir, name), []byte(content), 0o644))
	}
	return fspath.DirFS(tmpDir)
}

func TestValidateEsqlViewNames(t *testing.T) {
	t.Run("no esql_view folder", func(t *testing.T) {
		assert.Empty(t, ValidateEsqlViewNames(fspath.DirFS(t.TempDir())))
	})

	t.Run("unique names", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"a.yml": "name: view_a\nquery: FROM logs-*\n",
			"b.yml": "name: view_b\nquery: FROM logs-*\n",
		})
		assert.Empty(t, ValidateEsqlViewNames(fsys))
	})

	t.Run("duplicate names", func(t *testing.T) {
		fsys := makeEsqlViewFS(t, map[string]string{
			"a.yml": "name: same\nquery: FROM logs-*\n",
			"b.yml": "name: same\nquery: FROM metrics-*\n",
		})
		errs := ValidateEsqlViewNames(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), `ES|QL view name "same" is already defined in file`)
		assert.Contains(t, errs[0].Error(), "SVR00013")
	})
}
