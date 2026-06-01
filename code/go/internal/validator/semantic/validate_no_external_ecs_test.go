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

func TestValidateNoExternalEcs(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string // path → content
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "no external fields — no errors",
			files: map[string]string{
				"data_stream/foo/fields/fields.yml": "- name: message\n  type: keyword\n",
			},
			expectErrors: false,
		},
		{
			name: "materialized ECS field (no external key) — no errors",
			files: map[string]string{
				"data_stream/foo/fields/base-fields.yml": "- name: data_stream.type\n  type: constant_keyword\n  description: Data stream type.\n",
			},
			expectErrors: false,
		},
		{
			name: "field with external: ecs — rejected",
			files: map[string]string{
				"data_stream/foo/fields/ecs.yml": "- name: host.name\n  external: ecs\n",
			},
			expectErrors:  true,
			errorContains: []string{"host.name", "external: ecs", "ECS fields must be materialized"},
		},
		{
			name: "multiple fields with external: ecs — all rejected",
			files: map[string]string{
				"data_stream/foo/fields/ecs.yml": "- name: host.name\n  external: ecs\n- name: agent.version\n  external: ecs\n",
			},
			expectErrors:  true,
			errorContains: []string{"external: ecs", "ECS fields must be materialized"},
		},
		{
			name: "field with non-ecs external — not rejected by this rule",
			files: map[string]string{
				"data_stream/foo/fields/custom.yml": "- name: myfield\n  external: custom_dep\n",
			},
			expectErrors: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			for relPath, content := range tc.files {
				fullPath := filepath.Join(tempDir, relPath)
				require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
				require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
			}

			fsys := fspath.DirFS(tempDir)
			errs := ValidateNoExternalEcs(fsys)

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
