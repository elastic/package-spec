// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

func TestValidateDocsStructureContent(t *testing.T) {
	tests := []struct {
		title            string
		filename         string
		content          []byte
		enforcedSections []string
		expectError      bool
		expectedError    error
	}{
		{
			title:            "Valid test",
			filename:         "valid_test.md",
			content:          []byte("# Overview\n\n# Installation\n\n# Usage\n"),
			enforcedSections: []string{},
			expectError:      false,
			expectedError:    nil,
		},
		{
			title:            "Missing section",
			filename:         "missing_section.md",
			content:          []byte("# Overview\n\n# Usage\n"),
			enforcedSections: []string{"Installation"},
			expectError:      true,
			expectedError:    fmt.Errorf("missing required section 'Installation' in file 'missing_section.md'"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			actualResult := validateDocsStructureContent(tt.content, tt.enforcedSections, tt.filename)
			assert.True(t, compareErrors(tt.expectError, tt.expectedError, actualResult), "Error does not match expected")
		})
	}
}

func TestReadValidateDocsStructureConfig(t *testing.T) {
	tests := []struct {
		name           string
		pkgRoot        string
		config         *specerrors.DocsStructureEnforced
		expectedResult []string
		expectedError  error
	}{

		{
			name:    "Valid test",
			pkgRoot: "../../../../../test/packages/good_readme_structure",
			config: &specerrors.DocsStructureEnforced{
				Enabled: true,
				Version: 1,
				DocsStructureSkip: []specerrors.DocsStructureSkip{
					{Title: "Overview", Reason: "Not applicable for this integration"},
				},
			},
			expectedResult: []string{"What data does this integration collect?", "What do I need to use this integration?", "How do I deploy this integration?", "Troubleshooting", "Performance and scaling"},
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResult, err := readDocsStructureEnforcedConfig(fspath.DirFS(tt.pkgRoot), tt.config)
			assert.Equal(t, tt.expectedResult, actualResult, "Result does not match expected")
			assert.Equal(t, tt.expectedError, err, "Error does not match expected")
		})
	}
}

func TestValidateDocsStructure(t *testing.T) {
	tests := []struct {
		name          string
		pkgRoot       string
		expectError   bool
		expectedError error
	}{
		{
			name:          "Valid test",
			pkgRoot:       "../../../../../test/packages/good_readme_structure",
			expectError:   false,
			expectedError: nil,
		},
		{
			name:        "Invalid test",
			pkgRoot:     "../../../../../test/packages/bad_readme_structure",
			expectError: true,
			expectedError: specerrors.ValidationErrors{
				specerrors.NewStructuredError(
					fmt.Errorf("missing required section 'Overview' in file 'README_part1.md'\nmissing required section 'How do I deploy this integration?' in file 'README_part2.md'"), specerrors.UnassignedCode)},
		},
		{
			name:          "No validation test",
			pkgRoot:       "../../../../../test/packages/good_v3",
			expectError:   false,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDocsStructure(fspath.DirFS(tt.pkgRoot))
			assert.True(t, compareErrors(tt.expectError, tt.expectedError, err), "Error does not match expected")
		})
	}
}

func TestLoadSectionsFromConfig(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		expectedResult []string
		expectError    bool
		expectedError  error
	}{
		{
			name:           "Valid version",
			version:        "1",
			expectedResult: []string{"Overview", "What data does this integration collect?", "What do I need to use this integration?", "How do I deploy this integration?", "Troubleshooting", "Performance and scaling"},
			expectError:    false,
			expectedError:  nil,
		},
		{
			name:           "Invalid version",
			version:        "9999",
			expectedResult: nil,
			expectError:    true,
			expectedError:  fmt.Errorf("failed to read schema file: open enforced_sections_v9999.yml: file does not exist"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResult, err := loadSectionsFromConfig(tt.version)
			assert.Equal(t, tt.expectedResult, actualResult, "Result does not match expected")
			assert.True(t, compareErrors(tt.expectError, tt.expectedError, err), "Error does not match expected")
		})
	}
}

// compareErrors checks if the actual error matches the expected error based on the expectation flag.
func compareErrors(expectError bool, expectedError error, actualError error) bool {
	if !expectError {
		return true
	}
	if expectedError == nil && actualError == nil {
		return true
	}
	if expectedError != nil && actualError != nil {
		return expectedError.Error() == actualError.Error()
	}
	return false
}
