// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"net/url"
	"testing"

	ve "github.com/elastic/package-spec/code/go/internal/errors"

	"github.com/stretchr/testify/assert"
)

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
			errGHIssue,
		},
		{
			"https://github.com/elastic/integrations/pull/0",
			errGHIssue,
		},
		{
			"https://github.com/elastic/integrations/pull",
			errGHIssue,
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

	var tests = []struct {
		name   string
		links  []string
		errors ve.ValidationErrors
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
			ve.ValidationErrors{
				errGHIssue,
				errGHIssue,
			},
		},
		{
			"SomeInvalidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/1234",
				"https://github.com/elastic/integrations/pull",
			},
			ve.ValidationErrors{
				errGHIssue,
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
