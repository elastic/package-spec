// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	spec "github.com/elastic/package-spec/v2"
)

const specVersionEnv = "TEST_SPEC_VERSION"

func versionToComply(t *testing.T) semver.Version {
	t.Helper()

	v := os.Getenv(specVersionEnv)
	if v == "" {
		t.Fatalf("%s environment variable required with the version to test compliance", specVersionEnv)
	}

	version, err := semver.NewVersion(v)
	require.NoError(t, err)

	return *version
}

func versionsToTest(t *testing.T) string {
	t.Helper()

	maxVersion := versionToComply(t)

	versions, err := spec.VersionsInChangelog()
	require.NoError(t, err)

	var result strings.Builder
	for _, version := range versions {
		if version.GreaterThan(&maxVersion) {
			continue
		}

		if result.Len() != 0 {
			result.WriteString(",")
		}

		result.WriteString(version.String())
	}

	return result.String()
}
