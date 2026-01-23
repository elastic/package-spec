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

func TestValidatePackageManifestDeprecatedReplacedBy(t *testing.T) {
	t.Run("integration_deprecated_replaced_by_package", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
deprecated:
  since: '1.0.0'
  description: 'This integration is deprecated.'
  replaced_by:
    package: 'new-integration'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("integration_deprecated_replaced_by_error", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
deprecated:
  since: '1.0.0'
  description: 'This integration is deprecated.'
  replaced_by:
    input: 'new-input'
    policy_template: 'new-policy-template'
    variable: 'new-variable'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, "deprecated.replaced_by.package must be specified when deprecated.replaced_by is used")
	})

	t.Run("integration_policy_template_deprecated_replaced_by_policy_template", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - deprecated:
      since: '1.0.0'
      description: 'This policy template is deprecated.'
      replaced_by:
        package: 'new-integration'
        policy_template: 'new-policy-template'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("integration_policy_template_deprecated_replaced_by_error", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - deprecated:
      since: '1.0.0'
      description: 'This policy template is deprecated.'
      replaced_by:
        input: 'new-input'
        variable: 'new-variable'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, "policy_template deprecated.replaced_by.policy_template must be specified when deprecated.replaced_by is used")
	})

	t.Run("integration_input_deprecated_replaced_by_input", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - inputs:
    - name: input_example
      deprecated:
        since: '1.0.0'
        description: 'This input is deprecated.'
        replaced_by:
          package: 'new-integration'
          input: 'new-input'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("integration_input_deprecated_replaced_by_error", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - inputs:
    - name: input_example
      deprecated:
        since: '1.0.0'
        description: 'This input is deprecated.'
        replaced_by:
          package: 'new-integration'
          policy_template: 'new-policy-template'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, "input deprecated.replaced_by.input must be specified when deprecated.replaced_by is used")
	})

	t.Run("input_policy_template_deprecated_replaced_by_policy_template", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - deprecated:
      since: '1.0.0'
      description: 'This policy template is deprecated.'
      replaced_by:
        policy_template: 'new-policy-template'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("input_policy_template_deprecated_replaced_by_policy_template_error", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - deprecated:
      since: '1.0.0'
      description: 'This policy template is deprecated.'
      replaced_by:
        input: 'new-policy-template'
`), 0o644)
		require.NoError(t, err)

		errs := validatePackageManifestDeprecatedReplacedBy(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, "policy_template deprecated.replaced_by.policy_template must be specified when deprecated.replaced_by is used")
	})
}

func TestValidateDataStreamsDeprecatedReplacedBy(t *testing.T) {

	t.Run("data_stream_deprecated_replaced_by_data_stream", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "data_stream", "test"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "manifest.yml"), []byte(`
deprecated:
  since: '1.0.0'
  description: 'This data stream is deprecated.'
  replaced_by:
    data_stream: 'new-data-stream'
streams:
  - vars:
    - name: var_example
`), 0o644)
		require.NoError(t, err)

		errs := validateDataStreamsDeprecatedReplacedBy(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("data_stream_deprecated_replaced_by_data_stream_error", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "data_stream", "test"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "manifest.yml"), []byte(`
deprecated:
  since: '1.0.0'
  description: 'This data stream is deprecated.'
  replaced_by:
    variable: 'new-data-stream'
streams:
  - vars:
    - name: var_example
`), 0o644)
		require.NoError(t, err)

		errs := validateDataStreamsDeprecatedReplacedBy(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, "deprecated.replaced_by.data_stream must be specified when deprecated.replaced_by is used")
	})
	t.Run("stream_deprecated_variable_replaced_by_variable", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "data_stream", "test"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "manifest.yml"), []byte(`
streams:
  - vars:
    - name: var_example
      deprecated:
        since: '1.0.0'
        description: 'This variable is deprecated.'
        replaced_by:
          variable: 'new-variable'
`), 0o644)
		require.NoError(t, err)

		errs := validateDataStreamsDeprecatedReplacedBy(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("stream_deprecated_variable_replaced_by_error", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "data_stream", "test"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "manifest.yml"), []byte(`
streams:
  - vars:
    - name: var_example
      deprecated:
        since: '1.0.0'
        description: 'This variable is deprecated.'
        replaced_by:
          package: 'new-package'
`), 0o644)
		require.NoError(t, err)

		errs := validateDataStreamsDeprecatedReplacedBy(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorContains(t, errs, "variable deprecated.replaced_by.variable must be specified when deprecated.replaced_by is used")
	})

}
