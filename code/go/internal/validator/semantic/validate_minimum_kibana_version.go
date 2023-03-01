// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
)

// ValidateMinimumKibanaVersion if the package is an input package, and the package version is >= 1.0.0,
// then the kibana version condition must be >= 8.8.0
func ValidateMinimumKibanaVersion(fsys fspath.FS) ve.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	packageType, err := getPackageType(*manifest)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	packageVersion, err := getPackageVersion(*manifest)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	kibanaVersionCondition, err := getKibanaVersionCondition(*manifest)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	err = validateMinimumKibanaVersion(packageType, *packageVersion, kibanaVersionCondition)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	return nil
}

func validateMinimumKibanaVersion(packageType string, packageVersion semver.Version, kibanaVersionCondition string) error {

	if packageType != "input" {
		return nil
	}

	if packageVersion.LessThan(semver.MustParse("1.0.0")) {
		return nil
	}

	if kibanaVersionConditionIsGreaterThanOrEqualTo8_8_0(kibanaVersionCondition) {
		return nil
	}

	return errors.New("Warning: conditions.kibana.version must be ^8.8.0 or greater for non experimental input packages (version > 1.0.0)")
}

func readManifest(fsys fspath.FS) (*pkgpath.File, error) {
	manifestPath := "manifest.yml"
	f, err := pkgpath.Files(fsys, manifestPath)
	if err != nil {
		return nil, errors.Wrap(err, "can't locate manifest file")
	}

	if len(f) != 1 {
		return nil, errors.New("single manifest file expected")
	}

	return &f[0], nil
}

func getPackageType(manifest pkgpath.File) (string, error) {

	val, err := manifest.Values("$.type")
	if err != nil {
		return "", errors.Wrap(err, "can't read manifest type")
	}

	sVal, ok := val.(string)
	if !ok {
		return "", errors.New("manifest type is not a string")
	}

	return sVal, nil
}

func getPackageVersion(manifest pkgpath.File) (*semver.Version, error) {

	val, err := manifest.Values("$.version")
	if err != nil {
		return nil, errors.Wrap(err, "can't read manifest version")
	}

	sVal, ok := val.(string)
	if !ok {
		return nil, errors.New("package version is not a string")
	}

	sVersion, err := semver.NewVersion(sVal)
	return sVersion, nil
}

func getKibanaVersionCondition(manifest pkgpath.File) (string, error) {

	val, err := manifest.Values("$.conditions[\"kibana.version\"]")
	if err != nil {
		return "", nil
	}

	sVal, ok := val.(string)
	if !ok {
		return "", errors.New("manifest kibana version is not a string")
	}

	return sVal, nil
}

func kibanaVersionConditionIsGreaterThanOrEqualTo8_8_0(kibanaVersionCondition string) bool {
	if kibanaVersionCondition == "" {
		return false
	}

	if kibanaVersionCondition == "^8.8.0" {
		return true
	}

	// get all versions e.g 8.8.0, 8.8.1 from "^8.8.0 || ^8.8.1" and check if any of them is less than 8.8.0
	pattern := `(\d+\.\d+\.\d+)`
	semver8_8_0 := semver.MustParse("8.8.0")
	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllString(kibanaVersionCondition, -1)

	for _, match := range matches {
		matchVersion, err := semver.NewVersion(match)
		if err != nil {
			return false
		}

		if matchVersion.LessThan(semver8_8_0) {
			return false
		}
	}

	return true
}
