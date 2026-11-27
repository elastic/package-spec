// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"io/fs"
	"slices"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	spec "github.com/elastic/package-spec/v3"
	"github.com/elastic/package-spec/v3/code/go/internal/loader"
)

// packageTypes are the folder specs that describe a package type, and that a
// package may therefore declare in the type field of its manifest.
var packageTypes = []string{"content", "input", "integration"}

// nonPackageFolderSpecs are the folder specs bundled here that are not package
// types. They are loaded explicitly, never through a package manifest.
var nonPackageFolderSpecs = []string{"group"}

func TestLoadAllBundledVersions(t *testing.T) {
	versions, err := spec.VersionsInChangelog()
	require.NoError(t, err)

	for _, version := range versions {
		for _, pkgType := range packageTypes {
			testForVersionType(t, version, pkgType)
		}
	}
}

// TestEveryBundledFolderSpecIsClassified fails when a folder spec is added under
// spec/ without deciding whether it is a package type. Specs are resolved by
// folder name, so an unclassified folder silently becomes loadable as a package
// type.
func TestEveryBundledFolderSpecIsClassified(t *testing.T) {
	entries, err := fs.ReadDir(spec.FS(), ".")
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if slices.Contains(packageTypes, name) || slices.Contains(nonPackageFolderSpecs, name) {
			continue
		}
		t.Errorf("spec folder %q is neither a package type nor a declared non-package folder spec: "+
			"add it to packageTypes or nonPackageFolderSpecs", name)
	}
}

// TestLoadGroupFolderSpec checks the group folder spec loads, as it is not
// covered by TestLoadAllBundledVersions: a group has no format_version and is
// always validated against the latest spec version.
func TestLoadGroupFolderSpec(t *testing.T) {
	versions, err := spec.VersionsInChangelog()
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	latest := slices.MaxFunc(versions, func(a, b semver.Version) int { return a.Compare(&b) })
	released, err := latest.SetPrerelease("")
	require.NoError(t, err)

	_, err = loader.LoadSpec(spec.FS(), released, "group")
	require.NoError(t, err)
}

func testForVersionType(t *testing.T, version semver.Version, pkgType string) {
	t.Run(version.String(), func(t *testing.T) {
		t.Run(pkgType, func(t *testing.T) {
			fs := spec.FS()
			_, err := loader.LoadSpec(fs, version, pkgType)
			require.NoError(t, err)
		})
	})
}
