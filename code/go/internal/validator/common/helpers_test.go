package common

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			if err := os.Setenv(EnvVarWarningsAsErrors, test.envVarValue); err != nil {
				require.NoError(t, err)
			}
			value := IsDefinedWarningsAsErrors()
			assert.Equal(t, test.expected, value)

			if err := DisableWarningsAsErrors(); err != nil {
				require.NoError(t, err)
			}
		})
	}

	t.Run("undefined", func(t *testing.T) {
		if err := DisableWarningsAsErrors(); err != nil {
			require.NoError(t, err)
		}
		value := IsDefinedWarningsAsErrors()
		assert.Equal(t, false, value)
	})
}
