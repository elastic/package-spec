// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateVersionIntegrity returns validation errors if the version defined in manifest isn't referenced in the latest
// entry of the changelog file.
func ValidateVersionIntegrity(fsys fspath.FS) specerrors.ValidationErrors {
	manifestVersion, err := readManifestVersion(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	changelogVersions, err := readChangelogVersions(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	err = ensureUniqueVersions(changelogVersions)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	err = ensureManifestVersionHasChangelogEntry(manifestVersion, changelogVersions)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	err = ensureChangelogLatestVersionIsGreaterThanOthers(changelogVersions)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}
	return nil
}

func readManifestVersion(fsys fspath.FS) (string, error) {
	manifestPath := "manifest.yml"
	f, err := pkgpath.Files(fsys, manifestPath)
	if err != nil {
		return "", fmt.Errorf("can't locate manifest file: %w", err)
	}

	if len(f) != 1 {
		return "", errors.New("single manifest file expected")
	}

	val, err := f[0].Values("$.version")
	if err != nil {
		return "", fmt.Errorf("can't read manifest version: %w", err)
	}

	sVal, ok := val.(string)
	if !ok {
		return "", errors.New("version is undefined")
	}
	return sVal, nil
}

func readChangelogVersions(fsys fspath.FS) ([]string, error) {
	return readChangelog(fsys, "$[*].version")
}

func readChangelog(fsys fspath.FS, jsonpath string) ([]string, error) {
	changelogPath := "changelog.yml"
	f, err := pkgpath.Files(fsys, changelogPath)
	if err != nil {
		return nil, fmt.Errorf("can't locate changelog file: %w", err)
	}

	if len(f) != 1 {
		return nil, errors.New("single changelog file expected")
	}

	vals, err := f[0].Values(jsonpath)
	if err != nil {
		return nil, fmt.Errorf("can't changelog entries: %w", err)
	}

	versions, err := toStringSlice(vals)
	if err != nil {
		return nil, fmt.Errorf("can't convert slice entries: %w", err)
	}
	return versions, nil
}

func toStringSlice(val interface{}) ([]string, error) {
	vals, ok := val.([]interface{})
	if !ok {
		return nil, errors.New("conversion error")
	}

	var s []string
	for _, v := range vals {
		str, ok := v.(string)
		if !ok {
			return nil, errors.New("conversion error")
		}
		s = append(s, str)
	}
	return s, nil
}

func ensureUniqueVersions(versions []string) error {
	m := map[string]struct{}{}
	for _, v := range versions {
		if _, ok := m[v]; ok {
			return fmt.Errorf("versions in changelog must be unique, found at least two same versions (%s)", v)
		}
		m[v] = struct{}{}
	}
	return nil
}

func ensureManifestVersionHasChangelogEntry(manifestVersion string, versions []string) error {
	if len(versions) == 0 {
		return errors.New("no versions found in changelog")
	}

	if manifestVersion == versions[0] {
		return nil
	}

	for _, v := range versions {
		// It's allowed to keep additional record with "-next" suffix for changes that will be released in the future.
		if v == manifestVersion && strings.HasSuffix(versions[0], "-next") {
			return nil
		}
	}
	return errors.New("current manifest version doesn't have changelog entry")
}

func ensureChangelogLatestVersionIsGreaterThanOthers(versions []string) error {
	if len(versions) == 0 {
		return errors.New("no versions found in changelog")
	}

	latestVersion, err := semver.NewVersion(versions[0])
	if err != nil {
		return fmt.Errorf("could not read package manifest version [%s]: %w", versions[0], err)
	}

	for _, v := range versions[1:] {
		changelogVersion, err := semver.NewVersion(v)
		if err != nil {
			return fmt.Errorf("could not read package manifest version [%s]: %w", changelogVersion, err)
		}
		if changelogVersion.GreaterThanEqual(latestVersion) {
			return fmt.Errorf("changelog entry %s is greater than or equal to first changelog entry: %s", changelogVersion, latestVersion)
		}
	}
	return nil
}
