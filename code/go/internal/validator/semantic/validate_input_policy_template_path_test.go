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

func TestValidateInputPolicyTemplates(t *testing.T) {

	t.Run("input_manifest_with_policy_template_success", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: udp
    template_path: udp.yml.hbs
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "agent", "input", "udp.yml.hbs"), []byte("# UDP template"), 0o644)
		require.NoError(t, err)

		errs := ValidateInputPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")

	})

	t.Run("input_manifest_with_policy_template_missing_template_path", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: udp
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputPolicyTemplates(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected no validation errors")

		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "is missing required field \"template_path\"")
	})

	t.Run("input_manifest_with_policy_template_missing_template_file", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: udp
    template_path: missing.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputPolicyTemplates(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorIs(t, errs[0], errTemplateNotFound)
	})
	t.Run("integration_manifest_with_policy_template_success", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - inputs:
    - title: Test UDP
      template_path: udp.yml.hbs
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "agent", "input", "udp.yml.hbs"), []byte("# UDP template"), 0o644)
		require.NoError(t, err)

		errs := ValidateInputPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("integration_manifest_with_policy_template_invalid", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - inputs:
    - title: Test UDP
      template_path: missing.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputPolicyTemplates(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorIs(t, errs[0], errTemplateNotFound)
	})
}
