// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spec

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

//go:embed spec spec/integration/_dev spec/integration/data_stream/_dev spec/input/_dev spec/content/_dev docs_schema
var content embed.FS

// FS returns an io/fs.FS for accessing the "package-spec/spec" contents.
func FS() fs.FS {
	fs, err := fs.Sub(content, "spec")
	if err != nil {
		panic(err)
	}
	return fs
}

func DocsFS() fs.FS {
	fs, err := fs.Sub(content, "docs_schema")
	if err != nil {
		panic(err)
	}
	return fs
}

// CheckVersion checks if the given version is implemented by current spec. It returns
// the version of the spec matching with the given version.
func CheckVersion(version semver.Version) (*semver.Version, error) {
	versions, err := VersionsInChangelog()
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		// Ignore prereleases for comparison.
		if cmpVersion, err := v.SetPrerelease(""); err != nil {
			// This should never happen when setting an empty prerelease.
			panic(err)
		} else if cmpVersion.Equal(&version) {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("spec version %q not found", version.String())
}

// VersionsInChangelog returns the list of versions defined in the changelog file.
func VersionsInChangelog() ([]semver.Version, error) {
	d, err := fs.ReadFile(content, "spec/changelog.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to read spec changelog: %w", err)
	}

	var entries []struct {
		Version string `yaml:"version"`
	}
	err = yaml.Unmarshal(d, &entries)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec changelog: %w", err)
	}

	versions := make([]semver.Version, len(entries))
	for i, entry := range entries {
		semverVersion, err := semver.NewVersion(entry.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to parse version (%s) in spec changelog: %w", entry.Version, err)
		}
		versions[i] = *semverVersion
	}

	return versions, nil
}
