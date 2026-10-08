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

func makeWorkflowFS(t *testing.T, files map[string]string) fspath.FS {
	t.Helper()
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "kibana", "workflow")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	return fspath.DirFS(tmpDir)
}

func TestValidateWorkflowsEnabled(t *testing.T) {
	t.Run("no workflow folder", func(t *testing.T) {
		errs := ValidateWorkflowsEnabled(fspath.DirFS(t.TempDir()))
		assert.Empty(t, errs)
	})

	t.Run("enabled false", func(t *testing.T) {
		fsys := makeWorkflowFS(t, map[string]string{
			"mypkg-flow.yml": "name: flow\nenabled: false\n",
		})
		assert.Empty(t, ValidateWorkflowsEnabled(fsys))
	})

	t.Run("enabled absent", func(t *testing.T) {
		fsys := makeWorkflowFS(t, map[string]string{
			"mypkg-flow.yml": "name: flow\n",
		})
		assert.Empty(t, ValidateWorkflowsEnabled(fsys))
	})

	t.Run("enabled true", func(t *testing.T) {
		fsys := makeWorkflowFS(t, map[string]string{
			"mypkg-flow.yml": "name: flow\nenabled: true\n",
		})
		errs := ValidateWorkflowsEnabled(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "workflow should set enabled: false")
		assert.Contains(t, errs[0].Error(), "SVR00014")
	})

	t.Run("non-yml files are ignored", func(t *testing.T) {
		fsys := makeWorkflowFS(t, map[string]string{
			"mypkg-flow.txt": "enabled: true\n",
		})
		assert.Empty(t, ValidateWorkflowsEnabled(fsys))
	})
}
