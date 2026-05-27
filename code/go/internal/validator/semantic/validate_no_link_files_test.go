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

func TestValidateNoLinkFiles(t *testing.T) {
	tests := []struct {
		name          string
		files         []string // files to create relative to the temp root
		expectErrors  bool
		errorContains []string
	}{
		{
			name:         "no .link files",
			files:        []string{"data_stream/foo/fields/base-fields.yml"},
			expectErrors: false,
		},
		{
			name:          ".link file in fields directory",
			files:         []string{"data_stream/foo/fields/some-fields.yml.link"},
			expectErrors:  true,
			errorContains: []string{"some-fields.yml.link", ".link files are not allowed in built packages"},
		},
		{
			name:          ".link file at package root",
			files:         []string{"elasticsearch/ingest_pipeline/default.yml.link"},
			expectErrors:  true,
			errorContains: []string{"default.yml.link", ".link files are not allowed in built packages"},
		},
		{
			name: "multiple .link files",
			files: []string{
				"data_stream/foo/fields/some-fields.yml.link",
				"data_stream/foo/agent/stream/stream.yml.hbs.link",
			},
			expectErrors:  true,
			errorContains: []string{".link files are not allowed in built packages"},
		},
		{
			name:         "file with .link in the middle of the name is not rejected",
			files:        []string{"data_stream/foo/fields/base-fields.yml"},
			expectErrors: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			for _, f := range tc.files {
				fullPath := filepath.Join(tempDir, f)
				require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
				require.NoError(t, os.WriteFile(fullPath, []byte(""), 0o644))
			}

			fsys := fspath.DirFS(tempDir)
			errs := ValidateNoLinkFiles(fsys)

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
