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

func TestValidateIntegrationInputsDeprecation(t *testing.T) {

	t.Run("integration_deprecated_all_inputs_deprecated", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
deprecated:
  since: '1.0.0'
  description: 'This integration is deprecated.'
policy_templates:
  - inputs:
    - type: udp
      deprecated:
        since: '0.5.0'
        description: 'This input is deprecated.'
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationInputsDeprecation(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")

	})

	t.Run("integration_deprecated_none_inputs_deprecated", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
deprecated:
  since: '1.0.0'
  description: 'This integration is deprecated.'
policy_templates:
  - inputs:
    - type: udp
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationInputsDeprecation(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("integration_deprecated_partial_inputs_deprecated", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
deprecated:
  since: '1.0.0'
  description: 'This integration is deprecated.'
policy_templates:
  - inputs:
    - type: udp
      deprecated:
        since: '0.5.0'
        description: 'This input is deprecated.'
    - type: tcp
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationInputsDeprecation(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("integration_not_deprecated_all_inputs_deprecated", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - inputs:
    - type: udp
      deprecated:
        since: '0.5.0'
        description: 'This input is deprecated.'
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationInputsDeprecation(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		require.Len(t, errs, 1)
		assert.ErrorContains(t, errs[0], "all inputs are deprecated but the integration package is not marked as deprecated")

	})

	t.Run("integration_not_deprecated_none_inputs_deprecated", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - inputs:
    - type: udp
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationInputsDeprecation(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("integration_not_deprecated_partial_inputs_deprecated", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - inputs:
    - type: udp
      deprecated:
        since: '0.5.0'
        description: 'This input is deprecated.'
    - type: tcp
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationInputsDeprecation(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

}
