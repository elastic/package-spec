// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

func TestExclude(t *testing.T) {

	cases := []struct {
		title    string
		pattern  string
		errors   []string
		expected []string
	}{
		{
			title:    "using pattern",
			pattern:  "^exclude$",
			errors:   []string{"exclude", "1", "", "exclud", "notexclude"},
			expected: []string{"1", "", "exclud", "notexclude"},
		},
		{
			title:    "empty pattern",
			pattern:  "",
			errors:   []string{"exclude", "1", "", "exclud", "notexclude"},
			expected: []string{"exclude", "1", "", "exclud", "notexclude"},
		},
		{
			title:    "exclude all pattern",
			pattern:  ".*",
			errors:   []string{"exclude", "1", "", "exclud", "notexclude"},
			expected: []string{},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewExclude(c.pattern)
			var issues pve.ValidationErrors
			for _, e := range c.errors {
				issues = append(issues, fmt.Errorf(e))
			}

			processedIssues, err := p.Process(issues)
			require.NoError(t, err)

			assert.Len(t, processedIssues, len(c.expected))

			if len(c.expected) == 0 {
				return
			}
			var processedTexts []string
			for _, i := range processedIssues {
				processedTexts = append(processedTexts, i.Error())
			}
			assert.Equal(t, c.expected, processedTexts)
		})
	}
}
