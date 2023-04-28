// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/foo/fields/external-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/foo/fields/external-fields.yml",
					dataStream:   "foo",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/foo/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/foo/fields/some_fields.yml",
					dataStream:   "foo",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/hidden_data_stream/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/hidden_data_stream/fields/base-fields.yml",
					dataStream:   "hidden_data_stream",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/hidden_data_stream/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/hidden_data_stream/fields/some_fields.yml",
					dataStream:   "hidden_data_stream",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/ilm_policy/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/ilm_policy/fields/base-fields.yml",
					dataStream:   "ilm_policy",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/ilm_policy/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/ilm_policy/fields/some_fields.yml",
					dataStream:   "ilm_policy",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/k8s_data_stream/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/k8s_data_stream/fields/base-fields.yml",
					dataStream:   "k8s_data_stream",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/k8s_data_stream_no_definitions/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/k8s_data_stream_no_definitions/fields/base-fields.yml",
					dataStream:   "k8s_data_stream_no_definitions",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/pe/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/pe/fields/base-fields.yml",
					dataStream:   "pe",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/pe/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/pe/fields/some_fields.yml",
					dataStream:   "pe",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/skipped_tests/fields/base-fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/skipped_tests/fields/base-fields.yml",
					dataStream:   "skipped_tests",
					packageName:  "good_v2",
					packageType:  "integration",
				},
				fieldFileMetadata{
					filePath:     "data_stream/skipped_tests/fields/some_fields.yml",
					fullFilePath: "../../../../../test/packages/good_v2/data_stream/skipped_tests/fields/some_fields.yml",
					dataStream:   "skipped_tests",
					packageName:  "good_v2",
					packageType:  "integration",
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
					packageName:  "good_input",
					packageType:  "input",
				},
				fieldFileMetadata{
					filePath:     "fields/input.yml",
					fullFilePath: "../../../../../test/packages/good_input/fields/input.yml",
					dataStream:   "",
					packageName:  "good_input",
					packageType:  "input",
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
