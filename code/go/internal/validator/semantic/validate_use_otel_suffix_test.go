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
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

func TestValidateUseOtelSuffix(t *testing.T) {
	tests := map[string]struct {
		manifest      string
		expectError   bool
		errorContains string
	}{
		"valid_without_streams": {
			manifest: `title: Logs
type: logs
use_otel_suffix: true
`,
		},
		"valid_false_with_input": {
			manifest: `title: Logs
type: logs
use_otel_suffix: false
streams:
  - input: logfile
    title: Logs
    description: Collect logs
`,
		},
		"valid_omitted_with_input": {
			manifest: `title: Logs
type: logs
streams:
  - input: logfile
    title: Logs
    description: Collect logs
`,
		},
		"valid_true_with_package_reference_only": {
			manifest: `title: Logs
type: logs
use_otel_suffix: true
streams:
  - package: filelog_otel
    title: Logs
    description: Collect logs
`,
		},
		"invalid_true_with_logfile_input": {
			manifest: `title: Logs
type: logs
use_otel_suffix: true
streams:
  - input: logfile
    title: Logs
    description: Collect logs
`,
			expectError:   true,
			errorContains: `use_otel_suffix is only allowed on data streams without inputs (inputs defined: "logfile")`,
		},
		"invalid_true_with_otelcol_input": {
			manifest: `title: Logs
type: logs
use_otel_suffix: true
streams:
  - input: otelcol
    title: Logs
    description: Collect logs
`,
			expectError:   true,
			errorContains: `use_otel_suffix is only allowed on data streams without inputs (inputs defined: "otelcol"). If this data stream needs an input, use the otelcol input type instead`,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			packageRoot := t.TempDir()
			manifestDir := filepath.Join(packageRoot, "data_stream", "logs")
			require.NoError(t, os.MkdirAll(manifestDir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(manifestDir, "manifest.yml"), []byte(testCase.manifest), 0o644))

			errs := ValidateUseOtelSuffix(fspath.DirFS(packageRoot))
			if testCase.expectError {
				require.NotEmpty(t, errs)
				assert.ErrorContains(t, errs, testCase.errorContains)
				assert.Equal(t, specerrors.CodeUseOtelSuffixWithInput, errs[0].Code())
				return
			}
			require.Empty(t, errs)
		})
	}
}
