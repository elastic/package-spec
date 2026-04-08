// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func TestValidateFleetReservedVars(t *testing.T) {
	t.Run("non_input_or_integration_package_type_is_skipped", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: content
`), 0o644)
		require.NoError(t, err)

		errs := ValidateFleetReservedVars(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors for non-input/integration package types")
	})

	t.Run("non_reserved_variable_names_are_ignored", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
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

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: input
policy_templates:
  - name: sample
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
}
