// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jsonschema

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSchema_RetrieveOneFile(t *testing.T) {
	cases := []struct {
		title                  string
		version                string
		pkgType                string
		filePath               string
		expectedError          bool
		expectedJSONSchemaPath string
	}{
		{
			title:                  "input manifest version 1.0.0",
			version:                "1.0.0",
			pkgType:                "input",
			filePath:               "manifest.yml",
			expectedError:          false,
			expectedJSONSchemaPath: "testdata/input.manifest.version.1.0.0.yml",
		},
		{
			title:                  "integration manifest version 1.0.0",
			version:                "1.0.0",
			pkgType:                "integration",
			filePath:               "manifest.yml",
			expectedError:          false,
			expectedJSONSchemaPath: "testdata/integration.manifest.version.1.0.0.yml",
		},
		{
			title:                  "input manifest version 2.1.0",
			version:                "2.1.0",
			pkgType:                "input",
			filePath:               "manifest.yml",
			expectedError:          false,
			expectedJSONSchemaPath: "testdata/input.manifest.version.2.1.0.yml",
		},
		{
			title:                  "integration manifest version 2.1.0",
			version:                "2.1.0",
			pkgType:                "integration",
			filePath:               "manifest.yml",
			expectedError:          false,
			expectedJSONSchemaPath: "testdata/integration.manifest.version.2.1.0.yml",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			rendered, err := JSONSchema(c.filePath, c.version, c.pkgType)
			if c.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			contents, err := os.ReadFile(c.expectedJSONSchemaPath)
			require.NoError(t, err)
			assert.Equal(t, string(contents), string(rendered))
		})
	}
}
