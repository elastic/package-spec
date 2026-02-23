// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func TestValidateTestPackageRequirements(t *testing.T) {
	tests := map[string]struct {
		manifest       string
		testConfig     string
		testConfigPath string
		expectError    bool
		errorContains  string
	}{
		"valid_integration_test_requirement": {
			manifest: `name: test
format_version: 3.6.0
requires:
  input:
    - name: sql_input
      version: ^1.2.0`,
			testConfig: `system:
  requires:
    - package: sql_input
      version: 1.2.5`,
			testConfigPath: "_dev/test/config.yml",
			expectError:    false,
		},
		"valid_datastream_test_requirement": {
			manifest: `name: test
format_version: 3.6.0
requires:
  content:
    - name: logs_package
      version: ~1.0.0`,
			testConfig: `requires:
  - package: logs_package
    version: 1.0.3`,
			testConfigPath: "data_stream/example/_dev/test/system/config.yml",
			expectError:    false,
		},
		"package_not_in_manifest": {
			manifest: `name: test
format_version: 3.6.0
requires:
  input:
    - name: sql_input
      version: ^1.2.0`,
			testConfig: `policy:
  requires:
    - package: missing_package
      version: 1.0.0`,
			testConfigPath: "_dev/test/config.yml",
			expectError:    true,
			errorContains:  "missing_package\" which is not listed in manifest requires",
		},
		"version_does_not_satisfy_constraint": {
			manifest: `name: test
format_version: 3.6.0
requires:
  input:
    - name: sql_input
      version: ^2.0.0`,
			testConfig: `system:
  requires:
    - package: sql_input
      version: 1.5.0`,
			testConfigPath: "_dev/test/config.yml",
			expectError:    true,
			errorContains:  "version \"1.5.0\" does not satisfy constraint \"^2.0.0\"",
		},
		"invalid_test_version": {
			manifest: `name: test
format_version: 3.6.0
requires:
  input:
    - name: sql_input
      version: ^1.0.0`,
			testConfig: `system:
  requires:
    - package: sql_input
      version: invalid`,
			testConfigPath: "_dev/test/config.yml",
			expectError:    true,
			errorContains:  "invalid version",
		},
		"multiple_test_types_with_requirements": {
			manifest: `name: test
format_version: 3.6.0
requires:
  input:
    - name: pkg1
      version: ^1.0.0
  content:
    - name: pkg2
      version: ~2.0.0`,
			testConfig: `system:
  requires:
    - package: pkg1
      version: 1.5.0
policy:
  requires:
    - package: pkg2
      version: 2.0.5`,
			testConfigPath: "_dev/test/config.yml",
			expectError:    false,
		},
		"no_requires_in_test": {
			manifest: `name: test
format_version: 3.6.0
requires:
  input:
    - name: sql_input
      version: ^1.0.0`,
			testConfig: `system:
  skip:
    reason: Test reason
    link: https://example.com`,
			testConfigPath: "_dev/test/config.yml",
			expectError:    false,
		},
		"no_requires_in_manifest": {
			manifest: `name: test
format_version: 3.6.0`,
			testConfig: `system:
  requires:
    - package: sql_input
      version: 1.0.0`,
			testConfigPath: "_dev/test/config.yml",
			expectError:    true,
			errorContains:  "sql_input\" which is not listed in manifest requires",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pkgRoot := t.TempDir()

			// Create manifest
			err := os.WriteFile(filepath.Join(pkgRoot, "manifest.yml"), []byte(tc.manifest), 0644)
			require.NoError(t, err)

			// Create test config in appropriate path
			testConfigFullPath := filepath.Join(pkgRoot, tc.testConfigPath)
			err = os.MkdirAll(filepath.Dir(testConfigFullPath), 0755)
			require.NoError(t, err)
			err = os.WriteFile(testConfigFullPath, []byte(tc.testConfig), 0644)
			require.NoError(t, err)

			fsys := fspath.DirFS(pkgRoot)
			errs := ValidateTestPackageRequirements(fsys)

			if tc.expectError {
				require.NotEmpty(t, errs, "expected validation errors but got none")
				assert.Contains(t, errs[0].Error(), tc.errorContains)
			} else {
				assert.Empty(t, errs, "expected no validation errors but got: %v", errs)
			}
		})
	}
}
