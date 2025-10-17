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

func TestValidateInputPackagesPolicyTemplates(t *testing.T) {

	t.Run("policy_templates_have_template_path", func(t *testing.T) {
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

		errs := ValidateInputPackagesPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")

	})

	t.Run("policy_templates_empty_template_path", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: udp
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputPackagesPolicyTemplates(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected no validation errors")

		assert.Len(t, errs, 1)
		assert.ErrorIs(t, errs[0], errRequiredTemplatePath)
	})

	t.Run("policy_templates_missing_template_path", func(t *testing.T) {
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

		errs := ValidateInputPackagesPolicyTemplates(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorIs(t, errs[0], errTemplateNotFound)
	})

	t.Run("not_input_package_type", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - name: udp
    template_path: missing.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputPackagesPolicyTemplates(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.ErrorIs(t, errs[0], errInvalidPackageType)
	})

}
func TestValidateInputPackagePolicyTemplate(t *testing.T) {
	t.Run("existing_template_found", func(t *testing.T) {
		d := t.TempDir()

		// create directories and template file that should match the pattern "*/*/tmpl"
		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "agent", "input", "udp.yml.hbs"), []byte("# UDP template"), 0o644)
		require.NoError(t, err)

		err = validateInputPackagePolicyTemplate(fspath.DirFS(d), inputPolicyTemplate{
			TemplatePath: "udp.yml.hbs",
		})
		require.NoError(t, err)
	})

	t.Run("existing_template_found_nested", func(t *testing.T) {
		d := t.TempDir()

		// create directories and template file that should match the pattern "*/*/tmpl"
		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "agent", "stream", "udp.yml.hbs"), []byte("# UDP template"), 0o644)
		require.NoError(t, err)

		err = validateInputPackagePolicyTemplate(fspath.DirFS(d), inputPolicyTemplate{
			TemplatePath: "udp.yml.hbs",
		})
		require.NoError(t, err)
	})

	t.Run("template_not_found", func(t *testing.T) {
		d := t.TempDir()

		// no template file created
		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)

		err = validateInputPackagePolicyTemplate(fspath.DirFS(d), inputPolicyTemplate{
			TemplatePath: "missing.yml.hbs",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, errTemplateNotFound)
	})
}
