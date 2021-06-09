// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"path/filepath"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/pkgpath"
	"github.com/pkg/errors"
)

// ValidateVersionIntegrity returns validation errors if the version defined in manifest isn't referenced in the latest
// entry of the changelog file.
func ValidateVersionIntegrity(pkgRoot string) ve.ValidationErrors {
	manifestVersion, err := readManifestVersion(pkgRoot)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	changelogVersions, err := readChangelogVersions(pkgRoot)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	for _, v := range changelogVersions {
		if v == manifestVersion {
			return nil
		}
	}
	return ve.ValidationErrors{errors.New("current manifest version doesn't have changelog entry")}
}

func readManifestVersion(pkgRoot string) (string, error) {
	manifestPath := filepath.Join(pkgRoot, "manifest.yml")
	f, err := pkgpath.Files(manifestPath)
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

func readChangelogVersions(pkgRoot string) ([]string, error) {
	manifestPath := filepath.Join(pkgRoot, "changelog.yml")
	f, err := pkgpath.Files(manifestPath)
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