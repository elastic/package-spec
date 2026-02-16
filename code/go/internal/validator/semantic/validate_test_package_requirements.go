// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateTestPackageRequirements checks that package requirements in test configurations
// reference packages listed in the manifest and that versions satisfy constraints.
func ValidateTestPackageRequirements(fsys fspath.FS) specerrors.ValidationErrors {
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
				name, err := manifest.Values(fmt.Sprintf("$.requires.input[%d].name", i))
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
				name, err := manifest.Values(fmt.Sprintf("$.requires.content[%d].name", i))
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

func validateIntegrationTestRequirements(fsys fspath.FS, requiredPackages map[string]string) specerrors.ValidationErrors {
	testConfig, err := pkgpath.Files(fsys, "_dev/test/config.yml")
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

						if pkgName == "" || version == "" {
							continue
						}

						err := validateTestRequirement(config.Path(), testType, idx, pkgName, version, requiredPackages)
						if err != nil {
							errs = append(errs, err)
						}
					}
				}
			}
		}
	}

	return errs
}

func validateDataStreamTestRequirements(fsys fspath.FS, requiredPackages map[string]string) specerrors.ValidationErrors {
	testConfigs, err := pkgpath.Files(fsys, "data_stream/*/_dev/test/*/config.yml")
	if err != nil {
		return nil
	}

	var errs specerrors.ValidationErrors
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

					if pkgName == "" || version == "" {
						continue
					}

					err := validateTestRequirement(config.Path(), "", idx, pkgName, version, requiredPackages)
					if err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}

	return errs
}

func validateTestRequirement(configPath, testType string, idx int, pkgName, version string, requiredPackages map[string]string) *specerrors.StructuredError {
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
