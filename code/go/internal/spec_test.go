// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	spec "github.com/elastic/package-spec/v2"
	"github.com/elastic/package-spec/v2/code/go/internal/loader"
)

func TestLoadAllBundledVersions(t *testing.T) {
	versions, err := spec.VersionsInChangelog()
	require.NoError(t, err)

	for _, version := range versions {
		testForVersionType(t, version, "input")
		testForVersionType(t, version, "integration")
	}
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
