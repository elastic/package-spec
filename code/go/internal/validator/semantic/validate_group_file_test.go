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

const validGroupFile = `title: Nginx
description: Collect logs and metrics from Nginx servers.
icon: /img/logo_group.svg
`

func TestValidateGroupFileRequiresGroupField(t *testing.T) {
	cases := map[string]struct {
		// files written at the root of the package, by name.
		files map[string]string
		// expectedErrContains is empty when the package is expected to validate.
		expectedErrContains string
	}{
		"group_file_and_group_field": {
			files: map[string]string{
				"group.yml":    validGroupFile,
				"manifest.yml": "type: integration\ngroup: nginx\n",
			},
		},
		"group_file_without_group_field": {
			files: map[string]string{
				"group.yml":    validGroupFile,
				"manifest.yml": "type: integration\n",
			},
			expectedErrContains: `package contains a "group.yml" file but does not define the group it belongs to in the manifest`,
		},
		"group_file_with_empty_group_field": {
			files: map[string]string{
				"group.yml":    validGroupFile,
				"manifest.yml": "type: integration\ngroup: \"\"\n",
			},
			expectedErrContains: `does not define the group it belongs to in the manifest`,
		},
		// The reverse is allowed: packages that are not in a group directory join a
		// group by setting the field alone.
		"group_field_without_group_file": {
			files: map[string]string{
				"manifest.yml": "type: integration\ngroup: nginx\n",
			},
		},
		// Packages that have nothing to do with groups are not touched by the rule.
		"neither_group_file_nor_group_field": {
			files: map[string]string{
				"manifest.yml": "type: integration\n",
			},
		},
		"group_file_without_manifest": {
			files: map[string]string{
				"group.yml": validGroupFile,
			},
			expectedErrContains: "manifest.yml",
		},
		"group_file_with_malformed_manifest": {
			files: map[string]string{
				"group.yml":    validGroupFile,
				"manifest.yml": "type: integration\ngroup: [\n",
			},
			expectedErrContains: "failed to parse manifest",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			d := t.TempDir()
			for fileName, contents := range testCase.files {
				require.NoError(t, os.WriteFile(filepath.Join(d, fileName), []byte(contents), 0o644))
			}

			errs := ValidateGroupFileRequiresGroupField(fspath.DirFS(d))

			if testCase.expectedErrContains == "" {
				require.Empty(t, errs, "expected no validation errors")
				return
			}
			require.NotEmpty(t, errs, "expected validation errors")
			assert.Len(t, errs, 1)
			assert.ErrorContains(t, errs, testCase.expectedErrContains)
		})
	}
}
