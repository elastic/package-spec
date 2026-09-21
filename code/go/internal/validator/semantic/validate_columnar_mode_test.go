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

func TestDynamicIsFalse(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "bool false", value: false, want: true},
		{name: "bool true", value: true, want: false},
		{name: "string false", value: "false", want: true},
		{name: "string true", value: "true", want: false},
		{name: "string strict", value: "strict", want: false},
		{name: "nil", value: nil, want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, dynamicIsFalse(c.value))
		})
	}
}

func TestIsColumnarEnabled(t *testing.T) {
	cases := []struct {
		title    string
		manifest string
		want     bool
	}{
		{title: "logsdb_columnar", manifest: "elasticsearch:\n  index_mode: logsdb_columnar\n", want: true},
		{title: "columnar", manifest: "elasticsearch:\n  index_mode: columnar\n", want: true},
		{title: "logsdb", manifest: "elasticsearch:\n  index_mode: logsdb\n", want: false},
		{title: "time_series", manifest: "elasticsearch:\n  index_mode: time_series\n", want: false},
		{title: "empty manifest", manifest: "{}\n", want: false},
		{
			title:    "columnar supported without index mode",
			manifest: "elasticsearch:\n  columnar:\n    supported: true\n",
			want:     true,
		},
		{
			title:    "columnar supported false without index mode",
			manifest: "elasticsearch:\n  columnar:\n    supported: false\n",
			want:     false,
		},
		{
			title:    "columnar supported false with columnar index mode",
			manifest: "elasticsearch:\n  index_mode: columnar\n  columnar:\n    supported: false\n",
			want:     true,
		},
		{
			title:    "columnar supported with unrelated index mode",
			manifest: "elasticsearch:\n  index_mode: time_series\n  columnar:\n    supported: true\n",
			want:     true,
		},
		{
			title:    "empty columnar object",
			manifest: "elasticsearch:\n  columnar: {}\n",
			want:     false,
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			tempDir := t.TempDir()
			dsDir := filepath.Join(tempDir, "data_stream", "logs")
			require.NoError(t, os.MkdirAll(dsDir, 0755))

			require.NoError(t, os.WriteFile(filepath.Join(dsDir, "manifest.yml"), []byte(c.manifest), 0644))

			got, err := isColumnarEnabled(fspath.DirFS(tempDir), "logs")
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestCheckColumnarField(t *testing.T) {
	meta := fieldFileMetadata{filePath: "fields.yml", fullFilePath: "/pkg/fields.yml"}

	cases := []struct {
		title     string
		f         field
		wantErrs  bool
		wantCode  string
		wantCount int
	}{
		{
			title: "clean field passes",
			f:     field{Name: "host.name", Type: "keyword"},
		},
		{
			title:    "doc_values false is a hard error",
			f:        field{Name: "message", Type: "keyword", DocValues: boolPtr(false)},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "runtime field is a hard error",
			f:        field{Name: "computed", Type: "keyword", Runtime: runtimeField{enabled: true}},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "nested type is a filterable warning",
			f:        field{Name: "events", Type: "nested"},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarNestedField,
		},
		{
			title:    "enabled false is a filterable warning",
			f:        field{Name: "metadata", Type: "object", Enabled: boolPtr(false)},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarEnabledFalse,
		},
		{
			title:    "dynamic false on field is a filterable warning",
			f:        field{Name: "extra", Type: "object", Dynamic: false},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarDynamicFalse,
		},
		{
			title:    "dynamic string false on field is a filterable warning",
			f:        field{Name: "extra", Type: "object", Dynamic: "false"},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarDynamicFalse,
		},
		{
			title: "doc_values true is fine",
			f:     field{Name: "host.id", Type: "keyword", DocValues: boolPtr(true)},
		},
		{
			title: "enabled true is fine",
			f:     field{Name: "obj", Type: "object", Enabled: boolPtr(true)},
		},
		{
			title:    "copy_to is a hard error",
			f:        field{Name: "source.address", Type: "keyword", CopyTo: "source.ip"},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "copy_to as slice is a hard error",
			f:        field{Name: "source.address", Type: "keyword", CopyTo: []any{"source.ip", "host.ip"}},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "keyword with custom normalizer is a hard error",
			f:        field{Name: "status", Type: "keyword", Normalizer: "my_normalizer"},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "keyword with lowercase normalizer is accepted",
			f:        field{Name: "status", Type: "keyword", Normalizer: "lowercase"},
			wantErrs: false,
		},
		{
			title:    "completion type is a hard error",
			f:        field{Name: "suggest", Type: "completion"},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "search_as_you_type is a hard error",
			f:        field{Name: "title", Type: "search_as_you_type"},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "percolator is a hard error",
			f:        field{Name: "query", Type: "percolator"},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "non-keyword normalizer is ignored",
			f:        field{Name: "text_field", Type: "text", Normalizer: "something"},
			wantErrs: false,
		},
		{
			title: "columnar doc_values override fixes doc_values false",
			f: field{
				Name:      "event.original",
				Type:      "keyword",
				DocValues: boolPtr(false),
				Columnar:  &columnarOverrides{DocValues: boolPtr(true)},
			},
			wantErrs: false,
		},
		{
			title: "columnar doc_values false is a hard error",
			f: field{
				Name:     "event.original",
				Type:     "keyword",
				Columnar: &columnarOverrides{DocValues: boolPtr(false)},
			},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "columnar doc_values false reports once when base is also false",
			f: field{
				Name:      "event.original",
				Type:      "keyword",
				DocValues: boolPtr(false),
				Columnar:  &columnarOverrides{DocValues: boolPtr(false)},
			},
			wantErrs:  true,
			wantCode:  specerrors.UnassignedCode,
			wantCount: 1,
		},
		{
			title: "columnar index override is accepted",
			f: field{
				Name:     "host.name",
				Type:     "keyword",
				Columnar: &columnarOverrides{Index: boolPtr(true)},
			},
			wantErrs: false,
		},
		{
			title: "columnar index false is accepted",
			f: field{
				Name:     "host.name",
				Type:     "keyword",
				Columnar: &columnarOverrides{Index: boolPtr(false)},
			},
			wantErrs: false,
		},
		{
			title: "columnar doc_values override does not mask other errors",
			f: field{
				Name:      "source.address",
				Type:      "keyword",
				DocValues: boolPtr(false),
				CopyTo:    "source.ip",
				Columnar:  &columnarOverrides{DocValues: boolPtr(true)},
			},
			wantErrs:  true,
			wantCode:  specerrors.UnassignedCode,
			wantCount: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			errs := checkColumnarField(meta, c.f)
			if !c.wantErrs {
				assert.Empty(t, errs)
				return
			}
			assert.NotEmpty(t, errs)
			if c.wantCode != "" {
				assert.Equal(t, c.wantCode, errs[0].Code())
			}
			if c.wantCount != 0 {
				assert.Len(t, errs, c.wantCount)
			}
		})
	}
}
