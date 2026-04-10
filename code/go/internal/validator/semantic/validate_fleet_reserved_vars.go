// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	useAPMVarName        = "use_apm"
	datasetVarName       = "data_stream.dataset"
	tracesDataStreamType = "traces"
)

// varScope identifies where in a package manifest a variable is declared.
type varScope int

const (
	scopeRoot            varScope = iota // package-level vars
	scopePolicyTemplate                  // policy_templates[].vars
	scopeInput                           // policy_templates[].inputs[].vars
	scopeStream                          // data stream stream entries (the valid scope for reserved vars)
)

// reservedVarRule defines the constraints for a single Fleet-reserved variable.
type reservedVarRule struct {
	// streamOnly marks vars that are invalid outside stream-level declarations.
	// Declarations at root, policy template, or input level produce an error.
	streamOnly bool
	// validateAtStream checks type and eligibility constraints when the var is
	// declared at a valid stream scope. May be nil if only scope enforcement applies.
	validateAtStream func(v varDef, stream normalizedStream) specerrors.ValidationErrors
}

var reservedVarRules = map[string]reservedVarRule{
	useAPMVarName:  {streamOnly: true, validateAtStream: validateUseAPMVar},
	datasetVarName: {streamOnly: true, validateAtStream: validateDatasetVar},
}

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

// policyInput parses the type and vars from an integration policy template input entry.
type policyInput struct {
	Type string   `yaml:"type"`
	Vars []varDef `yaml:"vars"`
}

// policyTemplate parses fields from a package policy template. Input (singular)
// is used by input packages; Inputs (plural) is used by integration packages.
type policyTemplate struct {
	Name               string        `yaml:"name"`
	Input              string        `yaml:"input"`
	Type               string        `yaml:"type"`
	DynamicSignalTypes bool          `yaml:"dynamic_signal_types"`
	Vars               []varDef      `yaml:"vars"`
	Inputs             []policyInput `yaml:"inputs"`
}

// packageManifest captures all fields needed for reserved variable validation
// across both input and integration packages.
type packageManifest struct {
	Type            string           `yaml:"type"`
	Vars            []varDef         `yaml:"vars"`
	PolicyTemplates []policyTemplate `yaml:"policy_templates"`
}

// streamEntry parses the fields needed from a data stream manifest stream entry.
type streamEntry struct {
	Input              string   `yaml:"input"`
	DynamicSignalTypes bool     `yaml:"dynamic_signal_types"`
	Vars               []varDef `yaml:"vars"`
}

// ValidateFleetReservedVars validates that Fleet-reserved variables, when
// explicitly defined in package manifests, conform to Fleet's expectations.
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

	switch manifest.Type {
	case inputPackageType:
		return validateInputReservedVars(fsys, manifest)
	case integrationPackageType:
		return validateIntegrationReservedVars(fsys, manifest)
	}

	return nil
}

func validateInputReservedVars(fsys fspath.FS, manifest packageManifest) specerrors.ValidationErrors {
	filePath := fsys.Path("manifest.yml")
	var errs specerrors.ValidationErrors

	// Root-level vars
	for _, v := range manifest.Vars {
		errs = append(errs, checkReservedVarScope(v, filePath, "package root vars", scopeRoot)...)
	}

	// Policy template vars are stream context for input packages: Fleet synthesizes
	// a data stream from each policy template, placing these vars into streams[0].vars.
	for _, pt := range manifest.PolicyTemplates {
		stream := normalizeInputStream(pt, filePath)
		for _, v := range stream.vars {
			errs = append(errs, checkReservedVarScope(v, filePath, stream.contextStr, scopeStream)...)
			errs = append(errs, validateReservedVar(v, stream)...)
		}
	}

	return errs
}

func validateIntegrationReservedVars(fsys fspath.FS, manifest packageManifest) specerrors.ValidationErrors {
	filePath := fsys.Path("manifest.yml")
	var errs specerrors.ValidationErrors

	// Root-level vars
	for _, v := range manifest.Vars {
		errs = append(errs, checkReservedVarScope(v, filePath, "package root vars", scopeRoot)...)
	}

	for _, pt := range manifest.PolicyTemplates {
		ptCtx := fmt.Sprintf("policy template %q vars", pt.Name)
		// Policy template vars
		for _, v := range pt.Vars {
			errs = append(errs, checkReservedVarScope(v, filePath, ptCtx, scopePolicyTemplate)...)
		}

		for _, input := range pt.Inputs {
			inputCtx := fmt.Sprintf("policy template %q input %q vars", pt.Name, input.Type)
			// Input vars
			for _, v := range input.Vars {
				errs = append(errs, checkReservedVarScope(v, filePath, inputCtx, scopeInput)...)
			}
		}
	}

	// Stream-level vars
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return append(errs, specerrors.NewStructuredErrorf("failed to list data streams: %w", err))
	}

	for _, ds := range dataStreams {
		manifestPath := path.Join(dataStreamDir, ds, "manifest.yml")
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

		for _, entry := range dsManifest.Streams {
			stream := normalizeIntegrationStream(entry, dsManifest.Type, fsys.Path(manifestPath))
			for _, v := range stream.vars {
				errs = append(errs, validateReservedVar(v, stream)...)
			}
		}
	}

	return errs
}

func normalizeInputStream(pt policyTemplate, filePath string) normalizedStream {
	return normalizedStream{
		inputType:          pt.Input,
		dataStreamType:     pt.Type,
		dynamicSignalTypes: pt.DynamicSignalTypes,
		vars:               pt.Vars,
		contextStr:         fmt.Sprintf("policy template %q", pt.Name),
		filePath:           filePath,
	}
}

func normalizeIntegrationStream(entry streamEntry, dsType string, filePath string) normalizedStream {
	return normalizedStream{
		inputType:          entry.Input,
		dataStreamType:     dsType,
		dynamicSignalTypes: entry.DynamicSignalTypes,
		vars:               entry.Vars,
		contextStr:         fmt.Sprintf("stream with input type %q", entry.Input),
		filePath:           filePath,
	}
}

// checkReservedVarScope returns an error if v is a reserved variable that is
// not permitted at the given scope. scope identifies the declaration location
// so the function can enforce scope constraints regardless of call site.
// contextStr describes the location in human-readable form (e.g. "package root
// vars", "policy template \"foo\" vars").
func checkReservedVarScope(v varDef, filePath, contextStr string, scope varScope) specerrors.ValidationErrors {
	rule, ok := reservedVarRules[v.Name]
	if !ok {
		return nil
	}
	if rule.streamOnly && scope != scopeStream {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: %s: variable \"%s\" must only be declared at stream level",
				filePath, contextStr, v.Name),
		}
	}
	return nil
}

// validateReservedVar dispatches to per-variable stream-scope validation via
// the reservedVarRules registry.
func validateReservedVar(v varDef, stream normalizedStream) specerrors.ValidationErrors {
	rule, ok := reservedVarRules[v.Name]
	if !ok || rule.validateAtStream == nil {
		return nil
	}
	return rule.validateAtStream(v, stream)
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
