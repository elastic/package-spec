// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangelogLinks(t *testing.T) {

	var tests = []struct {
		name      string
		links     []string
		numErrors int
	}{
		{
			"AllValidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/2897",
				"https://github.com/elastic/integrations/pull/1001",
				"https://github.com/elastic/integrations/pull/1",
			},
			0,
		},
		{
			"AllInvalidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/abcd",
				"https://github.com/elastic/integrations/pull",
			},
			2,
		},
		{
			"SomeInvalidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/1234",
				"https://github.com/elastic/integrations/pull",
			},
			1,
		},
		{
			"BadLink",
			[]string{
				"https://github.com/elastic/integrations/pull/0",
			},
			1,
		},
		{
			"IgnoreCasesOtherThanGithubDotCom",
			[]string{
				"https://gitlab.com/elastic/integrations/pull/abcd",
				"https://zzz.com/elastic/integrations/pull/1234",
			},
			0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := ensureLinksAreValid(test.links)
			assert.Equal(t, test.numErrors, len(errs))
			if test.numErrors > 0 && errs == nil {
				t.Error("expecting error")
			} else if test.numErrors == 0 && errs != nil {
				t.Errorf("expecting no error while got %v", errs)
			}
		})
	}
}
