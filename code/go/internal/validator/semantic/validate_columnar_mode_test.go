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

func TestValidateLogsDBColumnarSettings(t *testing.T) {
	cases := []struct {
		title          string
		dataStreamType string
		indexMode      string
		streamValue    string
		want           []error
	}{
		{title: "unset on metrics", dataStreamType: "metrics"},
		{title: "unset with index_mode", dataStreamType: "logs", indexMode: "time_series"},
		{title: "set on logs", dataStreamType: "logs", streamValue: "opt_in"},
		{title: "set on metrics", dataStreamType: "metrics", streamValue: "opt_in", want: []error{errInvalidDSTypeColumnar}},
		{title: "set with index_mode", dataStreamType: "logs", indexMode: "time_series", streamValue: "unsupported", want: []error{errInvalidIndexModeLogsDBColumnar}},
		{
			title:          "set on metrics with index_mode",
			dataStreamType: "metrics", indexMode: "time_series", streamValue: "opt_in",
			want: []error{errInvalidDSTypeColumnar, errInvalidIndexModeLogsDBColumnar},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			dataStream := columnarDataStream{name: "ds", manifestPath: "ds/manifest.yml"}
			dataStream.manifest.Type = c.dataStreamType
			dataStream.manifest.Elasticsearch.IndexMode = c.indexMode
			dataStream.manifest.Elasticsearch.LogsDBColumnar = c.streamValue

			errs := validateLogsDBColumnarSettings([]columnarDataStream{dataStream})
			require.Len(t, errs, len(c.want))
			for i, want := range c.want {
				assert.ErrorIs(t, errs[i], want)
				assert.Equal(t, specerrors.UnassignedCode, errs[i].Code())
			}
		})
	}
}

