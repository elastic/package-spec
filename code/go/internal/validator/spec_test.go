// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/packages"
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
		*semver.MustParse("1.0.0"),
		fspath.DirFS("testdata/fakespec"),
	}
	pkg, err := packages.NewPackage("testdata/packages/features_ga")
	require.NoError(t, err)

	err = s.ValidatePackage(*pkg)
	require.Empty(t, err)
}

func TestBetaFeatures_Package_GA(t *testing.T) {
	// given
	s := Spec{
		*semver.MustParse("1.0.0"),
		*semver.MustParse("1.0.0"),
		fspath.DirFS("testdata/fakespec"),
	}
	pkg, err := packages.NewPackage("testdata/packages/features_beta")
	require.NoError(t, err)

	errs := s.ValidatePackage(*pkg)
	require.Len(t, errs, 1)
	require.Equal(t, "spec for [testdata/packages/features_beta/beta] defines beta features which can't be enabled for packages with a stable semantic version", errs[0].Error())
}

func TestFolderSpecInvalid(t *testing.T) {
	// given
	cases := []struct {
		title          string
		version        semver.Version
		spec           fspath.FS
		pkgPath        string
		valid          bool
		expectedErrors []string
	}{
		{
			title:   "valid spec 1.0.0",
			version: *semver.MustParse("1.0.0"),
			spec:    fspath.DirFS("testdata/fakespec"),
			pkgPath: "testdata/packages/folder_spec_patches",
			valid:   true,
		},
		{
			title:   "invalid spec - extra file 2.0.0",
			version: *semver.MustParse("2.0.0"),
			spec:    fspath.DirFS("testdata/fakespec"),
			pkgPath: "testdata/packages/folder_spec_patches",
			valid:   false,
			expectedErrors: []string{
				"item [other.yml] is not allowed in folder [testdata/packages/folder_spec_patches/patches]",
				"expecting to find [data_stream] folder in folder [testdata/packages/folder_spec_patches/patches]",
			},
		},
		{
			title:   "invalid spec - extra file 2.1.0",
			version: *semver.MustParse("2.1.0"),
			spec:    fspath.DirFS("testdata/fakespec"),
			pkgPath: "testdata/packages/folder_spec_patches",
			valid:   false,
			expectedErrors: []string{
				"item [other.yml] is not allowed in folder [testdata/packages/folder_spec_patches/patches]",
			},
		},
		{
			title:   "invalid spec chaining patches- extra file 2.9.0",
			version: *semver.MustParse("2.9.0"),
			spec:    fspath.DirFS("testdata/fakespec"),
			pkgPath: "testdata/packages/folder_spec_patches_chain",
			valid:   false,
			expectedErrors: []string{
				"item [other.yml] is not allowed in folder [testdata/packages/folder_spec_patches_chain/patches]",
				"expecting to find [other.yml] file in folder [testdata/packages/folder_spec_patches_chain/patches/data_stream]",
			},
		},
		{
			title:   "invalid spec chaining patches- extra file 3.0.0",
			version: *semver.MustParse("3.0.0"),
			spec:    fspath.DirFS("testdata/fakespec"),
			pkgPath: "testdata/packages/folder_spec_patches_chain",
			valid:   false,
			expectedErrors: []string{
				"item [other.yml] is not allowed in folder [testdata/packages/folder_spec_patches_chain/patches]",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			s := Spec{
				c.version,
				c.version,
				c.spec,
			}
			pkg, err := packages.NewPackage(c.pkgPath)
			require.NoError(t, err)

			errs := s.ValidatePackage(*pkg)
			if c.valid {
				require.Empty(t, errs)
				return
			}

			require.Len(t, errs, len(c.expectedErrors))
			for e := range errs {
				assert.Contains(t, c.expectedErrors, errs[e].Error())
			}
		})
	}

}
