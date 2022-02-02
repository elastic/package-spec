// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePrerelease(t *testing.T) {
	cases := []struct {
		version string
		valid   bool
	}{
		{"0.1.0", true},
		{"0.1.0-beta", false},
		{"0.1.0-rc1", false},
		{"1.0.0-beta", true},
		{"1.0.0-beta1", true},
		{"1.0.0-beta-1.0", true},
		{"1.0.0-beta.42", true},
		{"1.0.0-beta.alpha", true},
		{"1.0.0-beta+20220202", true},
		{"1.0.0-beta2+20220202", true},
		{"1.0.0-rc1", true},
		{"1.0.0-preview1", true},
		{"1.0.0-betapreview", false},
		{"1.0.0-alphabeta", false},
		{"1.0.0-123", false},
		{"1.0.0-123", false},

		// For convenience with previous recommendations
		{"1.0.0-SNAPSHOT", true},
		{"1.0.0-next", true},
	}

	for _, c := range cases {
		t.Run(c.version, func(t *testing.T) {
			err := validatePrerelease(c.version)
			if c.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
