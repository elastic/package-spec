// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	useAPMVarName        = "use_apm"
	datasetVarName       = "data_stream.dataset"
	tracesDataStreamType = "traces"
)

type varDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// normalizedStream is a unified representation of a stream for reserved variable
// validation, regardless of whether it originates from an input package policy
// template or an integration data stream manifest.
type normalizedStream struct {
	inputType          string
	dataStreamType     string
	dynamicSignalTypes bool
	vars               []varDef
	contextStr         string
	filePath           string
}

// policyTemplate parses the fields needed from an input package policy template.
type policyTemplate struct {
	Name               string   `yaml:"name"`
	Input              string   `yaml:"input"`
	Type               string   `yaml:"type"`
	DynamicSignalTypes bool     `yaml:"dynamic_signal_types"`
	Vars               []varDef `yaml:"vars"`
}

// packageManifest captures the package type and policy templates needed during
// normalization. PolicyTemplates is only meaningful for input packages.
type packageManifest struct {
	Type            string           `yaml:"type"`
	PolicyTemplates []policyTemplate `yaml:"policy_templates"`
}

// streamEntry parses the fields needed from a data stream manifest stream entry.
type streamEntry struct {
	Input              string   `yaml:"input"`
	DynamicSignalTypes bool     `yaml:"dynamic_signal_types"`
	Vars               []varDef `yaml:"vars"`
}

// ValidateFleetReservedVars validates that Fleet-reserved variables, when
// explicitly defined in package manifests, conform to Fleet's expectations:
//   - use_apm: must be type "bool", only on otelcol inputs that are "traces"
//     data streams or when "dynamic_signal_types" is true
//   - data_stream.dataset: must be type "text"
func ValidateFleetReservedVars(fsys fspath.FS) specerrors.ValidationErrors {
	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err)}
	}

	var manifest packageManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	streams, errs := normalizePackageStreams(fsys, manifest)
	for _, stream := range streams {
		for _, v := range stream.vars {
			errs = append(errs, validateReservedVar(v, stream)...)
		}
	}

	return errs
}

// normalizePackageStreams converts a package's manifests into a flat list of
// normalizedStream values.
func normalizePackageStreams(fsys fspath.FS, manifest packageManifest) ([]normalizedStream, specerrors.ValidationErrors) {
	switch manifest.Type {
	case inputPackageType:
		return normalizeInputStreams(fsys, manifest), nil
	case integrationPackageType:
		return normalizeIntegrationStreams(fsys)
	}
	return nil, nil
}

func normalizeInputStreams(fsys fspath.FS, manifest packageManifest) []normalizedStream {
	filePath := fsys.Path("manifest.yml")
	streams := make([]normalizedStream, 0, len(manifest.PolicyTemplates))
	for _, pt := range manifest.PolicyTemplates {
		streams = append(streams, normalizedStream{
			inputType:          pt.Input,
			dataStreamType:     pt.Type,
			dynamicSignalTypes: pt.DynamicSignalTypes,
			vars:               pt.Vars,
			contextStr:         fmt.Sprintf("policy template %q", pt.Name),
			filePath:           filePath,
		})
	}
	return streams
}

func normalizeIntegrationStreams(fsys fspath.FS) ([]normalizedStream, specerrors.ValidationErrors) {
	var streams []normalizedStream
	var errs specerrors.ValidationErrors

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return nil, specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("failed to list data streams: %w", err)}
	}

	for _, ds := range dataStreams {
		manifestPath := dataStreamDir + "/" + ds + "/manifest.yml"
		data, err := fs.ReadFile(fsys, manifestPath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		var dsManifest struct {
			Type    string        `yaml:"type"`
			Streams []streamEntry `yaml:"streams"`
		}
		if err := yaml.Unmarshal(data, &dsManifest); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		for _, stream := range dsManifest.Streams {
			streams = append(streams, normalizedStream{
				inputType:          stream.Input,
				dataStreamType:     dsManifest.Type,
				dynamicSignalTypes: stream.DynamicSignalTypes,
				vars:               stream.Vars,
				contextStr:         fmt.Sprintf("stream with input type %q", stream.Input),
				filePath:           fsys.Path(manifestPath),
			})
		}
	}

	return streams, errs
}

// validateReservedVar dispatches to per-variable validation based on the
// variable name.
func validateReservedVar(v varDef, stream normalizedStream) specerrors.ValidationErrors {
	switch v.Name {
	case useAPMVarName:
		return validateUseAPMVar(v, stream)
	case datasetVarName:
		return validateDatasetVar(v, stream)
	}
	return nil
}

// validateUseAPMVar enforces that use_apm is:
//   - defined on an otelcol input
//   - of type "bool"
//   - only present on "traces" data streams or when "dynamic_signal_types" is true
func validateUseAPMVar(v varDef, stream normalizedStream) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	if stream.inputType != otelcolInputType {
		errs = append(errs, newReservedVarError(stream.filePath, stream.contextStr, useAPMVarName,
			fmt.Sprintf("%q input", otelcolInputType), fmt.Sprintf("%q", stream.inputType)))
	}
	if v.Type != "bool" {
		errs = append(errs, newReservedVarError(stream.filePath, stream.contextStr, useAPMVarName,
			`type "bool"`, fmt.Sprintf("%q", v.Type)))
	}
	if stream.dataStreamType != tracesDataStreamType && !stream.dynamicSignalTypes {
		errs = append(errs, newReservedVarError(stream.filePath, stream.contextStr, useAPMVarName,
			fmt.Sprintf("%q data stream type or \"dynamic_signal_types: true\"", tracesDataStreamType),
			fmt.Sprintf("%q data stream type", stream.dataStreamType)))
	}

	return errs
}

// validateDatasetVar enforces that data_stream.dataset is of type "text".
func validateDatasetVar(v varDef, stream normalizedStream) specerrors.ValidationErrors {
	if v.Type != "text" {
		return specerrors.ValidationErrors{
			newReservedVarError(stream.filePath, stream.contextStr, datasetVarName,
				`type "text"`, fmt.Sprintf("%q", v.Type)),
		}
	}
	return nil
}

// newReservedVarError constructs a validation error for a Fleet-reserved
// variable that doesn't satisfy its constraint. expected describes the
// requirement (e.g. `type "bool"`) and got is the actual value found.
func newReservedVarError(filePath, contextStr, varName, expected, got string) *specerrors.StructuredError {
	return specerrors.NewStructuredErrorf(
		"file \"%s\" is invalid: %s: variable \"%s\" must be %s, got %s",
		filePath, contextStr, varName, expected, got)
}
