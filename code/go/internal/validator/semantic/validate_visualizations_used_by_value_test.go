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
		references interface{}
		expected   []reference
	}{
		{
			"References",
			[]interface{}{
				map[string]interface{}{
					"id":   "12345",
					"name": "panel_0",
					"type": "visualization",
				},
				map[string]interface{}{
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
		references interface{}
		expected   []reference
	}{
		{
			"SomeReferences",
			[]interface{}{
				map[string]interface{}{
					"id":   "12345",
					"name": "panel_0",
					"type": "visualization",
				},
				map[string]interface{}{
					"id":   "9000",
					"name": "panel_1",
					"type": "lens",
				},
				map[string]interface{}{
					"id":   "4",
					"name": "panel_2",
					"type": "map",
				},
				map[string]interface{}{
					"id":   "42",
					"name": "panel_3",
					"type": "index-pattern",
				},
				map[string]interface{}{
					"id":   "44",
					"name": "panel_4",
					"type": "search",
				},
				map[string]interface{}{
					"id":   "45",
					"name": "panel_5",
					"type": "tag",
				},
				map[string]interface{}{
					"id":   "50",
					"name": "panel_6",
					"type": "dashboard",
				},
			},
			[]reference{
				{"12345", "panel_0", "visualization"},
				{"9000", "panel_1", "lens"},
				{"4", "panel_2", "map"},
			},
		},
		{
			"Empty",
			[]interface{}{},
			[]reference{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ids, err := anyReference(test.references)
			require.NoError(t, err)
			assert.Equal(t, test.expected, ids)
		})
	}
}
