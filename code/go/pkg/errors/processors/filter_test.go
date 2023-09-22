// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

func TestFilter(t *testing.T) {
	cases := []struct {
		title                  string
		config                 ConfigFilter
		errors                 errors.ValidationErrors
		expectedErrors         errors.ValidationErrors
		expectedFilteredErrors errors.ValidationErrors
	}{
		{
			title: "test one exclude pattern",
			config: ConfigFilter{
				Issues: Processors{
					ExcludePatterns: []string{"exclude"},
				},
			},
			errors: errors.ValidationErrors{
				fmt.Errorf("exclude error"),
				fmt.Errorf("other error"),
			},
			expectedErrors: errors.ValidationErrors{
				fmt.Errorf("other error"),
			},
			expectedFilteredErrors: errors.ValidationErrors{
				fmt.Errorf("exclude error"),
			},
		},
		{
			title: "several exclude pattern",
			config: ConfigFilter{
				Issues: Processors{
					ExcludePatterns: []string{"exclude", "bar$"},
				},
			},
			errors: errors.ValidationErrors{
				fmt.Errorf("exclude error"),
				fmt.Errorf("other error"),
				fmt.Errorf("foo bar"),
				fmt.Errorf("foo bar foo"),
			},
			expectedErrors: errors.ValidationErrors{
				fmt.Errorf("other error"),
				fmt.Errorf("foo bar foo"),
			},
			expectedFilteredErrors: errors.ValidationErrors{
				fmt.Errorf("exclude error"),
				fmt.Errorf("foo bar"),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			runner := NewFilter(&c.config)

			finalErrors, filteredErrors, err := runner.Run(c.errors)
			require.NoError(t, err)

			assert.Len(t, finalErrors, len(c.expectedErrors))
			assert.Len(t, filteredErrors, len(c.expectedFilteredErrors))

			for _, e := range finalErrors {
				assert.True(
					t,
					slices.ContainsFunc(c.expectedErrors, func(elem error) bool {
						return elem.Error() == e.Error()
					}),
					"unexpected error: \"%s\"", e.Error(),
				)
			}
			for _, e := range filteredErrors {
				assert.True(
					t,
					slices.ContainsFunc(c.expectedFilteredErrors, func(elem error) bool {
						return elem.Error() == e.Error()
					}),
					"unexpected filtered error: \"%s\"", e.Error(),
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
			fsys := os.DirFS(".")
			config, err := LoadConfigFilter(fsys, c.configPath)
			require.NoError(t, err)

			assert.Equal(t, len(config.Issues.ExcludePatterns), c.expectedExcludePatterns)
		})
	}
}
