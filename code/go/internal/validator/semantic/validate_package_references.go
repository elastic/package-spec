// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidatePackageReferences checks that package references in policy templates and data streams
// are listed in the manifest's requires section.
func ValidatePackageReferences(fsys fspath.FS) specerrors.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path("manifest.yml"), err)}
	}

	// Build map of required packages
	requiredPackages, err := getRequiredPackages(*manifest)
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

func getRequiredPackages(manifest pkgpath.File) (map[string]bool, error) {
	requiredPackages := make(map[string]bool)

	// Get input packages from requires.input
	inputPackages, err := manifest.Values("$.requires.input")
	if err == nil && inputPackages != nil {
		extractPackageNamesFromRequires(inputPackages, requiredPackages)
	}

	// Get content packages from requires.content
	contentPackages, err := manifest.Values("$.requires.content")
	if err == nil && contentPackages != nil {
		extractPackageNamesFromRequires(contentPackages, requiredPackages)
	}

	return requiredPackages, nil
}

func extractPackageNamesFromRequires(packages interface{}, requiredPackages map[string]bool) {
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
			requiredPackages[name] = true
		}
	}
}

func validatePolicyTemplatePackageReferences(fsys fspath.FS, manifest pkgpath.File, requiredPackages map[string]bool) specerrors.ValidationErrors {
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

			if !requiredPackages[packageName] {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy_templates[%d].inputs[%d] references package \"%s\" which is not listed in requires section",
					fsys.Path("manifest.yml"), templateIndex, inputIndex, packageName))
			}
		}
	}

	return errs
}

func validateDataStreamPackageReferences(fsys fspath.FS, requiredPackages map[string]bool) specerrors.ValidationErrors {
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

			if !requiredPackages[packageName] {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: streams[%d] references package \"%s\" which is not listed in manifest requires section",
					dataStreamManifest.Path(), streamIndex, packageName))
			}
		}
	}

	return errs
}
