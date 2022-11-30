// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
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
	}
	pkg, err := NewPackage("testdata/packages/features_beta")
	require.NoError(t, err)

	errs := s.ValidatePackage(*pkg)
	require.Len(t, errs, 1)
	require.Equal(t, errs[0].Error(), "spec for [testdata/packages/features_beta/beta] defines beta features which can't be enabled for packages with a stable semantic version")
}

func TestMarshal_AllJSSchema(t *testing.T) {
	// given
	cases := []struct {
		title               string
		version             string
		specPath            string
		filePath            string
		expectedError       bool
		expectedOutputPath  string
		expectedNumberFiles int
	}{
		{
			title:               "manifest from version 1.0.0",
			version:             "1.0.0",
			specPath:            "testdata/simple-spec",
			filePath:            "manifest.yml",
			expectedError:       false,
			expectedOutputPath:  "testdata/manifest-1.0.0.yml",
			expectedNumberFiles: 4,
		},
		{
			title:               "manifest from version 2.1.0",
			version:             "2.1.0",
			specPath:            "testdata/simple-spec",
			filePath:            "manifest.yml",
			expectedError:       false,
			expectedOutputPath:  "testdata/manifest-2.1.0.yml",
			expectedNumberFiles: 4,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			s := Spec{
				*semver.MustParse(c.version),
				fspath.DirFS(c.specPath),
			}
			rendered, err := s.AllJSONSchema("")
			if c.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			contents, err := os.ReadFile(c.expectedOutputPath)
			require.NoError(t, err)

			// check contents of one file
			for _, jsonschema := range rendered {
				if jsonschema.name != c.filePath {
					continue
				}
				assert.Equal(t, string(contents), string(jsonschema.schemaJSON))
				break
			}

			assert.Equal(t, c.expectedNumberFiles, len(rendered))
		})
	}
}

func TestMarshal_GivenJSONSchema(t *testing.T) {
	// given
	cases := []struct {
		title              string
		version            string
		specPath           string
		filePath           string
		expectedError      bool
		expectedOutputPath string
	}{
		{
			title:              "not found",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "noexit.yml",
			expectedError:      true,
			expectedOutputPath: "",
		},
		{
			title:              "manifest from version 1.0.0",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "manifest.yml",
			expectedError:      false,
			expectedOutputPath: "testdata/manifest-1.0.0.yml",
		},
		{
			title:              "manifest from version 2.1.0",
			version:            "2.1.0",
			specPath:           "testdata/simple-spec",
			filePath:           "manifest.yml",
			expectedError:      false,
			expectedOutputPath: "testdata/manifest-2.1.0.yml",
		},
		{
			title:              "file with regex",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "data_2.yml",
			expectedError:      false,
			expectedOutputPath: "testdata/data-1.0.0.yml",
		},
		{
			title:              "file with regex not found",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "data_ng.yml",
			expectedError:      true,
			expectedOutputPath: "",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			s := Spec{
				*semver.MustParse(c.version),
				fspath.DirFS(c.specPath),
			}
			rendered, err := s.JSONSchema(c.filePath, "")
			if c.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			contents, err := os.ReadFile(c.expectedOutputPath)
			require.NoError(t, err)
			assert.Equal(t, string(contents), string(rendered.schemaJSON))
		})
	}
}
