// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, kibanaVersionConditionIsGreaterThanOrEqualTo8_8_0(test.version), test.expectedVal)
		})
	}
}
func TestValidateMinimumKibanaVersion(t *testing.T) {

	kbnVersionError := errors.New("Warning: conditions.kibana.version must be ^8.8.0 or greater for non experimental input packages (version > 1.0.0)")
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
			res := validateMinimumKibanaVersion(test.packageType, test.packageVersion, test.kibanaVersionCondition)

			if test.expectedErr == nil {
				assert.Nil(t, res)
			} else {
				assert.Equal(t, test.expectedErr.Error(), res.Error())
			}
		})
	}
}
