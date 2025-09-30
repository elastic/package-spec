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

func TestValidateStreamTemplates(t *testing.T) {

	t.Run("valid_data_stream_with_templates", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "manifest.yml"), []byte(`
streams:
  - input: udp
    template_path: udp.yml.hbs
    title: Test UDP
    description: Test UDP stream
`), 0o644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "agent", "stream", "udp.yml.hbs"), []byte("# UDP template"), 0o644)
		require.NoError(t, err)

		errs := ValidateStreamTemplates(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")

	})

	t.Run("missing_template_file", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "manifest.yml"), []byte(`
streams:
  - input: udp
    template_path: missing.yml.hbs
    title: Test UDP
    description: Test UDP stream
`), 0o644)
		require.NoError(t, err)

		errs := ValidateStreamTemplates(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Contains(t, errs.Error(), "references template_path \"missing.yml.hbs\": open "+filepath.Join("data_stream", "test", "agent", "stream", "missing.yml.hbs")+": no such file or directory")
	})

}
