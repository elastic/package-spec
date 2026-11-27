// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateGroupDirectory covers the group folder spec: the group.yml file, the
// img folder, and how far member packages are checked.
func TestValidateGroupDirectory(t *testing.T) {
	basePath := filepath.Join("..", "..", "..", "..", "test", "groups")
	tests := map[string]struct {
		expectedErrContains []string
	}{
		// A group with one member of each type that can belong to a group.
		"good_group": {},
		"bad_group_missing_group_file": {
			expectedErrContains: []string{"expecting to find [group.yml] file in folder"},
		},
		"bad_group_img": {
			expectedErrContains: []string{
				"item [notes.txt] is not allowed in folder",
				// Images have to sit directly in img, as they are copied from there
				// into each member package at build time.
				"item [screenshots] is not allowed in folder",
			},
		},
		"bad_group_member": {
			expectedErrContains: []string{
				// A directory that is not a package at all.
				"expecting to find [manifest.yml] file in folder",
				// A package of a type that cannot belong to a group.
				`field type: type must be one of the following: "integration", "input", "content"`,
			},
		},
		"bad_group_extra_content": {
			expectedErrContains: []string{"item [README.md] is not allowed in folder"},
		},
		// Paths rejected before any spec is loaded.
		"does_not_exist": {
			expectedErrContains: []string{"no group found at path"},
		},
		filepath.Join("good_group", "group.yml"): {
			expectedErrContains: []string{"no group folder found at path"},
		},
	}

	for groupName, testCase := range tests {
		t.Run(groupName, func(t *testing.T) {
			t.Parallel()
			err := ValidateGroupDirectory(filepath.Join(basePath, groupName))
			if len(testCase.expectedErrContains) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, expectedError := range testCase.expectedErrContains {
				require.ErrorContains(t, err, expectedError)
			}
		})
	}
}

// TestPackageTypeGroupNotSupported checks that a package cannot be validated
// against the group folder spec by declaring it as its type. The spec is looked up
// by folder name, so adding spec/group would otherwise make "group" a package type.
func TestPackageTypeGroupNotSupported(t *testing.T) {
	pkgPath := filepath.Join("..", "..", "..", "..", "test", "packages", "bad_group_package_type")
	for _, mode := range []Mode{BuildMode, SourceMode, LegacyMode} {
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()
			v, err := New(mode)
			require.NoError(t, err)
			err = v.ValidateFromPath(pkgPath)
			require.ErrorContains(t, err, `package type "group" not supported`)
		})
	}
}
