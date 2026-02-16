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

func TestValidatePackageReferences(t *testing.T) {
	t.Run("valid_policy_template_package_reference", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
format_version: 3.6.0
type: integration
requires:
  input:
    - name: filelog_otel
      version: "^1.0.0"
policy_templates:
  - name: apache
    inputs:
      - package: filelog_otel
        title: Collect logs
        description: Collecting Apache logs
`), 0o644)
		require.NoError(t, err)

		errs := ValidatePackageReferences(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("invalid_policy_template_package_reference", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
format_version: 3.6.0
type: integration
requires:
  input:
    - name: filelog_otel
      version: "^1.0.0"
policy_templates:
  - name: apache
    inputs:
      - package: missing_package
        title: Collect logs
        description: Collecting Apache logs
`), 0o644)
		require.NoError(t, err)

		errs := ValidatePackageReferences(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, `policy_templates[0].inputs[0] references package "missing_package" which is not listed in requires section`)
	})

	t.Run("valid_datastream_package_reference", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
format_version: 3.6.0
type: integration
requires:
  input:
    - name: filelog_otel
      version: "^1.0.0"
`), 0o644)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Join(d, "data_stream", "logs"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
title: Apache logs
type: logs
streams:
  - package: filelog_otel
    title: Apache Logs
    description: Collect Apache logs
`), 0o644)
		require.NoError(t, err)

		errs := ValidatePackageReferences(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("invalid_datastream_package_reference", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
format_version: 3.6.0
type: integration
requires:
  input:
    - name: filelog_otel
      version: "^1.0.0"
`), 0o644)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Join(d, "data_stream", "logs"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
title: Apache logs
type: logs
streams:
  - package: missing_package
    title: Apache Logs
    description: Collect Apache logs
`), 0o644)
		require.NoError(t, err)

		errs := ValidatePackageReferences(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, `streams[0] references package "missing_package" which is not listed in manifest requires section`)
	})

	t.Run("no_requires_section", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
format_version: 3.6.0
type: integration
policy_templates:
  - name: apache
    inputs:
      - package: some_package
        title: Collect logs
        description: Collecting Apache logs
`), 0o644)
		require.NoError(t, err)

		errs := ValidatePackageReferences(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, `policy_templates[0].inputs[0] references package "some_package" which is not listed in requires section`)
	})

	t.Run("multiple_invalid_references", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
format_version: 3.6.0
type: integration
requires:
  input:
    - name: valid_package
      version: "^1.0.0"
policy_templates:
  - name: apache
    inputs:
      - package: missing_package_1
        title: First input
        description: First description
      - package: missing_package_2
        title: Second input
        description: Second description
`), 0o644)
		require.NoError(t, err)

		errs := ValidatePackageReferences(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 2)
		assert.ErrorContains(t, errs, `policy_templates[0].inputs[0] references package "missing_package_1"`)
		assert.ErrorContains(t, errs, `policy_templates[0].inputs[1] references package "missing_package_2"`)
	})
}
