// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package common

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDefinedWarningsAsErrors(t *testing.T) {
	cases := []struct {
		name        string
		envVarValue string
		expected    bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"other", "other", false},
		{"empty", "", false},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(EnvVarWarningsAsErrors, test.envVarValue)
			t.Cleanup(func() { os.Unsetenv(EnvVarWarningsAsErrors) })

			value := IsDefinedWarningsAsErrors()
			assert.Equal(t, test.expected, value)
		})
	}

	t.Run("undefined", func(t *testing.T) {
		t.Setenv(EnvVarWarningsAsErrors, "")
		value := IsDefinedWarningsAsErrors()
		assert.Equal(t, false, value)
	})
}
