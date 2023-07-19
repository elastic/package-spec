// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/packages"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
)

// ValidateMinimumKibanaVersion ensures the minimum kibana version for a given package is the expected one
func ValidateMinimumKibanaVersion(fsys fspath.FS) ve.ValidationErrors {
	pkg, err := packages.NewPackageFromFS(fsys.Path(), fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	manifest, err := readManifest(fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	kibanaVersionCondition, err := getKibanaVersionCondition(*manifest)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	var errs ve.ValidationErrors
	err = validateMinimumKibanaVersionInputPackages(pkg.Type, *pkg.Version, kibanaVersionCondition)
	if err != nil {
		errs.Append(ve.ValidationErrors{err})
	}

	err = validateMinimumKibanaVersionRuntimeFields(fsys, *pkg.Version, kibanaVersionCondition)
	if err != nil {
		errs.Append(ve.ValidationErrors{err})
	}

	err = validateMinimumKibanaVersionSavedObjectTags(fsys, pkg.Type, *pkg.Version, kibanaVersionCondition)
	if err != nil {
		errs.Append(ve.ValidationErrors{err})
	}

	if errs != nil {
		return errs
	}

	return nil
}

// validateMinimumKibanaVersionInputPackages ensures the minimum kibana version if the package is an input package, and the package version is >= 1.0.0,
// then the kibana version condition for the package must be >= 8.8.0
func validateMinimumKibanaVersionInputPackages(packageType string, packageVersion semver.Version, kibanaVersionCondition string) error {
	const minimumKibanaVersion = "8.8.0"
	if packageType != "input" {
		return nil
	}

	if packageVersion.LessThan(semver.MustParse("1.0.0")) {
		return nil
	}

	if kibanaVersionConditionIsGreaterThanOrEqualTo(kibanaVersionCondition, minimumKibanaVersion) {
		return nil
	}

	return fmt.Errorf("conditions.kibana.version must be ^%s or greater for non experimental input packages (version > 1.0.0)", minimumKibanaVersion)
}

// validateMinimumKibanaVersionRuntimeFields ensures the minimum kibana version if the package defines any runtime field,
// then the kibana version condition for the package must be >= 8.10.0
func validateMinimumKibanaVersionRuntimeFields(fsys fspath.FS, packageVersion semver.Version, kibanaVersionCondition string) error {
	const minimumKibanaVersion = "8.10.0"
	errs := validateFields(fsys, validateNoRuntimeFields)
	if len(errs) == 0 {
		return nil
	}

	if kibanaVersionConditionIsGreaterThanOrEqualTo(kibanaVersionCondition, minimumKibanaVersion) {
		return nil
	}

	return fmt.Errorf("conditions.kibana.version must be ^%s or greater to include runtime fields", minimumKibanaVersion)
}

// validateMinimumKibanaVersionSavedObjectTags ensures the minimum kibana version if the package defines saved object tags file,
// then the kibana version condition for the package must be >= 8.10.0
func validateMinimumKibanaVersionSavedObjectTags(fsys fspath.FS, packageType string, packageVersion semver.Version, kibanaVersionCondition string) error {
	const minimumKibanaVersion = "8.10.0"
	if packageType == "input" {
		return nil
	}

	manifestPath := "kibana/tags.yml"
	f, err := pkgpath.Files(fsys, manifestPath)
	if err != nil {
		return fmt.Errorf("can't locate files with %v: %w", manifestPath, err)
	}

	if len(f) == 0 {
		return nil
	}

	if kibanaVersionConditionIsGreaterThanOrEqualTo(kibanaVersionCondition, minimumKibanaVersion) {
		return nil
	}

	return fmt.Errorf("conditions.kibana.version must be ^%s or greater to include saved object tags file: %s", minimumKibanaVersion, manifestPath)
}

func readManifest(fsys fspath.FS) (*pkgpath.File, error) {
	manifestPath := "manifest.yml"
	f, err := pkgpath.Files(fsys, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("can't locate manifest file: %w", err)
	}

	if len(f) != 1 {
		return nil, fmt.Errorf("single manifest file expected")
	}

	return &f[0], nil
}

func readSavedObjectTags(fsys fspath.FS) (*pkgpath.File, error) {
	manifestPath := "kibana/tags.yml"
	f, err := pkgpath.Files(fsys, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("can't locate kibana/tags file: %w", err)
	}

	if len(f) != 1 {
		return nil, fmt.Errorf("single kibana/tags file expected")
	}

	return &f[0], nil
}

func getKibanaVersionCondition(manifest pkgpath.File) (string, error) {

	val, err := manifest.Values("$.conditions[\"kibana.version\"]")
	if err != nil {
		val, err = manifest.Values("$.conditions.kibana.version")
		if err != nil {
			return "", nil
		}
	}

	sVal, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("manifest kibana version is not a string")
	}

	return sVal, nil
}

func kibanaVersionConditionIsGreaterThanOrEqualTo(kibanaVersionCondition, minimumVersion string) bool {
	if kibanaVersionCondition == "" {
		return false
	}

	if kibanaVersionCondition == fmt.Sprintf("^%s", minimumVersion) {
		return true
	}

	// get all versions e.g 8.8.0, 8.8.1 from "^8.8.0 || ^8.8.1" and check if any of them is less than 8.8.0
	pattern := `(\d+\.\d+\.\d+)`
	minSemver := semver.MustParse(minimumVersion)
	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllString(kibanaVersionCondition, -1)

	for _, match := range matches {
		matchVersion, err := semver.NewVersion(match)
		if err != nil {
			return false
		}

		if matchVersion.LessThan(minSemver) {
			return false
		}
	}

	return true
}

func validateNoRuntimeFields(metadata fieldFileMetadata, f field) ve.ValidationErrors {
	if f.Runtime.isEnabled() {
		return ve.ValidationErrors{fmt.Errorf("%v file contains a field %s with runtime key defined (%s)", metadata.fullFilePath, f.Name, f.Runtime)}
	}
	return nil
}
