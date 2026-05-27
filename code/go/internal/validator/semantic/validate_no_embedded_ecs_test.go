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

func TestValidateNoEmbeddedEcsInDynamicTemplates(t *testing.T) {
	tests := []struct {
		name            string
		manifestYAML    string
		skipWriteStream bool // when true, data_stream/ is not created (simulates input/content packages)
		expectErrors    bool
		errorContains   []string
	}{
		{
			name:            "no data_stream directory",
			skipWriteStream: true,
			expectErrors:    false,
		},
		{
			name: "no dynamic_templates",
			manifestYAML: `
title: Test stream
type: logs
streams:
  - input: logfile
    title: Sample
    description: Sample
`,
			expectErrors: false,
		},
		{
			name: "dynamic_templates with regular names",
			manifestYAML: `
title: Test stream
type: logs
streams:
  - input: logfile
    title: Sample
    description: Sample
elasticsearch:
  index_template:
    mappings:
      dynamic_templates:
        - my_ip_template:
            mapping:
              type: ip
            match: ip
        - my_date_template:
            mapping:
              type: date
            match: timestamp
`,
			expectErrors: false,
		},
		{
			name: "dynamic_templates with _embedded_ecs key",
			manifestYAML: `
title: Test stream
type: logs
streams:
  - input: logfile
    title: Sample
    description: Sample
elasticsearch:
  index_template:
    mappings:
      dynamic_templates:
        - _embedded_ecs-ip_to_ip:
            mapping:
              type: ip
            match: ip
`,
			expectErrors:  true,
			errorContains: []string{"_embedded_ecs-ip_to_ip", "_embedded_ecs"},
		},
		{
			name: "dynamic_templates mixed: regular and _embedded_ecs",
			manifestYAML: `
title: Test stream
type: logs
streams:
  - input: logfile
    title: Sample
    description: Sample
elasticsearch:
  index_template:
    mappings:
      dynamic_templates:
        - my_ip_template:
            mapping:
              type: ip
            match: ip
        - _embedded_ecs-date_to_date:
            mapping:
              type: date
            match: timestamp
`,
			expectErrors:  true,
			errorContains: []string{"_embedded_ecs-date_to_date"},
		},
		{
			name: "multiple _embedded_ecs entries",
			manifestYAML: `
title: Test stream
type: logs
streams:
  - input: logfile
    title: Sample
    description: Sample
elasticsearch:
  index_template:
    mappings:
      dynamic_templates:
        - _embedded_ecs-ip_to_ip:
            mapping:
              type: ip
            match: ip
        - _embedded_ecs-port_to_long:
            mapping:
              type: long
            match: port
`,
			expectErrors:  true,
			errorContains: []string{"_embedded_ecs-ip_to_ip", "_embedded_ecs-port_to_long"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			if !tc.skipWriteStream {
				// Create data_stream/test_stream/manifest.yml
				dsManifestDir := filepath.Join(tempDir, "data_stream", "test_stream")
				require.NoError(t, os.MkdirAll(dsManifestDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dsManifestDir, "manifest.yml"), []byte(tc.manifestYAML), 0644))
			}

			fsys := fspath.DirFS(tempDir)
			errs := ValidateNoEmbeddedEcsInDynamicTemplates(fsys)

			if !tc.expectErrors {
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
			for _, substr := range tc.errorContains {
				assert.Contains(t, combined, substr)
			}
		})
	}
}
