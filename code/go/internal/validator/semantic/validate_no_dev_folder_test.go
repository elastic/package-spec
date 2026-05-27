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

func TestValidateNoDevFolder(t *testing.T) {
	tests := []struct {
		name          string
		dirs          []string // directories to create relative to the temp root
		expectErrors  bool
		errorContains []string
	}{
		{
			name:         "no _dev directory",
			dirs:         []string{"data_stream/foo/fields"},
			expectErrors: false,
		},
		{
			name:          "_dev at package root",
			dirs:          []string{"_dev/build"},
			expectErrors:  true,
			errorContains: []string{"_dev", "_dev directory is not allowed in built packages"},
		},
		{
			name:          "_dev inside data_stream",
			dirs:          []string{"data_stream/foo/_dev/test"},
			expectErrors:  true,
			errorContains: []string{"_dev", "_dev directory is not allowed in built packages"},
		},
		{
			name: "multiple _dev directories",
			dirs: []string{
				"_dev/build",
				"data_stream/foo/_dev/test",
			},
			expectErrors:  true,
			errorContains: []string{"_dev directory is not allowed in built packages"},
		},
		{
			name:         "directory named _devtools is not rejected",
			dirs:         []string{"_devtools"},
			expectErrors: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			for _, d := range tc.dirs {
				require.NoError(t, os.MkdirAll(filepath.Join(tempDir, d), 0o755))
			}

			fsys := fspath.DirFS(tempDir)
			errs := ValidateNoDevFolder(fsys)

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
