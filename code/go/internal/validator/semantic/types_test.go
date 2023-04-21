// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDataStreamFromFieldsPath(t *testing.T) {
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
			fail:       true,
		},
	}

	for _, c := range cases {
		t.Run(c.pkgRoot+"_"+c.fieldsFile, func(t *testing.T) {
			dataStream, err := dataStreamFromFieldsPath(c.pkgRoot, c.fieldsFile)
			if c.fail {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, c.expected, dataStream)
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
		expected runtime
		valid    bool
	}{
		{"true", runtime{enabled: true, script: ""}, true},
		{"false", runtime{enabled: false, script: ""}, true},
		{"42", runtime{enabled: true, script: "42"}, true},
		{"\"doc['message'].value().doSomething()\"", runtime{enabled: true, script: "doc['message'].value().doSomething()"}, true},
	}

	for _, c := range cases {
		t.Run(c.json, func(t *testing.T) {
			var found runtime
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
