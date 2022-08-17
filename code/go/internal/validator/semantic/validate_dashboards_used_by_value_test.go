// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToReferencesToSlice(t *testing.T) {

	var tests = []struct {
		name       string
		references []map[string]string
		expected   []reference
	}{
		{
			"References",
			[]map[string]string{
				{
					"id":   "12345",
					"name": "panel_0",
					"type": "visualization",
				},
				{
					"id":   "9000",
					"name": "panel_1",
					"type": "other",
				},
			},
			[]reference{
				reference{
					ID:   "12345",
					Name: "panel_0",
					Type: "visualization",
				},
				reference{
					ID:   "9000",
					Name: "panel_1",
					Type: "other",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			references, err := toReferenceSlice(test.references)
			require.NoError(t, err)
			assert.Equal(t, test.expected, references)
		})
	}
}

func TestAnyReference(t *testing.T) {

	var tests = []struct {
		name       string
		path       string
		references []map[string]string
		expected   []string
	}{
		{
			"SomeReferences",
			"path",
			[]map[string]string{
				{
					"id":   "12345",
					"name": "panel_0",
					"type": "visualization",
				},
				{
					"id":   "9000",
					"name": "panel_1",
					"type": "lens",
				},
				{
					"id":   "4",
					"name": "panel_1",
					"type": "map",
				},
				{
					"id":   "42",
					"name": "panel_1",
					"type": "index-pattern",
				},
			},
			[]string{"12345", "9000"},
		},
		{
			"Empty",
			"path",
			[]map[string]string{},
			[]string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ids, err := anyReference(test.references, test.path)
			require.NoError(t, err)
			assert.Equal(t, test.expected, ids)
		})
	}
}
