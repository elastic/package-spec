// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangelogUniqueVersions(t *testing.T) {
	var tests = []struct {
		title       string
		versions    []string
		expectedErr bool
	}{
		{
			"unique changelog versions",
			[]string{
				"1.0.0",
				"1.0.1",
			},
			false,
		},
		{
			"repeated changelog versions",
			[]string{
				"1.0.0",
				"1.0.1",
				"1.0.0",
			},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			err := ensureUniqueVersions(test.versions)
			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManifestVersionHasChangelogEntry(t *testing.T) {
	var tests = []struct {
		title           string
		manifestVersion string
		versions        []string
		expectedErr     bool
	}{
		{
			"manifest version exists",
			"1.0.1",
			[]string{
				"1.0.1",
				"1.0.0",
			},
			false,
		},
		{
			"manifest version does not exist",
			"1.1.0",
			[]string{
				"1.0.2",
				"1.0.1",
				"1.0.0",
			},
			true,
		},
		{
			"changelog next version",
			"1.0.1",
			[]string{
				"1.1.0-next",
				"1.0.1",
				"1.0.0",
			},
			false,
		},
		{
			"changelog next version manifest older version",
			"1.0.0",
			[]string{
				"1.1.0-next",
				"1.0.1",
				"1.0.0",
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			err := ensureManifestVersionHasChangelogEntry(test.manifestVersion, test.versions)
			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChangelogLatestVersionIsGreaterThanOthers(t *testing.T) {
	var tests = []struct {
		title       string
		versions    []string
		expectedErr bool
	}{
		{
			"latest changelog entry greater",
			[]string{
				"1.0.1",
				"1.0.0",
			},
			false,
		},
		{
			"latest changelog entry lower prerelease tag",
			[]string{
				"1.0.1-preview01",
				"1.0.1",
				"1.0.0",
			},
			true,
		},
		{
			"latest changelog entry lower stable version",
			[]string{
				"1.0.1",
				"1.1.0",
				"1.0.0",
			},
			true,
		},
		{
			"latest changelog entry next version",
			[]string{
				"1.2.0-next",
				"1.1.0",
				"1.0.0",
			},
			false,
		},
		{
			"latest changelog entry next version older",
			[]string{
				"1.0.1-next",
				"1.1.0",
				"1.0.0",
			},
			true,
		},
		{
			"latest changelog entry prerelease tag",
			[]string{
				"1.1.1-preview01",
				"1.1.0",
				"1.0.0",
			},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			err := ensureChangelogLatestVersionIsGreaterThanOthers(test.versions)
			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
