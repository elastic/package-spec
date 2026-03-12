// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateTestPackageRequirements checks that package requirements in test configurations
// reference packages listed in the manifest and that versions satisfy constraints.
func ValidateTestPackageRequirements(fsys PackageFS) specerrors.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path("manifest.yml"), err)}
	}

	// Build map of required packages with their version constraints
	requiredPackages, err := getRequiredPackagesWithConstraints(*manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path("manifest.yml"), err)}
	}

	var errs specerrors.ValidationErrors

	// Validate integration-level test configs
	integrationTestErrs := validateIntegrationTestRequirements(fsys, requiredPackages)
	errs = append(errs, integrationTestErrs...)

	// Validate data stream test configs
	dataStreamTestErrs := validateDataStreamTestRequirements(fsys, requiredPackages)
	errs = append(errs, dataStreamTestErrs...)

	return errs
}

func getRequiredPackagesWithConstraints(manifest pkgpath.File) (map[string]string, error) {
	requiredPackages := make(map[string]string)

	// Get input packages from requires.input
	inputPackages, err := manifest.Values("$.requires.input")
	if err == nil && inputPackages != nil {
		if pkgArray, ok := inputPackages.([]interface{}); ok {
			for i := 0; i < len(pkgArray); i++ {
				name, err := manifest.Values(fmt.Sprintf("$.requires.input[%d].package", i))
				if err != nil || name == nil {
					continue
				}
				version, err := manifest.Values(fmt.Sprintf("$.requires.input[%d].version", i))
				if err != nil || version == nil {
					continue
				}
				if nameStr, ok := name.(string); ok {
					if versionStr, ok := version.(string); ok {
						requiredPackages[nameStr] = versionStr
					}
				}
			}
		}
	}

	// Get content packages from requires.content
	contentPackages, err := manifest.Values("$.requires.content")
	if err == nil && contentPackages != nil {
		if pkgArray, ok := contentPackages.([]interface{}); ok {
			for i := 0; i < len(pkgArray); i++ {
				name, err := manifest.Values(fmt.Sprintf("$.requires.content[%d].package", i))
				if err != nil || name == nil {
					continue
				}
				version, err := manifest.Values(fmt.Sprintf("$.requires.content[%d].version", i))
				if err != nil || version == nil {
					continue
				}
				if nameStr, ok := name.(string); ok {
					if versionStr, ok := version.(string); ok {
						requiredPackages[nameStr] = versionStr
					}
				}
			}
		}
	}

	return requiredPackages, nil
}

func validateIntegrationTestRequirements(fsys PackageFS, requiredPackages map[string]string) specerrors.ValidationErrors {
	testConfig, err := fsys.Files("_dev/test/config.yml")
	if err != nil || len(testConfig) == 0 {
		return nil
	}

	var errs specerrors.ValidationErrors
	for _, config := range testConfig {
		// Check each test type (system, policy, etc.)
		testTypes := []string{"system", "policy", "pipeline", "static", "asset"}
		for _, testType := range testTypes {
			requires, err := config.Values(fmt.Sprintf("$.%s.requires", testType))
			if err != nil || requires == nil {
				continue
			}

			if reqArray, ok := requires.([]interface{}); ok {
				for idx, req := range reqArray {
					if reqMap, ok := req.(map[string]interface{}); ok {
						pkgName, _ := reqMap["package"].(string)
						version, _ := reqMap["version"].(string)
						if pkgName != "" && version != "" {
							err := validateTestRequirementPackageVersion(fsys.Path("_dev/test/config.yml"), testType, idx, pkgName, version, requiredPackages)
							if err != nil {
								errs = append(errs, err)
							}
							continue
						}

						source, _ := reqMap["source"].(string)
						if source != "" {
							err := validateTestRequirementSource(fsys.Path("_dev/test/config.yml"), source)
							if err != nil {
								errs = append(errs, err)
							}
							continue
						}
					}
				}
			}
		}
	}

	return errs
}

func validateDataStreamTestRequirements(fsys PackageFS, requiredPackages map[string]string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	patterns := []string{
		"data_stream/*/_dev/test/system/test-*-config.yml",
		"data_stream/*/_dev/test/policy/test-*.yml",
		"data_stream/*/_dev/test/static/test-*-config.yml",
	}

	for _, pattern := range patterns {
		testConfigs, err := fsys.Files(pattern)
		if err != nil {
			continue
		}

		for _, config := range testConfigs {
			requires, err := config.Values("$.requires")
			if err != nil || requires == nil {
				continue
			}

			if reqArray, ok := requires.([]interface{}); ok {
				for idx, req := range reqArray {
					if reqMap, ok := req.(map[string]interface{}); ok {
						pkgName, _ := reqMap["package"].(string)
						version, _ := reqMap["version"].(string)
						if pkgName != "" && version != "" {
							err := validateTestRequirementPackageVersion(fsys.Path(config.Path()), "", idx, pkgName, version, requiredPackages)
							if err != nil {
								errs = append(errs, err)
							}
							continue
						}

						source, _ := reqMap["source"].(string)
						if source != "" {
							err := validateTestRequirementSource(fsys.Path(config.Path()), source)
							if err != nil {
								errs = append(errs, err)
							}
							continue
						}
					}
				}
			}
		}
	}

	return errs
}

func validateTestRequirementPackageVersion(configPath, testType string, idx int, pkgName, version string, requiredPackages map[string]string) *specerrors.StructuredError {
	constraint, exists := requiredPackages[pkgName]
	if !exists {
		location := fmt.Sprintf("requires[%d]", idx)
		if testType != "" {
			location = fmt.Sprintf("%s.%s", testType, location)
		}
		return specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: %s references package \"%s\" which is not listed in manifest requires section",
			configPath, location, pkgName)
	}

	// Parse the test version
	testVersion, err := semver.NewVersion(version)
	if err != nil {
		location := fmt.Sprintf("requires[%d]", idx)
		if testType != "" {
			location = fmt.Sprintf("%s.%s", testType, location)
		}
		return specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: %s has invalid version \"%s\": %w",
			configPath, location, version, err)
	}

	// Parse the constraint from manifest
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		location := fmt.Sprintf("requires[%d]", idx)
		if testType != "" {
			location = fmt.Sprintf("%s.%s", testType, location)
		}
		return specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: %s package \"%s\" has invalid constraint in manifest: %w",
			configPath, location, pkgName, err)
	}

	// Check if test version satisfies constraint
	if !c.Check(testVersion) {
		location := fmt.Sprintf("requires[%d]", idx)
		if testType != "" {
			location = fmt.Sprintf("%s.%s", testType, location)
		}
		return specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: %s package \"%s\" version \"%s\" does not satisfy constraint \"%s\"",
			configPath, location, pkgName, version, constraint)
	}

	return nil
}

// validateTestRequirementSource checks if the relative path exists. This could be done with "format: relative-path" in
// the spec, but this format checker only works with relative files inside the package. In this case the source package
// is going to be outside the current package.
func validateTestRequirementSource(configFile, source string) *specerrors.StructuredError {
	cleanSource := filepath.Clean(filepath.FromSlash(source))
	if filepath.IsAbs(cleanSource) {
		return specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: source path to required package \"%s\" must be relative",
			configFile, source)
	}
	targetPath := filepath.Join(filepath.Dir(configFile), filepath.FromSlash(source))
	if _, err := os.Stat(targetPath); err != nil {
		return specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: source path to required package \"%s\" does not exist",
			configFile, source)
	}
	return nil
}
