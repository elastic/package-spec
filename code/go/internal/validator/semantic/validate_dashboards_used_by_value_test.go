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
		expected   bool
	}{
		{
			"AllReferences",
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
					"type": "visualization",
				},
			},
			true,
		},
		{
			"Empty",
			"path",
			[]map[string]string{},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			anyReference, err := anyReference(test.references, test.path)
			require.NoError(t, err)
			assert.Equal(t, test.expected, anyReference)
		})
	}
}
