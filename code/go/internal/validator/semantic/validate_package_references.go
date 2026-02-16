// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"slices"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidatePackageReferences checks that package references in policy templates and data streams
// are listed in the manifest's requires section and are of the correct type (input packages only).
func ValidatePackageReferences(fsys fspath.FS) specerrors.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path("manifest.yml"), err)}
	}

	// Build lists of required packages by type
	requiredPackages, err := getRequiredPackagesByType(*manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path("manifest.yml"), err)}
	}

	// Validate policy template input package references
	errs := validatePolicyTemplatePackageReferences(fsys, *manifest, requiredPackages)

	// Validate data stream stream package references
	dsErrs := validateDataStreamPackageReferences(fsys, requiredPackages)
	errs = append(errs, dsErrs...)

	return errs
}

// requiredPackages contains lists of required packages organized by type.
type requiredPackages struct {
	input   []string
	content []string
}

func getRequiredPackagesByType(manifest pkgpath.File) (requiredPackages, error) {
	packages := requiredPackages{
		input:   []string{},
		content: []string{},
	}

	// Get input packages from requires.input
	inputPackages, err := manifest.Values("$.requires.input")
	if err == nil && inputPackages != nil {
		extractPackageNamesFromRequires(inputPackages, &packages.input)
	}

	// Get content packages from requires.content
	contentPackages, err := manifest.Values("$.requires.content")
	if err == nil && contentPackages != nil {
		extractPackageNamesFromRequires(contentPackages, &packages.content)
	}

	return packages, nil
}

func extractPackageNamesFromRequires(packages interface{}, result *[]string) {
	pkgArray, ok := packages.([]interface{})
	if !ok {
		return
	}

	for _, pkg := range pkgArray {
		pkgMap, ok := pkg.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := pkgMap["name"].(string)
		if ok {
			*result = append(*result, name)
		}
	}
}

func validatePolicyTemplatePackageReferences(fsys fspath.FS, manifest pkgpath.File, requiredPackages requiredPackages) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	policyTemplates, err := manifest.Values("$.policy_templates")
	if err != nil || policyTemplates == nil {
		return nil
	}

	policyTemplateArray, ok := policyTemplates.([]interface{})
	if !ok {
		return nil
	}

	for templateIndex, template := range policyTemplateArray {
		templateMap, ok := template.(map[string]interface{})
		if !ok {
			continue
		}

		inputs, ok := templateMap["inputs"].([]interface{})
		if !ok {
			continue
		}

		for inputIndex, input := range inputs {
			inputMap, ok := input.(map[string]interface{})
			if !ok {
				continue
			}

			packageName, ok := inputMap["package"].(string)
			if !ok || packageName == "" {
				continue
			}

			// Check if it's a content package (not allowed)
			if slices.Contains(requiredPackages.content, packageName) {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy_templates[%d].inputs[%d] references package \"%s\" which is a content package, only input packages allowed",
					fsys.Path("manifest.yml"), templateIndex, inputIndex, packageName))
				continue
			}

			// Check if it's in required input packages
			if !slices.Contains(requiredPackages.input, packageName) {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy_templates[%d].inputs[%d] references package \"%s\" which is not listed in requires section",
					fsys.Path("manifest.yml"), templateIndex, inputIndex, packageName))
			}
		}
	}

	return errs
}

func validateDataStreamPackageReferences(fsys fspath.FS, requiredPackages requiredPackages) specerrors.ValidationErrors {
	dataStreamManifests, err := pkgpath.Files(fsys, "data_stream/*/manifest.yml")
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("error while searching for data stream manifests: %w", err)}
	}

	var errs specerrors.ValidationErrors
	for _, dataStreamManifest := range dataStreamManifests {
		streams, err := dataStreamManifest.Values("$.streams")
		if err != nil || streams == nil {
			continue
		}

		streamArray, ok := streams.([]interface{})
		if !ok {
			continue
		}

		for streamIndex, stream := range streamArray {
			streamMap, ok := stream.(map[string]interface{})
			if !ok {
				continue
			}

			packageName, ok := streamMap["package"].(string)
			if !ok || packageName == "" {
				continue
			}

			// Check if it's a content package (not allowed)
			if slices.Contains(requiredPackages.content, packageName) {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: streams[%d] references package \"%s\" which is a content package, only input packages allowed",
					dataStreamManifest.Path(), streamIndex, packageName))
				continue
			}

			// Check if it's in required input packages
			if !slices.Contains(requiredPackages.input, packageName) {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: streams[%d] references package \"%s\" which is not listed in manifest requires section",
					dataStreamManifest.Path(), streamIndex, packageName))
			}
		}
	}

	return errs
}
