// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import "testing"

func TestChangelogLinks(t *testing.T) {

	var tests = []struct {
		name        string
		links       []string
		expectError bool
	}{
		{
			"ValidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/2897",
				"https://github.com/elastic/integrations/pull/1001",
				"https://github.com/elastic/integrations/pull/1",
			},
			false,
		},
		{
			"InvalidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/abcd",
				"https://github.com/elastic/integrations/pull",
			},
			true,
		},
		{
			"SomeInvalidLinks",
			[]string{
				"https://github.com/elastic/integrations/pull/1234",
				"https://github.com/elastic/integrations/pull",
			},
			true,
		},
		{
			"BadLink",
			[]string{
				"https://github.com/elastic/integrations/pull/0",
			},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ensureLinksAreValid(test.links)
			if test.expectError && err == nil {
				t.Error("expecting error")
			} else if !test.expectError && err != nil {
				t.Errorf("expecting no error while got %v", err)
			}
		})
	}
}
