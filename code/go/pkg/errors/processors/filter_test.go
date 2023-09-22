// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

func createValidationErrors(messages []string) ve.ValidationErrors {
	var allErrors ve.ValidationErrors
	for _, m := range messages {
		allErrors = append(allErrors, ve.NewStructuredError(errors.New(m), ve.UnassignedCode))
	}
	return allErrors
}

func createValidationError(message, code string) ve.ValidationError {
	return ve.NewStructuredError(errors.New(message), code)
}

func TestFilter(t *testing.T) {
	cases := []struct {
		title                  string
		config                 ConfigFilter
		errors                 ve.ValidationErrors
		expectedErrors         ve.ValidationErrors
		expectedFilteredErrors ve.ValidationErrors
	}{
		{
			title: "using codes",
			config: ConfigFilter{
				Issues: Processors{
					ExcludeChecks: []string{"CODE01", "CODE03"},
				},
			},
			errors: ve.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedErrors: ve.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("foo bar", "CODE02"),
			},
			expectedFilteredErrors: ve.ValidationErrors{
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar foo", "CODE03"),
			},
		},
		{
			title: "using unassigned code",
			config: ConfigFilter{
				Issues: Processors{
					ExcludeChecks: []string{""},
				},
			},
			errors: ve.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedErrors: ve.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedFilteredErrors: nil,
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
					slices.ContainsFunc(c.expectedErrors, func(elem ve.ValidationError) bool {
						return elem.Error() == e.Error() && elem.Code() == e.Code()
					}),
					"unexpected error: \"%s\" (%s)", e.Error(), e.Code(),
				)
			}
			for _, e := range filteredErrors {
				assert.True(
					t,
					slices.ContainsFunc(c.expectedFilteredErrors, func(elem ve.ValidationError) bool {
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
