// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToMapKeysSlice(t *testing.T) {
	tests := []struct {
		title    string
		contents any
		expected []string
	}{
		{
			"keys defined",
			map[string]any{
				"ecs": "a",
				"foo": 4,
			},
			[]string{"ecs", "foo"},
		},
		{
			"empty",
			map[string]any{},
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			list, err := toMapKeysSlice(test.contents)
			require.NoError(t, err)

			assert.Equal(t, test.expected, list)
		})
	}
}
