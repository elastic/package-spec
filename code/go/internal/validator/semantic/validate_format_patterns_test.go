// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateFormatPatterns(t *testing.T) {
	cases := []struct {
		pattern string
		valid   bool
	}{
		{"", true},
		{`\[\\/*?\"<>|\s,#-]+`, true},
		{`(?![# -]+)`, true},
		{`\[\s,#-]+[`, false},
	}

	for _, c := range cases {
		t.Run("/"+c.pattern+"/", func(t *testing.T) {
			err := validateFormatPattern(c.pattern)
			if c.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

		})
	}
}
