// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

func TestValidateIntegrationInputQualifier(t *testing.T) {
	tests := map[string]struct {
		manifest      string
		expectedErrs  []string
		expectedCodes []string
	}{
		"single_input_per_type_no_name": {
			manifest: `
type: integration
policy_templates:
  - name: nginx
    inputs:
      - type: logfile
        title: Logs
        description: Collect logs
      - type: httpjson
        title: Metrics
        description: Collect metrics
`,
		},
		"multiple_inputs_same_type_with_names": {
			manifest: `
type: integration
policy_templates:
  - name: nginx
    inputs:
      - name: filelog_otel
        type: otelcol
        title: Logs
        description: Collect logs
      - name: nginx_otel
        type: otelcol
        title: Metrics
        description: Collect metrics
`,
		},
		"multiple_inputs_same_type_no_names": {
			manifest: `
type: integration
policy_templates:
  - name: nginx
    inputs:
      - type: otelcol
        title: Logs
        description: Collect logs
      - type: otelcol
        title: Metrics
        description: Collect metrics
`,
			expectedErrs: []string{
				`policy template "nginx": input with type "otelcol" must have a name when multiple inputs of the same type are present`,
			},
			expectedCodes: []string{
				specerrors.CodeIntegrationInputQualifierRequired,
			},
		},
		"multiple_inputs_same_type_partial_names": {
			manifest: `
type: integration
policy_templates:
  - name: nginx
    inputs:
      - name: filelog_otel
        type: otelcol
        title: Logs
        description: Collect logs
      - type: otelcol
        title: Metrics
        description: Collect metrics
`,
			expectedErrs: []string{
				`policy template "nginx": input with type "otelcol" must have a name when multiple inputs of the same type are present`,
			},
			expectedCodes: []string{
				specerrors.CodeIntegrationInputQualifierRequired,
			},
		},
		"non_integration_package": {
			manifest: `
type: input
policy_templates:
  - name: sample
    inputs:
      - type: logfile
        title: Logs
        description: Collect logs
      - type: logfile
        title: More logs
        description: Collect more logs
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d := t.TempDir()
			err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(tc.manifest), 0o644)
			require.NoError(t, err)

			errs := ValidateIntegrationInputQualifier(fspath.DirFS(d))

			if len(tc.expectedErrs) == 0 {
				require.Empty(t, errs)
				return
			}

			require.Len(t, errs, len(tc.expectedErrs))
			for i, expected := range tc.expectedErrs {
				assert.True(t, strings.Contains(errs[i].Error(), expected),
					"error %q does not contain %q", errs[i].Error(), expected)
				assert.Equal(t, tc.expectedCodes[i], errs[i].Code())
			}
		})
	}
}
