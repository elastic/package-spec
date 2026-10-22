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
	"gopkg.in/yaml.v3"

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

func TestIsColumnarReady(t *testing.T) {
	cases := []struct {
		title          string
		dataStreamType string
		indexMode      string
		streamValue    string
		packageValue   string
		want           bool
	}{
		{title: "nothing declared", dataStreamType: "logs"},
		{title: "package opt_in", dataStreamType: "logs", packageValue: "opt_in", want: true},
		{title: "package default", dataStreamType: "logs", packageValue: "default", want: true},
		{title: "stream opt_in", dataStreamType: "logs", streamValue: "opt_in", want: true},
		{title: "stream default", dataStreamType: "logs", streamValue: "default", want: true},
		{title: "stream unsupported overrides package default", dataStreamType: "logs", packageValue: "default", streamValue: "unsupported"},
		{title: "stream opt_in overrides absent package value", dataStreamType: "logs", streamValue: "opt_in", want: true},
		{title: "stream default overrides package opt_in", dataStreamType: "logs", packageValue: "opt_in", streamValue: "default", want: true},
		{title: "package value ignored on metrics", dataStreamType: "metrics", packageValue: "default"},
		{title: "stream value ignored on metrics", dataStreamType: "metrics", streamValue: "opt_in"},
		{title: "fixed index mode wins", dataStreamType: "logs", packageValue: "default", indexMode: "time_series"},
		{title: "unknown value", dataStreamType: "logs", packageValue: "maybe"},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var manifest columnarDataStreamManifest
			manifest.Type = c.dataStreamType
			manifest.Elasticsearch.IndexMode = c.indexMode
			manifest.Elasticsearch.LogsDBColumnar = c.streamValue
			assert.Equal(t, c.want, isColumnarReady(manifest, c.packageValue))
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
			title: "doc_values false is accepted",
			f:     field{Name: "event.original", Type: "keyword", DocValues: boolPtr(false)},
		},
		{
			title:    "runtime field is a hard error",
			f:        field{Name: "computed", Type: "keyword", Runtime: runtimeField{enabled: true}},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "single level nested is accepted",
			f:     field{Name: "events", Type: "nested"},
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
			title: "store true is accepted",
			f:     field{Name: "message", Type: "keyword"},
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

func TestColumnarNestedInNested(t *testing.T) {
	cases := []struct {
		title  string
		nested []columnarNestedField
		want   []string
	}{
		{
			title:  "single level nested is accepted",
			nested: []columnarNestedField{{name: "events", filePath: "fields.yml"}},
		},
		{
			title: "sibling nested fields are accepted",
			nested: []columnarNestedField{
				{name: "events", filePath: "fields.yml"},
				{name: "attributes", filePath: "fields.yml"},
			},
		},
		{
			title: "structural nesting is reported",
			nested: []columnarNestedField{
				{name: "events", filePath: "fields.yml"},
				{name: "events.inner", filePath: "fields.yml"},
			},
			want: []string{
				`file "fields.yml" is invalid: field "events.inner" is a nested field inside another nested field ("events"); ` +
					`columnar index modes support only a single level of nesting`,
			},
		},
		{
			title: "dotted siblings in different files are reported against the child file",
			nested: []columnarNestedField{
				{name: "attributes", filePath: "base-fields.yml"},
				{name: "attributes.items", filePath: "fields.yml"},
			},
			want: []string{
				`file "fields.yml" is invalid: field "attributes.items" is a nested field inside another nested field ("attributes"); ` +
					`columnar index modes support only a single level of nesting`,
			},
		},
		{
			title: "the closest nested ancestor is reported",
			nested: []columnarNestedField{
				{name: "a", filePath: "fields.yml"},
				{name: "a.b", filePath: "fields.yml"},
				{name: "a.b.c", filePath: "fields.yml"},
			},
			want: []string{
				`file "fields.yml" is invalid: field "a.b" is a nested field inside another nested field ("a"); ` +
					`columnar index modes support only a single level of nesting`,
				`file "fields.yml" is invalid: field "a.b.c" is a nested field inside another nested field ("a.b"); ` +
					`columnar index modes support only a single level of nesting`,
			},
		},
		{
			title: "a name sharing a prefix without a dot is not nested",
			nested: []columnarNestedField{
				{name: "events", filePath: "fields.yml"},
				{name: "events_extra", filePath: "fields.yml"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			collected := newColumnarStreamFields()
			collected.nested = c.nested

			var messages []string
			for _, err := range collected.nestedInNestedErrors() {
				messages = append(messages, err.Error())
			}
			assert.Equal(t, c.want, messages)
		})
	}
}

func TestCheckColumnarIndexSort(t *testing.T) {
	streamFields := newColumnarStreamFields()
	streamFields.defined = map[string]columnarFieldInfo{
		"@timestamp":      {fieldType: "date"},
		"host.name":       {fieldType: "keyword"},
		"organization.id": {}, // declared with external: ecs, no local type
		"message":         {fieldType: "text"},
		"labels":          {fieldType: "object"},
		"event.original":  {fieldType: "keyword", docValues: boolPtr(false)},
	}

	cases := []struct {
		title string
		sort  *columnarIndexSort
		want  []string
	}{
		{
			title: "no sort settings",
		},
		{
			title: "valid sort",
			sort:  &columnarIndexSort{Field: []string{"host.name", "@timestamp"}, Order: []string{"asc", "desc"}},
		},
		{
			title: "external ecs fields count as defined",
			sort:  &columnarIndexSort{Field: []string{"organization.id", "@timestamp"}},
		},
		{
			title: "undefined field",
			sort:  &columnarIndexSort{Field: []string{"missing.field", "@timestamp"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort field "missing.field" is not defined in the data stream fields`,
			},
		},
		{
			title: "type without doc values",
			sort:  &columnarIndexSort{Field: []string{"message", "labels", "@timestamp"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort field "message" has no doc values (type "text"), index sorting requires doc values`,
				`file "ds/manifest.yml" is invalid: index.sort field "labels" has no doc values (type "object"), index sorting requires doc values`,
			},
		},
		{
			title: "doc_values disabled on the field",
			sort:  &columnarIndexSort{Field: []string{"event.original", "@timestamp"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort field "event.original" has no doc values (type "keyword"), index sorting requires doc values`,
			},
		},
		{
			title: "missing @timestamp",
			sort:  &columnarIndexSort{Field: []string{"host.name"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort must include @timestamp`,
			},
		},
		{
			title: "order length mismatch",
			sort:  &columnarIndexSort{Field: []string{"host.name", "@timestamp"}, Order: []string{"asc"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort.order has 1 entries but index.sort.field has 2, they must have the same length`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var messages []string
			for _, err := range checkColumnarIndexSort("ds/manifest.yml", c.sort, streamFields) {
				messages = append(messages, err.Error())
			}
			assert.Equal(t, c.want, messages)
		})
	}
}

func TestColumnarStringListUnmarshal(t *testing.T) {
	cases := []struct {
		title string
		yaml  string
		want  columnarStringList
	}{
		{title: "list", yaml: "field:\n  - a\n  - b\n", want: columnarStringList{"a", "b"}},
		{title: "single string", yaml: "field: a\n", want: columnarStringList{"a"}},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var sort columnarIndexSort
			require.NoError(t, yaml.Unmarshal([]byte(c.yaml), &sort))
			assert.Equal(t, c.want, sort.Field)
		})
	}
}

func TestCheckColumnarManifest(t *testing.T) {
	cases := []struct {
		title    string
		manifest string
		want     []string
	}{
		{
			title:    "no index template",
			manifest: "type: logs\nelasticsearch:\n  logsdb_columnar: opt_in\n",
		},
		{
			title: "copy_to in nested properties",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      properties:
        event:
          properties:
            original:
              type: keyword
              copy_to: message
`,
			want: []string{
				"elasticsearch.index_template.mappings.properties.event.properties.original has copy_to set, " +
					"which prevents synthetic source reconstruction in columnar index mode; " +
					"use an ingest pipeline to copy the value instead",
			},
		},
		{
			title: "copy_to inside multi-fields",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      properties:
        host.name:
          type: keyword
          fields:
            text:
              type: text
              copy_to: message
`,
			want: []string{
				"elasticsearch.index_template.mappings.properties.host.name.fields.text has copy_to set, " +
					"which prevents synthetic source reconstruction in columnar index mode; " +
					"use an ingest pipeline to copy the value instead",
			},
		},
		{
			title: "dynamic false is a filterable warning",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      dynamic: false
`,
			want: []string{
				"elasticsearch.index_template.mappings.dynamic is set to false; " +
					"columnar index mode has no stored _source, so unmapped fields are permanently lost when dynamic is false (SVR00014)",
			},
		},
		{
			title: "doc_values and store are no longer reported",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      properties:
        message:
          type: keyword
          doc_values: false
          store: true
`,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			tempDir := t.TempDir()
			dsDir := filepath.Join(tempDir, "data_stream", "logs")
			require.NoError(t, os.MkdirAll(dsDir, 0755))
			require.NoError(t, os.WriteFile(filepath.Join(dsDir, "manifest.yml"), []byte(c.manifest), 0644))

			fsys := fspath.DirFS(tempDir)
			errs := checkColumnarManifest(fsys, "data_stream/logs/manifest.yml", newColumnarStreamFields())

			var messages []string
			prefix := `file "` + fsys.Path("data_stream/logs/manifest.yml") + `" is invalid: `
			for _, err := range errs {
				messages = append(messages, strings.TrimPrefix(err.Error(), prefix))
			}
			assert.Equal(t, c.want, messages)
		})
	}
}

func TestValidateColumnarModeConstraints(t *testing.T) {
	const logsStreamManifest = "title: Logs\ntype: logs\n"
	const blockerFields = "- name: short_message\n  type: keyword\n  copy_to: message\n"
	const copyToError = `field "short_message" has copy_to set, which prevents synthetic source reconstruction in columnar index mode; ` +
		`use an ingest pipeline to copy the value instead`

	cases := []struct {
		title           string
		packageManifest string
		dataStreams     map[string]map[string]string
		want            []string
	}{
		{
			title:           "package without readiness is skipped",
			packageManifest: "type: integration\n",
			dataStreams: map[string]map[string]string{
				"logs": {"manifest.yml": logsStreamManifest, "fields/base-fields.yml": blockerFields},
			},
		},
		{
			title:           "package opt_in enables the checks",
			packageManifest: "type: integration\nelasticsearch:\n  logsdb_columnar: opt_in\n",
			dataStreams: map[string]map[string]string{
				"logs": {"manifest.yml": logsStreamManifest, "fields/base-fields.yml": blockerFields},
			},
			want: []string{`file "<fields>" is invalid: ` + copyToError},
		},
		{
			title:           "data stream unsupported disables the checks",
			packageManifest: "type: integration\nelasticsearch:\n  logsdb_columnar: default\n",
			dataStreams: map[string]map[string]string{
				"logs": {
					"manifest.yml":           logsStreamManifest + "elasticsearch:\n  logsdb_columnar: unsupported\n",
					"fields/base-fields.yml": blockerFields,
				},
			},
		},
		{
			title:           "data stream opt_in enables the checks without a package value",
			packageManifest: "type: integration\n",
			dataStreams: map[string]map[string]string{
				"logs": {
					"manifest.yml":           logsStreamManifest + "elasticsearch:\n  logsdb_columnar: opt_in\n",
					"fields/base-fields.yml": blockerFields,
				},
			},
			want: []string{`file "<fields>" is invalid: ` + copyToError},
		},
		{
			title:           "logsdb_columnar on a non logs data stream is an error",
			packageManifest: "type: integration\n",
			dataStreams: map[string]map[string]string{
				"logs": {
					"manifest.yml": "title: Metrics\ntype: metrics\nelasticsearch:\n  logsdb_columnar: opt_in\n",
				},
			},
			want: []string{
				`file "<manifest>" is invalid: elasticsearch.logsdb_columnar is only supported on logs data streams (data stream type is "metrics")`,
			},
		},
		{
			title:           "index_mode and logsdb_columnar cannot both be set",
			packageManifest: "type: integration\n",
			dataStreams: map[string]map[string]string{
				"logs": {
					"manifest.yml": logsStreamManifest + "elasticsearch:\n  index_mode: time_series\n  logsdb_columnar: opt_in\n",
				},
			},
			want: []string{
				`file "<manifest>" is invalid: elasticsearch.index_mode and elasticsearch.logsdb_columnar cannot both be set; ` +
					`logsdb_columnar only applies when index_mode is unset`,
			},
		},
		{
			title:           "nested in nested is reported across fields files",
			packageManifest: "type: integration\nelasticsearch:\n  logsdb_columnar: opt_in\n",
			dataStreams: map[string]map[string]string{
				"logs": {
					"manifest.yml":            logsStreamManifest,
					"fields/base-fields.yml":  "- name: attributes\n  type: nested\n",
					"fields/extra-fields.yml": "- name: attributes.items\n  type: nested\n",
				},
			},
			want: []string{
				`file "<extra>" is invalid: field "attributes.items" is a nested field inside another nested field ("attributes"); ` +
					`columnar index modes support only a single level of nesting`,
			},
		},
		{
			title:           "package level value does not reach metrics data streams",
			packageManifest: "type: integration\nelasticsearch:\n  logsdb_columnar: opt_in\n",
			dataStreams: map[string]map[string]string{
				"logs": {
					"manifest.yml":           "title: Metrics\ntype: metrics\n",
					"fields/base-fields.yml": blockerFields,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			tempDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(tempDir, "manifest.yml"), []byte(c.packageManifest), 0644))
			for dataStream, files := range c.dataStreams {
				for name, contents := range files {
					target := filepath.Join(append([]string{tempDir, "data_stream", dataStream}, strings.Split(name, "/")...)...)
					require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))
					require.NoError(t, os.WriteFile(target, []byte(contents), 0644))
				}
			}

			fsys := fspath.DirFS(tempDir)
			replacer := strings.NewReplacer(
				fsys.Path("data_stream/logs/fields/base-fields.yml"), "<fields>",
				fsys.Path("data_stream/logs/fields/extra-fields.yml"), "<extra>",
				fsys.Path("data_stream/logs/manifest.yml"), "<manifest>",
			)

			var messages []string
			for _, err := range ValidateColumnarModeConstraints(fsys) {
				messages = append(messages, replacer.Replace(err.Error()))
			}
			assert.Equal(t, c.want, messages)
		})
	}
}
