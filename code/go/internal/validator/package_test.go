package validator

import (
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestNewPackage(t *testing.T) {
	tests := map[string]struct {
		expectedErrContains string
		expectedSpecVersion *semver.Version
	}{
		"good": {
			expectedSpecVersion: semver.MustParse("1.0.4"),
		},
		"non_existent": {
			expectedErrContains: "no package found at",
		},
		"no_manifest": {
			expectedErrContains: "no package manifest file found at",
		},
		"no_spec_version": {
			expectedErrContains: "could not read specification version",
		},
	}

	for pkgName, test := range tests {
		t.Run(pkgName, func(t *testing.T) {
			pkgRootPath := filepath.Join("test", "packages", pkgName)
			pkg, err := NewPackage(pkgRootPath)
			if test.expectedErrContains == "" {
				require.NoError(t, err)
				require.Equal(t, test.expectedSpecVersion, pkg.SpecVersion)
				require.Equal(t, pkgRootPath, pkg.RootPath)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrContains)
				require.Nil(t, pkg)
			}
		})
	}
}
