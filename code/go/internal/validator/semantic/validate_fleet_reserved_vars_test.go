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

func TestValidateFleetReservedVars(t *testing.T) {
	t.Run("non_input_or_integration_package_type_is_skipped", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: content
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors for non-input/integration package types")
	})

	t.Run("non_reserved_variable_names_are_ignored", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: sample
    input: otelcol
    vars:
      - name: foobar
        type: bool
      - name: some_other_var
        type: text
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors for non-reserved variable names")
	})

	t.Run("use_apm_wrong_input_and_wrong_type_both_reported", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: sample
    type: traces
    input: logfile
    vars:
      - name: use_apm
        type: text
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Len(t, errs, 2, "expected both input type and variable type violations to be reported")
		assert.Contains(t, errs[0].Error(), `variable "use_apm" must be "otelcol" input, got "logfile"`)
		assert.Contains(t, errs[1].Error(), `variable "use_apm" must be type "bool", got "text"`)
	})

	t.Run("use_apm_wrong_eligibility_reported", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: sample
    type: logs
    input: otelcol
    vars:
      - name: use_apm
        type: bool
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Len(t, errs, 1, "expected one eligibility violation to be reported")
		assert.Contains(t, errs[0].Error(), `variable "use_apm" must be "traces" data stream type or "dynamic_signal_types: true", got "logs" data stream type`)
	})

	t.Run("use_apm_dynamic_signal_types_bypasses_eligibility", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
policy_templates:
  - name: sample
    type: logs
    input: otelcol
    dynamic_signal_types: true
    vars:
      - name: use_apm
        type: bool
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Empty(t, errs, "expected no errors when dynamic_signal_types is true")
	})

	t.Run("reserved_var_at_input_package_root_level_is_flagged", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: input
vars:
  - name: data_stream.dataset
    type: text
policy_templates:
  - name: sample
    type: traces
    input: otelcol
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Len(t, errs, 1, "expected one scope violation for root-level reserved var in input package")
		assert.Contains(t, errs[0].Error(), `package root vars: variable "data_stream.dataset" must only be declared at stream level`)
	})

	t.Run("reserved_vars_at_integration_non_stream_scopes_are_flagged", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
vars:
  - name: use_apm
    type: bool
policy_templates:
  - name: sample
    vars:
      - name: data_stream.dataset
        type: text
    inputs:
      - type: logfile
        vars:
          - name: use_apm
            type: bool
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Len(t, errs, 3, "expected scope violations at root, policy template, and input levels")
		assert.Contains(t, errs[0].Error(), `package root vars: variable "use_apm" must only be declared at stream level`)
		assert.Contains(t, errs[1].Error(), `policy template "sample" vars: variable "data_stream.dataset" must only be declared at stream level`)
		assert.Contains(t, errs[2].Error(), `policy template "sample" input "logfile" vars: variable "use_apm" must only be declared at stream level`)
	})
}
