// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/cucumber/godog"
	cucumbermessages "github.com/cucumber/messages/go/v21"
	"github.com/stretchr/testify/require"

	spec "github.com/elastic/package-spec/v3"
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
		version, _ := version.SetPrerelease("")
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

// checkFeaturesVersions checks that all features and scenarios contain only valid version tags.
func checkFeaturesVersions(t *testing.T, fs fs.FS, paths []string) {
	suite := godog.TestSuite{
		Options: &godog.Options{
			Paths:    paths,
			FS:       fs,
			TestingT: t,
			Strict:   false,
		},
	}

	features, err := suite.RetrieveFeatures()
	require.NoError(t, err)

	versionsInChangelog, err := spec.VersionsInChangelog()
	require.NoError(t, err)

	var versions []string
	for _, v := range versionsInChangelog {
		v, _ := v.SetPrerelease("")
		versions = append(versions, v.String())
	}

	validTags := func(tags []*cucumbermessages.Tag, requireTags bool) error {
		if requireTags && len(tags) == 0 {
			return fmt.Errorf("no version tags")
		}
		for _, tag := range tags {
			if !slices.Contains(versions, strings.TrimLeft(tag.Name, "@")) {
				return fmt.Errorf("tag indicates an unknown spec version %s", tag.Name)
			}
		}
		return nil
	}

	for _, feature := range features {
		if err := validTags(feature.Feature.Tags, false); err != nil {
			t.Fatalf("Feature %q has an invalid tag: %s", feature.Feature.Name, err)
		}

		for _, child := range feature.Feature.Children {
			if err := validTags(child.Scenario.Tags, true); err != nil {
				t.Fatalf("Scenario %q in feature %q has an invalid tag: %s", child.Scenario.Name, feature.Feature.Name, err)
			}
		}
	}
}
