// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
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

func TestValidateGithubLink(t *testing.T) {
	var tests = []struct {
		link        string
		expectedErr error
	}{
		{
			"https://github.com/elastic/integrations/pull/2897",
			nil,
		},
		{
			"https://github.com/elastic/integrations/pull/abcd",
			errGithubIssue,
		},
		{
			"https://github.com/elastic/integrations/pull/0",
			errGithubIssue,
		},
		{
			"https://github.com/elastic/integrations/pull",
			errGithubIssue,
		},
	}
	for _, test := range tests {
		t.Run(test.link, func(t *testing.T) {
			linkURL, _ := url.Parse(test.link)
			err := validateGithubLink(linkURL)
			if err != nil {
				assert.ErrorIs(t, err, test.expectedErr)
			} else {
				assert.Equal(t, err, test.expectedErr)
			}
		})
	}
}

func TestEnsureLinksAreValid(t *testing.T) {
	var githubError = specerrors.NewStructuredError(errGithubIssue, specerrors.UnassignedCode)

	var tests = []struct {
		name   string
		links  []string
		errors specerrors.ValidationErrors
	}{
		{
			"AllValidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/2897",
				"https://github.com/elastic/integrations/pull/1001",
				"https://github.com/elastic/integrations/pull/1",
			},
			nil,
		},
		{
			"AllInvalidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/abcd",
				"https://github.com/elastic/integrations/pull",
			},
			specerrors.ValidationErrors{
				githubError,
				githubError,
			},
		},
		{
			"SomeInvalidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/1234",
				"https://github.com/elastic/integrations/pull",
			},
			specerrors.ValidationErrors{
				githubError,
			},
		},
		{
			"IgnoreCasesOtherThanGithubDotCom",
			[]string{
				"https://gitlab.com/elastic/integrations/pull/abcd",
				"https://zzz.com/elastic/integrations/pull/1234",
			},
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := ensureLinksAreValid(test.links)
			if test.errors == nil {
				assert.Equal(t, test.errors, errs)
			} else {
				assert.Equal(t, len(errs), len(test.errors))
			}
		})
	}
}
