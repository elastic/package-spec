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
)

func TestValidateStreamInputBundled(t *testing.T) {
	tests := []struct {
		name string
		// files maps relative path → YAML content written under a temp dir.
		files         map[string]string
		expectErrors  bool
		errorContains []string
	}{
		// ---------------------------------------------------------------
		// Data stream manifest: happy paths
		// ---------------------------------------------------------------
		{
			name: "data stream with input: set — no errors",
			files: map[string]string{
				"data_stream/logs/manifest.yml": "title: Logs\ntype: logs\nstreams:\n  - input: logfile\n    title: Logs\n    description: Collect logs\n",
			},
			expectErrors: false,
		},
		{
			name: "no data_stream directory — no errors",
			files: map[string]string{
				"manifest.yml": "type: integration\n",
			},
			expectErrors: false,
		},
		{
			name: "data stream manifest with no streams array — no errors",
			files: map[string]string{
				"data_stream/logs/manifest.yml": "title: Logs\ntype: logs\n",
			},
			expectErrors: false,
		},
		// ---------------------------------------------------------------
		// Data stream manifest: error cases
		// ---------------------------------------------------------------
		{
			name: "data stream stream has package: — rejected",
			files: map[string]string{
				"data_stream/logs/manifest.yml": "title: Logs\ntype: logs\nstreams:\n  - package: filelog_otel\n    title: Logs\n    description: Collect logs\n",
			},
			expectErrors: true,
			errorContains: []string{
				"stream[0]",
				"'package:'",
				"source-only",
				"build packages must use 'input:'",
			},
		},
		{
			// Schema validation (oneOf: input|package) catches this before the
			// semantic layer runs; the semantic check only looks for 'package:'.
			name: "data stream stream missing input: — not caught by semantic layer",
			files: map[string]string{
				"data_stream/logs/manifest.yml": "title: Logs\ntype: logs\nstreams:\n  - title: Logs\n    description: Collect logs\n",
			},
			expectErrors: false,
		},
		{
			name: "multiple data streams, one bad — rejected",
			files: map[string]string{
				"data_stream/good/manifest.yml": "title: Good\ntype: logs\nstreams:\n  - input: logfile\n    title: Good\n    description: Good\n",
				"data_stream/bad/manifest.yml":  "title: Bad\ntype: logs\nstreams:\n  - package: some_package\n    title: Bad\n    description: Bad\n",
			},
			expectErrors: true,
			errorContains: []string{
				"'package:'",
				"source-only",
			},
		},
		// ---------------------------------------------------------------
		// Package manifest policy_templates: happy paths
		// ---------------------------------------------------------------
		{
			name: "policy_template input with type: set — no errors",
			files: map[string]string{
				"manifest.yml": "type: integration\npolicy_templates:\n  - name: logs\n    title: Logs\n    description: Logs\n    inputs:\n      - type: logfile\n        title: Logs\n        description: Logs\n",
			},
			expectErrors: false,
		},
		{
			name: "non-integration package manifest — no errors",
			files: map[string]string{
				"manifest.yml": "type: input\npolicy_templates:\n  - name: logs\n    title: Logs\n    description: Logs\n    inputs:\n      - package: some_package\n        title: Logs\n        description: Logs\n",
			},
			expectErrors: false,
		},
		// ---------------------------------------------------------------
		// Package manifest policy_templates: error cases
		// ---------------------------------------------------------------
		{
			name: "policy_template input has package: — rejected",
			files: map[string]string{
				"manifest.yml": "type: integration\npolicy_templates:\n  - name: events\n    title: Events\n    description: Events\n    inputs:\n      - package: filelog_otel\n        title: Collect events\n        description: Collecting events\n",
			},
			expectErrors: true,
			errorContains: []string{
				"policy_template",
				"events",
				"input[0]",
				"'package:'",
				"source-only",
				"build packages must use 'type:'",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			tempDir := t.TempDir()

			for relPath, content := range testCase.files {
				fullPath := filepath.Join(tempDir, relPath)
				require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
				require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
			}

			fsys := fspath.DirFS(tempDir)
			errs := ValidateStreamInputBundled(fsys)

			if !testCase.expectErrors {
				assert.Nil(t, errs, "expected no errors but got: %v", errs)
				return
			}

			require.NotNil(t, errs, "expected validation errors but got none")
			var sb strings.Builder
			for _, e := range errs {
				sb.WriteString(e.Error())
				sb.WriteString("\n")
			}
			combined := sb.String()
			for _, substr := range testCase.errorContains {
				assert.Contains(t, combined, substr)
			}
		})
	}
}
