// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
)

func TestValidateKibanaVersionGreaterThan(t *testing.T) {
	var tests = []struct {
		version     string
		expectedVal bool
	}{
		{
			"^8.8.0",
			true,
		},
		{
			"^10.11.12",
			true,
		},
		{
			"8.8.0",
			true,
		},
		{
			"^8.8.0 || ^9.9.0",
			true,
		},
		{
			"^8.8.0 || ^9.9.0 || ^10.11.12",
			true,
		},
		{
			"^7.7.0",
			false,
		},
		{
			"^7.7.0 || ^8.8.0",
			false,
		},
		{
			"^7.7.0 || ^10.11.12",
			false,
		},
		{
			"",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			assert.Equal(t, kibanaVersionConditionIsGreaterThanOrEqualTo(test.version, "8.8.0"), test.expectedVal)
		})
	}
}
func TestValidateMinimumKibanaVersionInputPackages(t *testing.T) {

	kbnVersionError := errors.New("conditions.kibana.version must be ^8.8.0 or greater for non experimental input packages (version > 1.0.0)")
	var tests = []struct {
		packageType            string
		packageVersion         semver.Version
		kibanaVersionCondition string
		expectedErr            error
	}{
		{
			"integration",
			*semver.MustParse("1.0.0"),
			"^7.14.0",
			nil,
		},
		{
			"input",
			*semver.MustParse("0.1.0"),
			"^7.14.0",
			nil,
		},
		{
			"input",
			*semver.MustParse("1.0.0"),
			"^7.14.0",
			kbnVersionError,
		},
		{
			"input",
			*semver.MustParse("1.0.0"),
			"^8.8.0 || ^7.14.0",
			kbnVersionError,
		},
		{
			"input",
			*semver.MustParse("1.0.0"),
			"^8.8.0",
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.packageType+"--"+test.packageVersion.String()+"--"+test.kibanaVersionCondition, func(t *testing.T) {
			res := validateMinimumKibanaVersionInputPackages(test.packageType, test.packageVersion, test.kibanaVersionCondition)

			if test.expectedErr == nil {
				assert.Nil(t, res)
			} else {
				require.Error(t, res)
				assert.Equal(t, test.expectedErr.Error(), res.Error())
			}
		})
	}
}

func TestValidateMinimumKibanaVersionRuntimeFields(t *testing.T) {
	kbnVersionError := errors.New("conditions.kibana.version must be ^8.10.0 or greater to include runtime fields")
	var tests = []struct {
		pkgRoot                string
		packageVersion         semver.Version
		kibanaVersionCondition string
		expectedErr            error
	}{
		{
			"../../../../../test/packages/good_v2",
			*semver.MustParse("1.0.0"),
			"^7.14.0",
			kbnVersionError,
		},
		{
			"../../../../../test/packages/good_v2",
			*semver.MustParse("0.1.0"),
			"^7.14.0",
			kbnVersionError,
		},
		{
			"../../../../../test/packages/good_v2",
			*semver.MustParse("1.0.0"),
			"^7.14.0",
			kbnVersionError,
		},
		{
			"../../../../../test/packages/good_v2",
			*semver.MustParse("1.0.0"),
			"^8.10.0 || ^7.14.0",
			kbnVersionError,
		},
		{
			"../../../../../test/packages/good_v2",
			*semver.MustParse("1.0.0"),
			"^8.10.0",
			nil,
		},
	}

	for _, test := range tests {
		t.Run(filepath.Base(test.pkgRoot)+"--"+test.packageVersion.String()+"--"+test.kibanaVersionCondition, func(t *testing.T) {
			res := validateMinimumKibanaVersionRuntimeFields(fspath.DirFS(test.pkgRoot), test.packageVersion, test.kibanaVersionCondition)

			if test.expectedErr == nil {
				assert.Nil(t, res)
			} else {
				require.Error(t, res)
				assert.Equal(t, test.expectedErr.Error(), res.Error())
			}
		})
	}
}

func TestValidateMinimumKibanaVersionSavedObjectsTags(t *testing.T) {
	kbnVersionError := errors.New("conditions.kibana.version must be ^8.10.0 or greater to include saved object tags file: kibana/tags.yml")
	var tests = []struct {
		pkgRoot                string
		pkgType                string
		packageVersion         semver.Version
		kibanaVersionCondition string
		expectedErr            error
	}{
		{
			"../../../../../test/packages/good_v2",
			"integration",
			*semver.MustParse("1.0.0"),
			"^7.14.0",
			kbnVersionError,
		},
		{
			"../../../../../test/packages/good_v2",
			"integration",
			*semver.MustParse("1.0.0"),
			"^8.10.0 || ^7.14.0",
			kbnVersionError,
		},
		{
			"../../../../../test/packages/good_v2",
			"integration",
			*semver.MustParse("1.0.0"),
			"^8.10.0",
			nil,
		},
		{
			"../../../../../test/packages/good_input",
			"input",
			*semver.MustParse("1.0.0"),
			"^7.17.0",
			nil,
		},
	}

	for _, test := range tests {
		t.Run(filepath.Base(test.pkgRoot)+"--"+test.packageVersion.String()+"--"+test.kibanaVersionCondition, func(t *testing.T) {
			res := validateMinimumKibanaVersionSavedObjectTags(fspath.DirFS(test.pkgRoot), test.pkgType, test.packageVersion, test.kibanaVersionCondition)

			if test.expectedErr == nil {
				assert.Nil(t, res)
			} else {
				require.Error(t, res)
				assert.Equal(t, test.expectedErr.Error(), res.Error())
			}
		})
	}
}
