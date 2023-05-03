// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
)

func TestListFieldsFiles(t *testing.T) {
	cases := []struct {
		pkgName  string
		expected []fieldFileMetadata
	}{
		{
			pkgName: "good_v2",
			expected: []fieldFileMetadata{
				fieldFileMetadata{
					filePath:     "data_stream/foo/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/foo/fields/base-fields.yml",
					dataStream:   "foo",
				},
				fieldFileMetadata{
					filePath:     "data_stream/foo/fields/external-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/foo/fields/external-fields.yml",
					dataStream:   "foo",
				},
				fieldFileMetadata{
					filePath:     "data_stream/foo/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/foo/fields/some_fields.yml",
					dataStream:   "foo",
				},
				fieldFileMetadata{
					filePath:     "data_stream/hidden_data_stream/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/hidden_data_stream/fields/base-fields.yml",
					dataStream:   "hidden_data_stream",
				},
				fieldFileMetadata{
					filePath:     "data_stream/hidden_data_stream/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/hidden_data_stream/fields/some_fields.yml",
					dataStream:   "hidden_data_stream",
				},
				fieldFileMetadata{
					filePath:     "data_stream/ilm_policy/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/ilm_policy/fields/base-fields.yml",
					dataStream:   "ilm_policy",
				},
				fieldFileMetadata{
					filePath:     "data_stream/ilm_policy/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/ilm_policy/fields/some_fields.yml",
					dataStream:   "ilm_policy",
				},
				fieldFileMetadata{
					filePath:     "data_stream/k8s_data_stream/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/k8s_data_stream/fields/base-fields.yml",
					dataStream:   "k8s_data_stream",
				},
				fieldFileMetadata{
					filePath:     "data_stream/k8s_data_stream_no_definitions/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/k8s_data_stream_no_definitions/fields/base-fields.yml",
					dataStream:   "k8s_data_stream_no_definitions",
				},
				fieldFileMetadata{
					filePath:     "data_stream/pe/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/pe/fields/base-fields.yml",
					dataStream:   "pe",
				},
				fieldFileMetadata{
					filePath:     "data_stream/pe/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/pe/fields/some_fields.yml",
					dataStream:   "pe",
				},
				fieldFileMetadata{
					filePath:     "data_stream/skipped_tests/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/skipped_tests/fields/base-fields.yml",
					dataStream:   "skipped_tests",
				},
				fieldFileMetadata{
					filePath:     "data_stream/skipped_tests/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/skipped_tests/fields/some_fields.yml",
					dataStream:   "skipped_tests",
				},
			},
		},
		{
			pkgName: "good_input",
			expected: []fieldFileMetadata{
				fieldFileMetadata{
					filePath:     "fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_input/fields/base-fields.yml",
					dataStream:   "",
				},
				fieldFileMetadata{
					filePath:     "fields/input.yml",
					fullFilePath: "../../../../../test/packages/good_input/fields/input.yml",
					dataStream:   "",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.pkgName, func(t *testing.T) {
			pkgRootPath := path.Join("..", "..", "..", "..", "..", "test", "packages", c.pkgName)

			fsys := fspath.DirFS(pkgRootPath)
			fieldFilesMetadata, err := listFieldsFiles(fsys)
			require.NoError(t, err)

			require.Len(t, fieldFilesMetadata, len(c.expected))

			for i, metadata := range fieldFilesMetadata {
				assert.Equal(t, c.expected[i], metadata)
			}
		})
	}
}

func TestRuntimeUnmarshal(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		testRuntimeUnmarshalFormat(t, json.Unmarshal)
	})
	t.Run("yaml", func(t *testing.T) {
		testRuntimeUnmarshalFormat(t, yaml.Unmarshal)
	})
}

func testRuntimeUnmarshalFormat(t *testing.T, unmarshaler func([]byte, interface{}) error) {
	cases := []struct {
		json     string
		expected runtimeField
		valid    bool
	}{
		{"true", runtimeField{enabled: true, script: ""}, true},
		{"false", runtimeField{enabled: false, script: ""}, true},
		{"42", runtimeField{enabled: true, script: "42"}, true},
		{"\"doc['message'].value().doSomething()\"", runtimeField{enabled: true, script: "doc['message'].value().doSomething()"}, true},
	}

	for _, c := range cases {
		t.Run(c.json, func(t *testing.T) {
			var found runtimeField
			err := unmarshaler([]byte(c.json), &found)
			if c.valid {
				require.NoError(t, err)
				assert.Equal(t, c.expected, found)
			} else {
				require.Error(t, err)
			}
		})
	}
}
