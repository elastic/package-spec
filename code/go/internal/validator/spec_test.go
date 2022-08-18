// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/code/go/internal/fspath"
)

func TestNewSpec(t *testing.T) {
	tests := map[string]struct {
		expectedErrContains string
	}{
		"1.0.0": {},
		"9999.99.999": {
			"could not load specification for version [9999.99.999]",
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

func TestNoBetaFeatures_Package_GA(t *testing.T) {
	// given
	s := Spec{
		*semver.MustParse("1.0.0"),
		fspath.DirFS("testdata/fakespec"),
		"1",
	}
	pkg, err := NewPackage("testdata/packages/features_ga")
	require.NoError(t, err)

	err = s.ValidatePackage(*pkg)
	require.Empty(t, err)
}

func TestBetaFeatures_Package_GA(t *testing.T) {
	// given
	s := Spec{
		*semver.MustParse("1.0.0"),
		fspath.DirFS("testdata/fakespec"),
		"1",
	}
	pkg, err := NewPackage("testdata/packages/features_beta")
	require.NoError(t, err)

	errs := s.ValidatePackage(*pkg)
	require.Len(t, errs, 1)
	require.Equal(t, errs[0].Error(), "spec for [testdata/packages/features_beta/beta] defines beta features which can't be enabled for packages with a stable semantic version")
}
