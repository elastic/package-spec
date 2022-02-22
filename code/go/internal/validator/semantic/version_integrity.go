// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/elastic/package-spec/code/go/internal/pkgpath"
)

// ValidateVersionIntegrity returns validation errors if the version defined in manifest isn't referenced in the latest
// entry of the changelog file.
func ValidateVersionIntegrity(fsys fspath.FS) ve.ValidationErrors {
	manifestVersion, err := readManifestVersion(fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	changelogVersions, err := readChangelogVersions(fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	err = ensureUniqueVersions(changelogVersions)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	err = ensureManifestVersionHasChangelogEntry(manifestVersion, changelogVersions)
	if err != nil {
		return ve.ValidationErrors{err}
	}
	return nil
}

func readManifestVersion(fsys fspath.FS) (string, error) {
	manifestPath := "manifest.yml"
	f, err := pkgpath.Files(fsys, manifestPath)
	if err != nil {
		return "", errors.Wrap(err, "can't locate manifest file")
	}

	if len(f) != 1 {
		return "", errors.New("single manifest file expected")
	}

	val, err := f[0].Values("$.version")
	if err != nil {
		return "", errors.Wrap(err, "can't read manifest version")
	}

	sVal, ok := val.(string)
	if !ok {
		return "", errors.New("version is undefined")
	}
	return sVal, nil
}

func readChangelogVersions(fsys fspath.FS) ([]string, error) {
	changelogPath := "changelog.yml"
	f, err := pkgpath.Files(fsys, changelogPath)
	if err != nil {
		return nil, errors.Wrap(err, "can't locate changelog file")
	}

	if len(f) != 1 {
		return nil, errors.New("single changelog file expected")
	}

	vals, err := f[0].Values("$[*].version")
	if err != nil {
		return nil, errors.Wrap(err, "can't changelog entries")
	}

	versions, err := toStringSlice(vals)
	if err != nil {
		return nil, errors.Wrap(err, "can't convert slice entries")
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
