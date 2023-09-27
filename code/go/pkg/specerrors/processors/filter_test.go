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

	"github.com/elastic/package-spec/v2/code/go/pkg/specerrors"
)

func createValidationErrors(messages []string) specerrors.ValidationErrors {
	var allErrors specerrors.ValidationErrors
	for _, m := range messages {
		allErrors = append(allErrors, specerrors.NewStructuredErrorf(m))
	}
	return allErrors
}

func createValidationError(message, code string) specerrors.ValidationError {
	return specerrors.NewStructuredError(errors.New(message), code)
}

func TestFilter(t *testing.T) {
	cases := []struct {
		title                  string
		config                 ConfigFilter
		errors                 specerrors.ValidationErrors
		expectedErrors         specerrors.ValidationErrors
		expectedFilteredErrors specerrors.ValidationErrors
	}{
		{
			title: "using codes",
			config: ConfigFilter{
				Errors: Processors{
					ExcludeChecks: []string{"CODE01", "CODE03"},
				},
			},
			errors: specerrors.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedErrors: specerrors.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("foo bar", "CODE02"),
			},
			expectedFilteredErrors: specerrors.ValidationErrors{
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar foo", "CODE03"),
			},
		},
		{
			title: "using unassigned code",
			config: ConfigFilter{
				Errors: Processors{
					ExcludeChecks: []string{""},
				},
			},
			errors: specerrors.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedErrors: specerrors.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedFilteredErrors: nil,
		},
		{
			title: "filter all errors",
			config: ConfigFilter{
				Errors: Processors{
					ExcludeChecks: []string{"CODE00"},
				},
			},
			errors: specerrors.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
			},
			expectedErrors: nil,
			expectedFilteredErrors: specerrors.ValidationErrors{
				createValidationError("exclude error", "CODE00"),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			runner := NewFilter(&c.config)

			result, err := runner.Run(c.errors)
			require.NoError(t, err)

			if c.expectedErrors == nil {
				assert.Nil(t, result.Processed)
			}
			if c.expectedFilteredErrors == nil {
				assert.Nil(t, result.Removed)
			}

			if c.expectedErrors != nil {
				veFinalErrors, ok := result.Processed.(specerrors.ValidationErrors)
				require.True(t, ok)
				assert.Len(t, veFinalErrors, len(c.expectedErrors))
				for _, e := range veFinalErrors {
					assert.True(
						t,
						slices.ContainsFunc(c.expectedErrors, func(elem specerrors.ValidationError) bool {
							return elem.Error() == e.Error() && elem.Code() == e.Code()
						}),
						"unexpected error: \"%s\" (%s)", e.Error(), e.Code(),
					)
				}
			}

			if c.expectedFilteredErrors != nil {
				veFilteredErrors, ok := result.Removed.(specerrors.ValidationErrors)
				require.True(t, ok)
				assert.Len(t, veFilteredErrors, len(c.expectedFilteredErrors))
				for _, e := range veFilteredErrors {
					assert.True(
						t,
						slices.ContainsFunc(c.expectedFilteredErrors, func(elem specerrors.ValidationError) bool {
							return elem.Error() == e.Error()
						}),
						"unexpected filtered error: \"%s\"", e.Error(),
					)
				}
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
			configPath:              "testdata/errors.config.yml",
			expectedExcludePatterns: 2,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			fsys := os.DirFS(".")
			config, err := LoadConfigFilter(fsys, c.configPath)
			require.NoError(t, err)

			assert.Equal(t, len(config.Errors.ExcludeChecks), c.expectedExcludePatterns)
		})
	}
}
