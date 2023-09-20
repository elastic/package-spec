// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

func createStructuredError(message string) errors.ValidationError {
	return ve.NewStructuredError(
		fmt.Errorf(message),
		"file",
		"",
		errors.Critical,
	)
}

func TestFilter(t *testing.T) {
	cases := []struct {
		title          string
		config         ConfigFilter
		errors         errors.ValidationErrors
		expectedErrors errors.ValidationErrors
	}{
		{
			title: "test one exclude pattern",
			config: ConfigFilter{
				Issues: Processors{
					ExcludePatterns: []string{"exclude"},
				},
			},
			errors: []errors.ValidationError{
				createStructuredError("exclude error"),
				createStructuredError("other error"),
			},
			expectedErrors: []errors.ValidationError{
				createStructuredError("other error"),
			},
		},
		{
			title: "several exclude pattern",
			config: ConfigFilter{
				Issues: Processors{
					ExcludePatterns: []string{"exclude", "bar$"},
				},
			},
			errors: []errors.ValidationError{
				createStructuredError("exclude error"),
				createStructuredError("other error"),
				createStructuredError("foo bar"),
				createStructuredError("foo bar foo"),
			},
			expectedErrors: []errors.ValidationError{
				createStructuredError("other error"),
				createStructuredError("foo bar foo"),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			runner, err := NewFilter(c.config)
			require.NoError(t, err)

			filteredErrors, err := runner.Run(c.errors)
			require.NoError(t, err)

			assert.Len(t, filteredErrors, len(c.expectedErrors))
			assert.NotEqual(t, c.errors, filteredErrors)
			for _, e := range filteredErrors {
				assert.True(
					t,
					slices.ContainsFunc(c.expectedErrors, func(elem errors.ValidationError) bool {
						return elem.Error() == e.Error()
					}),
					"unexpected error: \"%s\"", e.Error(),
				)
			}
		})
	}
}

func TestLoadConfigFilter(t *testing.T) {
	cases := []struct {
		title                   string
		configPath              string
		expectedExcludePatterns int
	}{
		{
			title:                   "test exclude config",
			configPath:              "testdata/issues.config.yml",
			expectedExcludePatterns: 2,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			config, err := LoadConfigFilter(c.configPath)
			require.NoError(t, err)

			assert.Equal(t, len(config.Issues.ExcludePatterns), c.expectedExcludePatterns)
		})
	}
}
