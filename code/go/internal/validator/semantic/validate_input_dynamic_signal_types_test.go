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

func TestValidateInputDynamicSignalTypes(t *testing.T) {

	t.Run("valid_otelcol_with_dynamic_signal_types_true", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
format_version: 3.6.0
type: input
policy_templates:
  - name: otel_logs
    input: otelcol
    template_path: input.yml.hbs
    dynamic_signal_types: true
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("valid_otelcol_with_dynamic_signal_types_false", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
format_version: 3.6.0
type: input
policy_templates:
  - name: otel_logs
    input: otelcol
    template_path: input.yml.hbs
    dynamic_signal_types: false
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("valid_otelcol_without_dynamic_signal_types", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
format_version: 3.6.0
type: input
policy_templates:
  - name: otel_logs
    input: otelcol
    template_path: input.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("invalid_non_otelcol_with_dynamic_signal_types_true", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
format_version: 3.6.0
type: input
policy_templates:
  - name: logfile
    input: logfile
    template_path: input.yml.hbs
    dynamic_signal_types: true
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "dynamic_signal_types is only allowed when input is 'otelcol'")
		assert.Contains(t, errs[0].Error(), "got 'logfile'")
	})

	t.Run("valid_non_input_package_without_dynamic_signal_types", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: integration
policy_templates:
  - name: apache
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors for non-input packages without field")
	})

	t.Run("invalid_integration_package_non_otelcol_with_dynamic_signal_types", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: integration
policy_templates:
  - name: apache
    inputs:
      - type: logfile
        title: Log Files
        description: Collect logs from files
        dynamic_signal_types: true
`), 0o644)
		require.NoError(t, err)

		// Create empty data_stream directory
		err = os.Mkdir(d+"/data_stream", 0o755)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors for non-otelcol with dynamic_signal_types")
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "dynamic_signal_types is only allowed when input is 'otelcol'")
	})

	t.Run("valid_non_otelcol_without_dynamic_signal_types", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: input
policy_templates:
  - name: logfile
    input: logfile
    template_path: input.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors")
	})

	t.Run("invalid_multiple_templates_one_invalid", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
format_version: 3.6.0
type: input
policy_templates:
  - name: otel_logs
    input: otelcol
    template_path: input.yml.hbs
    dynamic_signal_types: true
  - name: file_logs
    input: logfile
    template_path: input2.yml.hbs
    dynamic_signal_types: true
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors")
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "dynamic_signal_types is only allowed when input is 'otelcol'")
		assert.Contains(t, errs[0].Error(), "file_logs")
		assert.Contains(t, errs[0].Error(), "got 'logfile'")
	})

	t.Run("valid_type_field_without_dynamic_signal_types", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: input
policy_templates:
  - name: otel_logs
    type: logs
    input: otelcol
    template_path: input.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors when type is present without dynamic_signal_types")
	})

	t.Run("valid_type_field_with_dynamic_signal_types_false", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: input
policy_templates:
  - name: otel_logs
    type: logs
    input: otelcol
    template_path: input.yml.hbs
    dynamic_signal_types: false
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors when type is present with dynamic_signal_types: false")
	})

	t.Run("valid_integration_data_stream_with_otelcol_and_dynamic_signal_types", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: integration
`), 0o644)
		require.NoError(t, err)

		// Create data_stream directory and manifest
		err = os.Mkdir(d+"/data_stream", 0o755)
		require.NoError(t, err)
		err = os.Mkdir(d+"/data_stream/logs", 0o755)
		require.NoError(t, err)

		err = os.WriteFile(d+"/data_stream/logs/manifest.yml", []byte(`
streams:
  - input: otelcol
    title: OTel Logs
    description: Collect logs via OTel
    dynamic_signal_types: true
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.Empty(t, errs, "expected no validation errors for data stream with otelcol")
	})

	t.Run("invalid_integration_data_stream_non_otelcol_with_dynamic_signal_types", func(t *testing.T) {
		d := t.TempDir()

		err := os.WriteFile(d+"/manifest.yml", []byte(`
type: integration
`), 0o644)
		require.NoError(t, err)

		// Create data_stream directory and manifest
		err = os.Mkdir(d+"/data_stream", 0o755)
		require.NoError(t, err)
		err = os.Mkdir(d+"/data_stream/logs", 0o755)
		require.NoError(t, err)

		err = os.WriteFile(d+"/data_stream/logs/manifest.yml", []byte(`
streams:
  - input: logfile
    title: Log Files
    description: Collect logs from files
    dynamic_signal_types: true
`), 0o644)
		require.NoError(t, err)

		errs := ValidateInputDynamicSignalTypes(fspath.DirFS(d))
		require.NotEmpty(t, errs, "expected validation errors for data stream with non-otelcol")
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "dynamic_signal_types is only allowed when input is 'otelcol'")
	})
}
