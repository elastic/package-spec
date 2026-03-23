// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specerrors

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createValidationError(message, code string) ValidationError {
	return NewStructuredError(errors.New(message), code)
}

func TestFilter(t *testing.T) {
	cases := []struct {
		title                  string
		config                 ConfigFilter
		errors                 ValidationErrors
		expectedErrors         ValidationErrors
		expectedFilteredErrors ValidationErrors
	}{
		{
			title: "using codes",
			config: ConfigFilter{
				Errors: Processors{
					ExcludeChecks: []string{"CODE01", "CODE03"},
				},
			},
			errors: ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedErrors: ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("foo bar", "CODE02"),
			},
			expectedFilteredErrors: ValidationErrors{
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
			errors: ValidationErrors{
				createValidationError("exclude error", "CODE00"),
				createValidationError("other error", "CODE01"),
				createValidationError("foo bar", "CODE02"),
				createValidationError("foo bar foo", "CODE03"),
			},
			expectedErrors: ValidationErrors{
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
			errors: ValidationErrors{
				createValidationError("exclude error", "CODE00"),
			},
			expectedErrors: nil,
			expectedFilteredErrors: ValidationErrors{
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
				veFinalErrors, ok := result.Processed.(ValidationErrors)
				require.True(t, ok)
				assert.Len(t, veFinalErrors, len(c.expectedErrors))
				for _, e := range veFinalErrors {
					assert.True(
						t,
						slices.ContainsFunc(c.expectedErrors, func(elem ValidationError) bool {
							return elem.Error() == e.Error() && elem.Code() == e.Code()
						}),
						"unexpected error: \"%s\" (%s)", e.Error(), e.Code(),
					)
				}
			}

			if c.expectedFilteredErrors != nil {
				veFilteredErrors, ok := result.Removed.(ValidationErrors)
				require.True(t, ok)
				assert.Len(t, veFilteredErrors, len(c.expectedFilteredErrors))
				for _, e := range veFilteredErrors {
					assert.True(
						t,
						slices.ContainsFunc(c.expectedFilteredErrors, func(elem ValidationError) bool {
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
		packagePath             string
		expectedExcludePatterns int
		configPathFound         bool
	}{
		{
			title:                   "test exclude config",
			packagePath:             "testdata/",
			expectedExcludePatterns: 2,
			configPathFound:         true,
		},
		{
			title:                   "test exclude config",
			packagePath:             ".",
			expectedExcludePatterns: 0,
			configPathFound:         false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			fsys := os.DirFS(c.packagePath)
			config, err := LoadConfigFilter(fsys)
			if c.configPathFound {
				require.NoError(t, err)
				assert.Equal(t, len(config.Errors.ExcludeChecks), c.expectedExcludePatterns)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}
