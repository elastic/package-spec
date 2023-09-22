// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

func TestExclude(t *testing.T) {

	cases := []struct {
		title            string
		pattern          string
		errors           []string
		expected         []string
		expectedFiltered []string
	}{
		{
			title:            "using pattern",
			pattern:          "^exclude$",
			errors:           []string{"exclude", "1", "", "exclud", "notexclude"},
			expected:         []string{"1", "", "exclud", "notexclude"},
			expectedFiltered: []string{"exclude"},
		},
		{
			title:            "empty pattern",
			pattern:          "",
			errors:           []string{"exclude", "1", "", "exclud", "notexclude"},
			expected:         []string{"exclude", "1", "", "exclud", "notexclude"},
			expectedFiltered: nil,
		},
		{
			title:            "exclude all pattern",
			pattern:          ".*",
			errors:           []string{"exclude", "1", "", "exclud", "notexclude"},
			expected:         []string{},
			expectedFiltered: []string{"exclude", "1", "", "exclud", "notexclude"},
		},
		{
			title:            "containing a substring pattern",
			pattern:          "excl",
			errors:           []string{"exclude", "1", "", "exclud", "notexclude"},
			expected:         []string{"1", ""},
			expectedFiltered: []string{"exclude", "exclud", "notexclude"},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewExclude(c.pattern)
			var issues pve.ValidationErrors
			for _, e := range c.errors {
				issues = append(issues, pve.NewStructuredError(errors.New(e), pve.TODO_code))
			}

			processedIssues, filteredIssues, err := p.Process(issues)
			require.NoError(t, err)

			assert.Len(t, processedIssues, len(c.expected))
			assert.Len(t, filteredIssues, len(c.expectedFiltered))

			var processedTexts []string
			for _, i := range filteredIssues {
				processedTexts = append(processedTexts, i.Error())
			}
			assert.Equal(t, c.expectedFiltered, processedTexts)

			processedTexts = []string{}
			for _, i := range processedIssues {
				processedTexts = append(processedTexts, i.Error())
			}
			assert.Equal(t, c.expected, processedTexts)
		})
	}
}
