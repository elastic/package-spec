// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	otelcolInputType string = "otelcol"
)

// Input package structures
type inputPolicyTemplateWithDynamic struct {
	Name               string `yaml:"name"`
	Input              string `yaml:"input"`
	Type               string `yaml:"type"`
	DynamicSignalTypes bool   `yaml:"dynamic_signal_types"`
}

type inputPackageManifestDynamic struct {
	Type            string                           `yaml:"type"`
	PolicyTemplates []inputPolicyTemplateWithDynamic `yaml:"policy_templates"`
}

// Integration package structures - reuse types from other validators
type integrationInputDynamic struct {
	Type               string `yaml:"type"`
	DynamicSignalTypes bool   `yaml:"dynamic_signal_types"`
}

type integrationPolicyTemplateDynamic struct {
	Name   string                    `yaml:"name"`
	Inputs []integrationInputDynamic `yaml:"inputs"`
}

type integrationPackageManifestDynamic struct {
	Type            string                             `yaml:"type"`
	PolicyTemplates []integrationPolicyTemplateDynamic `yaml:"policy_templates"`
}

// ValidateInputDynamicSignalTypes validates that dynamic_signal_types field is only used with otelcol input type
func ValidateInputDynamicSignalTypes(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// Validate package manifest
	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err)}
	}

	// Try to determine package type first
	var typeCheck struct {
		Type string `yaml:"type"`
	}
	err = yaml.Unmarshal(data, &typeCheck)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	switch typeCheck.Type {
	case inputPackageType:
		errs = append(errs, validateInputPackageDynamicSignalTypes(fsys, data, manifestPath)...)
	case integrationPackageType:
		errs = append(errs, validateIntegrationPackageDynamicSignalTypes(fsys, data, manifestPath)...)
		// Also validate data stream manifests for integration packages
		errs = append(errs, validateDataStreamManifests(fsys)...)
	default:
		// Not an input or integration package, nothing to validate
		return nil
	}

	return errs
}

func validateInputPackageDynamicSignalTypes(fsys fspath.FS, data []byte, manifestPath string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	var manifest inputPackageManifestDynamic
	err := yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	for _, policyTemplate := range manifest.PolicyTemplates {
		// Skip if dynamic_signal_types is not set
		if !policyTemplate.DynamicSignalTypes {
			continue
		}
		// Must be otelcol input
		if policyTemplate.Input != otelcolInputType {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: policy template \"%s\": dynamic_signal_types is only allowed when input is 'otelcol', got '%s'",
				fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.Input))
			continue
		}
	}

	return errs
}

func validateIntegrationPackageDynamicSignalTypes(fsys fspath.FS, data []byte, manifestPath string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	var manifest integrationPackageManifestDynamic
	err := yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	for _, policyTemplate := range manifest.PolicyTemplates {
		for _, input := range policyTemplate.Inputs {
			// Skip if dynamic_signal_types is not set
			if !input.DynamicSignalTypes {
				continue
			}
			// Must be otelcol input
			if input.Type != otelcolInputType {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\": input type \"%s\": dynamic_signal_types is only allowed when input is 'otelcol'",
					fsys.Path(manifestPath), policyTemplate.Name, input.Type))
			}
		}
	}

	return errs
}

// Data stream structures
type dataStreamStream struct {
	Input              string `yaml:"input"`
	DynamicSignalTypes bool   `yaml:"dynamic_signal_types"`
}

type dataStreamManifestDynamic struct {
	Streams []dataStreamStream `yaml:"streams"`
}

func validateDataStreamManifests(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("failed to list data streams: %w", err)}
	}

	for _, dataStream := range dataStreams {
		manifestPath := dataStreamDir + "/" + dataStream + "/manifest.yml"
		data, err := fs.ReadFile(fsys, manifestPath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		var manifest dataStreamManifestDynamic
		err = yaml.Unmarshal(data, &manifest)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		for _, stream := range manifest.Streams {
			// Skip if dynamic_signal_types is not set
			if !stream.DynamicSignalTypes {
				continue
			}
			// Must be otelcol input
			if stream.Input != otelcolInputType {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: stream with input type \"%s\": dynamic_signal_types is only allowed when input is '%s'",
					fsys.Path(manifestPath), stream.Input, otelcolInputType))
			}
		}
	}

	return errs
}