func TestValidateColumnarField(t *testing.T) {
	meta := fieldFileMetadata{filePath: "fields.yml", fullFilePath: "/pkg/fields.yml"}

	cases := []struct {
		title    string
		f        field
		wantErr  error
		wantCode string
	}{
		{
			title: "clean field passes",
			f:     field{Name: "host.name", Type: "keyword"},
		},
		{
			title:    "doc_values false is a hard error",
			f:        field{Name: "event.original", Type: "keyword", DocValues: boolPtr(false)},
			wantErr:  errColumnarDocValuesFalse,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "doc_values true is accepted",
			f:     field{Name: "event.original", Type: "keyword", DocValues: boolPtr(true)},
		},
		{
			title: "doc_values false on a multi-field is accepted",
			f: field{Name: "message", Type: "text", MultiFields: fields{
				{Name: "raw", Type: "keyword", DocValues: boolPtr(false)},
			}},
		},
		{
			title:    "store true is a hard error",
			f:        field{Name: "message", Type: "text", Store: boolPtr(true)},
			wantErr:  errColumnarStoreTrue,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "store true on a multi-field is a hard error",
			f: field{Name: "message", Type: "text", MultiFields: fields{
				{Name: "raw", Type: "keyword", Store: boolPtr(true)},
			}},
			wantErr:  errColumnarStoreTrue,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "store false is accepted",
			f:     field{Name: "message", Type: "text", Store: boolPtr(false)},
		},
		{
			title:    "runtime field is a hard error",
			f:        field{Name: "computed", Type: "keyword", Runtime: runtimeField{enabled: true}},
			wantErr:  errColumnarRuntimeField,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "single level nested is accepted",
			f:     field{Name: "events", Type: "nested"},
		},
		{
			title:    "enabled false is a hard error",
			f:        field{Name: "metadata", Type: "object", Enabled: boolPtr(false)},
			wantErr:  errColumnarEnabledFalse,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "dynamic false on field is a hard error",
			f:        field{Name: "extra", Type: "object", Dynamic: false},
			wantErr:  errColumnarDynamicFalse,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "dynamic string false on field is a hard error",
			f:        field{Name: "extra", Type: "object", Dynamic: "false"},
			wantErr:  errColumnarDynamicFalse,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "dynamic runtime on field is a hard error",
			f:        field{Name: "extra", Type: "object", Dynamic: "runtime"},
			wantErr:  errColumnarDynamicRuntime,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "dynamic strict on field is accepted",
			f:     field{Name: "extra", Type: "object", Dynamic: "strict"},
		},
		{
			title: "enabled true is fine",
			f:     field{Name: "obj", Type: "object", Enabled: boolPtr(true)},
		},
		{
			title:    "copy_to is a hard error",
			f:        field{Name: "source.address", Type: "keyword", CopyTo: "source.ip"},
			wantErr:  errColumnarCopyTo,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "copy_to as slice is a hard error",
			f:        field{Name: "source.address", Type: "keyword", CopyTo: []any{"source.ip", "host.ip"}},
			wantErr:  errColumnarCopyTo,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "keyword with custom normalizer is a hard error",
			f:        field{Name: "status", Type: "keyword", Normalizer: "my_normalizer"},
			wantErr:  errColumnarNormalizer,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title: "keyword with lowercase normalizer is accepted",
			f:     field{Name: "status", Type: "keyword", Normalizer: "lowercase"},
		},
		{
			title: "non-keyword normalizer is ignored",
			f:     field{Name: "text_field", Type: "text", Normalizer: "something"},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			errs := validateColumnarField(meta, c.f)
			if c.wantErr == nil {
				assert.Empty(t, errs)
				return
			}
			require.Len(t, errs, 1)
			assert.ErrorIs(t, errs[0], c.wantErr)
			assert.Equal(t, c.wantCode, errs[0].Code())
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
				`file "fields.yml" is invalid: field "events.inner" is inside nested field "events": ` +
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
				`file "fields.yml" is invalid: field "attributes.items" is inside nested field "attributes": ` +
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
				`file "fields.yml" is invalid: field "a.b" is inside nested field "a": ` +
					`columnar index modes support only a single level of nesting`,
				`file "fields.yml" is invalid: field "a.b.c" is inside nested field "a.b": ` +
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

func TestValidateColumnarIndexSort(t *testing.T) {
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
				`file "ds/manifest.yml" is invalid: index.sort field "missing.field": index sort fields must be defined in the data stream fields`,
			},
		},
		{
			title: "type without doc values",
			sort:  &columnarIndexSort{Field: []string{"message", "labels", "@timestamp"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort field "message": index sort fields must have doc values (type "text")`,
				`file "ds/manifest.yml" is invalid: index.sort field "labels": index sort fields must have doc values (type "object")`,
			},
		},
		{
			title: "doc_values disabled on the field",
			sort:  &columnarIndexSort{Field: []string{"event.original", "@timestamp"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort field "event.original": index sort fields must have doc values (type "keyword")`,
			},
		},
		{
			title: "missing @timestamp",
			sort:  &columnarIndexSort{Field: []string{"host.name"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort does not include @timestamp; most queries filter on a time range, so sorting on it keeps time-range pruning effective (SVR00016)`,
			},
		},
		{
			title: "order length mismatch",
			sort:  &columnarIndexSort{Field: []string{"host.name", "@timestamp"}, Order: []string{"asc"}},
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort.order must have one entry per index.sort.field (2 fields, 1 orders)`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var messages []string
			for _, err := range validateColumnarIndexSort("ds/manifest.yml", c.sort, streamFields) {
				messages = append(messages, err.Error())
			}
			assert.Equal(t, c.want, messages)
		})
	}
}

func TestValidateColumnarIndexSortImplicitFields(t *testing.T) {
	// @timestamp is always mapped by Fleet, so it needs no local definition. host.name is only
	// injected by Elasticsearch for its default sort: a custom sort has to map it like any field.
	streamFields := newColumnarStreamFields()

	errs := validateColumnarIndexSort("ds/manifest.yml", &columnarIndexSort{Field: []string{"@timestamp"}}, streamFields)
	assert.Empty(t, errs)

	errs = validateColumnarIndexSort("ds/manifest.yml", &columnarIndexSort{Field: []string{"host.name", "@timestamp"}}, streamFields)
	require.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], errColumnarSortFieldUndefined)
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

func TestValidateColumnarIndexTemplate(t *testing.T) {
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
			title: "dynamic false is a hard error",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      dynamic: false
`,
			want: []string{
				`file "ds/manifest.yml" is invalid: elasticsearch.index_template.mappings: ` +
					`dynamic: false is not supported in columnar index modes, which have no stored _source to keep unmapped fields in`,
			},
		},
		{
			title: "dynamic runtime is a hard error",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      dynamic: runtime
`,
			want: []string{
				`file "ds/manifest.yml" is invalid: elasticsearch.index_template.mappings: ` +
					`dynamic: runtime is not supported in columnar index modes, which have no stored _source to load runtime fields from`,
			},
		},
		{
			title: "dynamic strict is accepted",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      dynamic: strict
`,
		},
		{
			title: "dynamic templates with doc_values false or store true are hard errors",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    mappings:
      dynamic_templates:
        - _embedded_ecs-event_original_non_indexed_keyword:
            path_match: event.original
            mapping:
              type: keyword
              index: false
              doc_values: false
        - _embedded_ecs-stored:
            path_match: "*.stored"
            mapping:
              type: text
              fields:
                raw:
                  type: keyword
                  store: true
        - _embedded_ecs-port_to_long:
            match: port
            mapping:
              type: long
`,
			want: []string{
				`file "ds/manifest.yml" is invalid: elasticsearch.index_template.mappings.dynamic_templates "_embedded_ecs-event_original_non_indexed_keyword": ` +
					`doc_values: false is not supported in columnar index modes, which rebuild _source from doc values`,
				`file "ds/manifest.yml" is invalid: elasticsearch.index_template.mappings.dynamic_templates "_embedded_ecs-stored": ` +
					`store: true is not supported in columnar index modes, which read every field from doc values`,
			},
		},
		{
			title: "index sort is validated",
			manifest: `
type: logs
elasticsearch:
  logsdb_columnar: opt_in
  index_template:
    settings:
      index:
        sort:
          field: host.name
`,
			want: []string{
				`file "ds/manifest.yml" is invalid: index.sort field "host.name": index sort fields must be defined in the data stream fields`,
				`file "ds/manifest.yml" is invalid: index.sort does not include @timestamp; most queries filter on a time range, so sorting on it keeps time-range pruning effective (SVR00016)`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			dataStream := columnarDataStream{name: "ds", manifestPath: "ds/manifest.yml"}
			require.NoError(t, yaml.Unmarshal([]byte(c.manifest), &dataStream.manifest))

			var messages []string
			for _, err := range validateColumnarIndexTemplate(dataStream, newColumnarStreamFields()) {
				messages = append(messages, err.Error())
			}
			assert.Equal(t, c.want, messages)
		})
	}
}

func TestValidateColumnarModeConstraints(t *testing.T) {
	const logsStreamManifest = "title: Logs\ntype: logs\n"
	const blockerFields = "- name: short_message\n  type: keyword\n  copy_to: message\n"
	const copyToError = `field "short_message": copy_to is not supported in columnar index modes, it prevents synthetic source reconstruction; ` +
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
				`file "<extra>" is invalid: field "attributes.items" is inside nested field "attributes": ` +
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
