// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValid(t *testing.T) {
	tests := map[string]struct {
		mode  Mode
		valid bool
	}{
		"valid": {
			mode:  LegacyMode,
			valid: true,
		},
		"invalid": {
			mode:  Mode("invalid"),
			valid: false,
		},
		"source": {
			mode:  SourceMode,
			valid: true,
		},
		"build": {
			mode:  BuildMode,
			valid: true,
		},
		"": {
			mode:  Mode(""),
			valid: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.valid, test.mode.Valid(), "mode %s should be %v", test.mode, test.valid)
		})
	}
}
