// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spectypes

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentTypeMarshalJSON(t *testing.T) {
	jsonContentType := ContentType{"application/json", nil}
	yamlContentType := ContentType{"application/x-yaml", map[string]string{
		"require-document-dashes": "true",
	}}

	cases := []struct {
		contentType ContentType
		expected    string
	}{
		{ContentType{}, `""`},
		{jsonContentType, `"application/json"`},
		{yamlContentType, `"application/x-yaml; require-document-dashes=true"`},
	}

	for _, c := range cases {
		t.Run(c.expected, func(t *testing.T) {
			d, err := json.Marshal(c.contentType)
			require.NoError(t, err)
			assert.Equal(t, c.expected, string(d))
		})
	}
}

func TestContentTypeMarshalYAML(t *testing.T) {
	jsonContentType := ContentType{"application/json", nil}
	yamlContentType := ContentType{"application/x-yaml", map[string]string{
		"require-document-dashes": "true",
	}}

	cases := []struct {
		contentType ContentType
		expected    string
	}{
		{ContentType{}, "\"\"\n"},
		{jsonContentType, "application/json\n"},
		{yamlContentType, "application/x-yaml; require-document-dashes=true\n"},
	}

	for _, c := range cases {
		t.Run(c.expected, func(t *testing.T) {
			d, err := yaml.Marshal(c.contentType)
			require.NoError(t, err)
			assert.Equal(t, c.expected, string(d))
		})
	}
}

func TestContentTypeUnmarshal(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		testContentTypeUnmarshalFormat(t, json.Unmarshal)
	})
	t.Run("yaml", func(t *testing.T) {
		testContentTypeUnmarshalFormat(t, yaml.Unmarshal)
	})
}

func testContentTypeUnmarshalFormat(t *testing.T, unmarshaler func([]byte, interface{}) error) {
	cases := []struct {
		json           string
		expectedType   string
		expectedParams map[string]string
		valid          bool
	}{
		{`"application/json"`, "application/json", nil, true},
		{
			`"application/x-yaml; require-document-dashes=true"`,
			"application/x-yaml",
			map[string]string{"require-document-dashes": "true"},
			true,
		},
		{
			`"application/x-yaml; require-document-dashes=true; charset=utf-8"`,
			"application/x-yaml",
			map[string]string{
				"require-document-dashes": "true",
				"charset":                 "utf-8",
			},
			true,
		},
		{`"application`, "", nil, false},
		{`""`, "", nil, false},
		{`"application/json; charset"`, "", nil, false},
	}

	for _, c := range cases {
		t.Run(c.json, func(t *testing.T) {
			var found ContentType
			err := unmarshaler([]byte(c.json), &found)
			if c.valid {
				require.NoError(t, err)
				assert.Equal(t, c.expectedType, found.MediaType)
				if len(c.expectedParams) == 0 {
					assert.Empty(t, found.Params)
				} else {
					assert.EqualValues(t, c.expectedParams, found.Params)
				}
			} else {
				t.Log(found)
				require.Error(t, err)
			}
		})
	}
}
