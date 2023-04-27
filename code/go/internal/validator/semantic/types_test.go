// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageAndDataStreamFromFieldsPath(t *testing.T) {
	cases := []struct {
		pkgRoot    string
		fieldsFile string
		expected   string
		fail       bool
	}{
		{
			pkgRoot:    "package",
			fieldsFile: "package/data_stream/foo/fields/some-fields.yml",
			expected:   "foo",
		},
		{
			pkgRoot:    "package/",
			fieldsFile: "package/data_stream/foo/fields/some-fields.yml",
			expected:   "foo",
		},
		{
			pkgRoot:    "/package/",
			fieldsFile: "/package/data_stream/foo/fields/some-fields.yml",
			expected:   "foo",
		},
		{
			pkgRoot:    "/package/",
			fieldsFile: "/package/fields/some-fields.yml",
			expected:   "package",
		},
		{
			pkgRoot:    "/package/",
			fieldsFile: "/package/fields.yml",
			fail:       true,
		},
	}

	for _, c := range cases {
		t.Run(c.pkgRoot+"_"+c.fieldsFile, func(t *testing.T) {
			dataStream, err := packageAndDataStreamFromFieldsPath(c.pkgRoot, c.fieldsFile)
			if c.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, c.expected, dataStream)
			}
		})
	}
}
