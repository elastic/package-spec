// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFS implements fspath.FS for testing
type mockFS struct {
	fstest.MapFS
	basePath string
}

func newMockFS() *mockFS {
	return &mockFS{
		MapFS:    make(fstest.MapFS),
		basePath: "/test/package",
	}
}

func (m *mockFS) Path(names ...string) string {
	return filepath.Join(append([]string{m.basePath}, names...)...)
}

func (m *mockFS) WithDataStream(name, packageType string) *mockFS {
	// Create data stream directory structure
	manifestPath := filepath.Join("data_stream", name, "manifest.yml")
	m.MapFS[manifestPath] = &fstest.MapFile{
		Data:    []byte(""),
		Mode:    0644,
		ModTime: time.Now(),
	}

	// Create package manifest if it's an integration
	if packageType == "integration" {
		m.MapFS["manifest.yml"] = &fstest.MapFile{
			Data:    []byte("format_version: 3.0.0\nname: " + name + "\ntype: integration\n"),
			Mode:    0644,
			ModTime: time.Now(),
		}
	}
	return m
}

func (m *mockFS) WithFile(path, content string) *mockFS {
	// Normalize path separators
	path = filepath.ToSlash(path)
	m.MapFS[path] = &fstest.MapFile{
		Data:    []byte(strings.TrimPrefix(content, "\n")),
		Mode:    0644,
		ModTime: time.Now(),
	}
	return m
}

func TestValidateStreamTemplates(t *testing.T) {
	tests := []struct {
		name    string
		fsys    fspath.FS
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_data_stream_with_templates",
			fsys: newMockFS().
				WithDataStream("test", "integration").
				WithFile("data_stream/test/manifest.yml", `
streams:
  - input: udp
    template_path: udp.yml.hbs
    title: Test UDP
    description: Test UDP stream
`).
				WithFile("data_stream/test/agent/stream/udp.yml.hbs", "# UDP template"),
			wantErr: false,
		},
		{
			name: "missing_template_file",
			fsys: newMockFS().
				WithDataStream("test", "integration").
				WithFile("data_stream/test/manifest.yml", `
streams:
  - input: udp
    template_path: missing.yml.hbs
    title: Test UDP
    description: Test UDP stream
`),
			wantErr: true,
			errMsg:  "references template_path \"missing.yml.hbs\" but file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateStreamTemplates(tt.fsys)

			if tt.wantErr {
				require.NotEmpty(t, errs, "expected validation errors")
				assert.Contains(t, errs.Error(), tt.errMsg)
			} else {
				assert.Empty(t, errs, "expected no validation errors")
			}
		})
	}
}
