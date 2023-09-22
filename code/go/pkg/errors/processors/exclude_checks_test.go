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

func TestExcludeChecks(t *testing.T) {

	cases := []struct {
		title            string
		code             string
		codes            []string
		expected         []string
		expectedFiltered []string
	}{
		{
			title:            "exclude specific code",
			code:             "CODE01",
			codes:            []string{"CODE01", "OTHER", "", "42", "NOCODE01"},
			expected:         []string{"OTHER", "", "42", "NOCODE01"},
			expectedFiltered: []string{"CODE01"},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewExcludeCheck(c.code)
			var issues pve.ValidationErrors
			for i, code := range c.codes {
				issues = append(issues,
					pve.NewStructuredError(fmt.Errorf("error %d", i), code),
				)
			}

			processedIssues, filteredIssues, err := p.Process(issues)
			require.NoError(t, err)

			assert.Len(t, processedIssues, len(c.expected))
			assert.Len(t, filteredIssues, len(c.expectedFiltered))

			var processedTexts []string
			for _, i := range filteredIssues {
				processedTexts = append(processedTexts, i.Code())
			}
			assert.Equal(t, c.expectedFiltered, processedTexts)

			processedTexts = []string{}
			for _, i := range processedIssues {
				processedTexts = append(processedTexts, i.Code())
			}
			assert.Equal(t, c.expected, processedTexts)
		})
	}
}
