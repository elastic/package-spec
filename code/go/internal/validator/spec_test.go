package validator

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestNewSpec(t *testing.T) {
	tests := map[string]struct {
		expectedErrContains string
	}{
		"1.0.0": {},
		"9999.99.999": {
			"no specification found for version [9999.99.999]",
		},
	}

	for version, test := range tests {
		spec, err := NewSpec(*semver.MustParse(version))
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
