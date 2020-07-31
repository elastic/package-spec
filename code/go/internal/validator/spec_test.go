package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSpec(t *testing.T) {
	tests := map[string]struct {
		expectedErrContains string
	}{
		"1.0.0": {},
		"non_existent": {
			"no specification found for version [non_existent]",
		},
	}

	for version, test := range tests {
		spec, err := NewSpec(version)
		if test.expectedErrContains == "" {
			require.NoError(t, err)
			require.IsType(t, &Spec{}, spec)
		} else {
			require.Error(t, err)
			require.Contains(t, err.Error(), test.expectedErrContains)
			require.Nil(t, spec)
		}
	}
}
