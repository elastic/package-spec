// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateMinimumKibanaVersion(t *testing.T) {
	var tests = []struct {
		version     string
		expectedVal bool
	}{
		{
			"^8.8.0",
			true,
		},
		{
			"^10.11.12",
			true,
		},
		{
			"8.8.0",
			true,
		},
		{
			"^8.8.0 || ^9.9.0",
			true,
		},
		{
			"^8.8.0 || ^9.9.0 || ^10.11.12",
			true,
		},
		{
			"^7.7.0",
			false,
		},
		{
			"^7.7.0 || ^8.8.0",
			false,
		},
		{
			"^7.7.0 || ^10.11.12",
			false,
		},
		{
			"",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			assert.Equal(t, kibanaVersionConditionIsGreaterThanOrEqualTo8_8_0(test.version), test.expectedVal)
		})
	}
}
