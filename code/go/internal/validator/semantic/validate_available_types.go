// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	otelcolInput string = "otelcol"
)

var validSignalTypes = []string{"logs", "metrics", "traces"}

type availableTypesPolicyTemplate struct {
	Name           string   `yaml:"name"`
	Input          string   `yaml:"input"`
	AvailableTypes []string `yaml:"available_types"`
}

type availableTypesPackageManifest struct {
	Type            string                         `yaml:"type"`
	PolicyTemplates []availableTypesPolicyTemplate `yaml:"policy_templates"`
}

// ValidateAvailableTypes validates that the available_types field is only used
// with OTel input packages (input: otelcol) and that all values are valid signal types.
func ValidateAvailableTypes(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err),
		}
	}

	var manifest availableTypesPackageManifest
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err),
		}
	}

	// Only validate input packages
	if manifest.Type != "input" {
		return nil
	}

	// Validate each policy template individually
	for _, policyTemplate := range manifest.PolicyTemplates {
		// If available_types is not present in this template, skip validation
		if len(policyTemplate.AvailableTypes) == 0 {
			continue
		}

		// Validate that available_types is only used with otelcol input
		if policyTemplate.Input != otelcolInput {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: policy template \"%s\" has available_types but does not use otelcol input (input: %s). available_types can only be used with OTel input packages (input: otelcol)",
				fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.Input))
		}

		// Validate that all values in available_types are valid signal types
		for _, signalType := range policyTemplate.AvailableTypes {
			if !slices.Contains(validSignalTypes, signalType) {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\" has invalid signal type %q in available_types, valid values are: %v",
					fsys.Path(manifestPath), policyTemplate.Name, signalType, validSignalTypes))
			}
		}
	}

	return errs
}
