// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

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
type varScope string

const (
	scopeRoot           varScope = "root"            // package-level vars
	scopePolicyTemplate varScope = "policy template" // policy_templates[].vars
	scopeInput          varScope = "input"           // policy_templates[].inputs[].vars
	scopeStream         varScope = "stream"          // data stream stream entries (the valid scope for reserved vars)
)

// reservedVarRule defines the constraints for a single Fleet-reserved variable.
// Adding a new reserved variable is a matter of adding an entry to reservedVarRules.
type reservedVarRule struct {
	// allowedScopes lists the scopes where this variable may be declared.
	allowedScopes []varScope
	// validate checks type and eligibility constraints at an allowed scope.
	// May be nil if only scope enforcement is needed.
	validate func(v varDef, ctx varValidationContext) specerrors.ValidationErrors
}

// isAllowedAt reports whether scope is in the rule's allowedScopes.
func (r reservedVarRule) isAllowedAt(scope varScope) bool {
	return slices.Contains(r.allowedScopes, scope)
}

// scopeViolationMsg returns a human-readable description of where the variable
// must be declared, derived from allowedScopes (e.g. "stream level" or
// "root or stream level").
func (r reservedVarRule) scopeViolationMsg() string {
	names := make([]string, len(r.allowedScopes))
	for i, s := range r.allowedScopes {
		names[i] = string(s) + " level"
	}
	return strings.Join(names, " or ")
}

var reservedVarRules = map[string]reservedVarRule{
	useAPMVarName:  {allowedScopes: []varScope{scopeStream}, validate: validateUseAPMVar},
	datasetVarName: {allowedScopes: []varScope{scopeStream}, validate: validateDatasetVar},
}

type varDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// varValidationContext carries all available context at the point a variable is
// validated. filePath, contextStr, manifest, and scope are always set.
// policyTemplate is set at scopePolicyTemplate, scopeInput, and scopeStream
// for input packages. stream is set at scopeStream only.
type varValidationContext struct {
	manifest       packageManifest
	scope          varScope
	filePath       string
	contextStr     string
	policyTemplate *policyTemplate
	stream         *normalizedStream
}

// normalizedStream holds stream-specific fields used for content validation.
// It is a unified representation regardless of whether the stream originates
// from an input package policy template or an integration data stream manifest,
// mirroring Fleet's own stream normalization approach.
type normalizedStream struct {
	inputType          string
	dataStreamType     string
	dynamicSignalTypes bool
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

// reservedVarChecker accumulates validation errors while walking the var
// declarations in a package manifest. Its check method is the single call
// site for all reserved variable validation, regardless of scope.
type reservedVarChecker struct {
	errs specerrors.ValidationErrors
}

// check validates every var in vars using the provided context.
func (c *reservedVarChecker) check(vars []varDef, ctx varValidationContext) {
	for _, v := range vars {
		c.errs = append(c.errs, validateReservedVar(v, ctx)...)
	}
}

func validateInputReservedVars(fsys fspath.FS, manifest packageManifest) specerrors.ValidationErrors {
	filePath := fsys.Path("manifest.yml")
	c := &reservedVarChecker{}

	c.check(manifest.Vars, varValidationContext{
		manifest:   manifest,
		scope:      scopeRoot,
		filePath:   filePath,
		contextStr: "package root vars",
	})

	// Policy template vars are stream context for input packages: Fleet synthesizes
	// a data stream from each policy template, placing these vars into streams[0].vars.
	for _, pt := range manifest.PolicyTemplates {
		stream := normalizeInputStream(pt)
		c.check(pt.Vars, varValidationContext{
			manifest:       manifest,
			scope:          scopeStream,
			filePath:       filePath,
			contextStr:     fmt.Sprintf("policy template %q", pt.Name),
			policyTemplate: &pt,
			stream:         &stream,
		})
	}

	return c.errs
}

func validateIntegrationReservedVars(fsys fspath.FS, manifest packageManifest) specerrors.ValidationErrors {
	filePath := fsys.Path("manifest.yml")
	c := &reservedVarChecker{}

	c.check(manifest.Vars, varValidationContext{
		manifest:   manifest,
		scope:      scopeRoot,
		filePath:   filePath,
		contextStr: "package root vars",
	})

	for _, pt := range manifest.PolicyTemplates {
		c.check(pt.Vars, varValidationContext{
			manifest:       manifest,
			scope:          scopePolicyTemplate,
			filePath:       filePath,
			contextStr:     fmt.Sprintf("policy template %q vars", pt.Name),
			policyTemplate: &pt,
		})
		for _, input := range pt.Inputs {
			c.check(input.Vars, varValidationContext{
				manifest:       manifest,
				scope:          scopeInput,
				filePath:       filePath,
				contextStr:     fmt.Sprintf("policy template %q input %q vars", pt.Name, input.Type),
				policyTemplate: &pt,
			})
		}
	}

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return append(c.errs, specerrors.NewStructuredErrorf("failed to list data streams: %w", err))
	}

	for _, ds := range dataStreams {
		manifestPath := path.Join(dataStreamDir, ds, "manifest.yml")
		data, err := fs.ReadFile(fsys, manifestPath)
		if err != nil {
			c.errs = append(c.errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		var dsManifest struct {
			Type    string        `yaml:"type"`
			Streams []streamEntry `yaml:"streams"`
		}
		if err := yaml.Unmarshal(data, &dsManifest); err != nil {
			c.errs = append(c.errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		dsFilePath := fsys.Path(manifestPath)
		for _, entry := range dsManifest.Streams {
			stream := normalizeIntegrationStream(entry, dsManifest.Type)
			c.check(entry.Vars, varValidationContext{
				manifest:   manifest,
				scope:      scopeStream,
				filePath:   dsFilePath,
				contextStr: fmt.Sprintf("stream with input type %q", entry.Input),
				stream:     &stream,
			})
		}
	}

	return c.errs
}

func normalizeInputStream(pt policyTemplate) normalizedStream {
	return normalizedStream{
		inputType:          pt.Input,
		dataStreamType:     pt.Type,
		dynamicSignalTypes: pt.DynamicSignalTypes,
	}
}

func normalizeIntegrationStream(entry streamEntry, dsType string) normalizedStream {
	return normalizedStream{
		inputType:          entry.Input,
		dataStreamType:     dsType,
		dynamicSignalTypes: entry.DynamicSignalTypes,
	}
}

// validateReservedVar is the single entry point for all reserved variable
// validation. It checks scope constraints via the rule's allowedScopes and,
// when the scope is valid, runs per-variable content validation.
func validateReservedVar(v varDef, ctx varValidationContext) specerrors.ValidationErrors {
	rule, ok := reservedVarRules[v.Name]
	if !ok {
		return nil
	}

	var errs specerrors.ValidationErrors

	if !rule.isAllowedAt(ctx.scope) {
		errs = append(errs, specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: %s: variable \"%s\" must only be declared at %s",
			ctx.filePath, ctx.contextStr, v.Name, rule.scopeViolationMsg()))
	}

	if rule.isAllowedAt(ctx.scope) && rule.validate != nil {
		errs = append(errs, rule.validate(v, ctx)...)
	}

	return errs
}

// validateUseAPMVar enforces that use_apm is:
//   - defined on an otelcol input
//   - of type "bool"
//   - only present on "traces" data streams or when "dynamic_signal_types" is true
func validateUseAPMVar(v varDef, ctx varValidationContext) specerrors.ValidationErrors {
	stream := ctx.stream
	var errs specerrors.ValidationErrors

	if stream.inputType != otelcolInputType {
		errs = append(errs, newReservedVarError(ctx.filePath, ctx.contextStr, useAPMVarName,
			fmt.Sprintf("%q input", otelcolInputType), fmt.Sprintf("%q", stream.inputType)))
	}
	if v.Type != "bool" {
		errs = append(errs, newReservedVarError(ctx.filePath, ctx.contextStr, useAPMVarName,
			`type "bool"`, fmt.Sprintf("%q", v.Type)))
	}
	if stream.dataStreamType != tracesDataStreamType && !stream.dynamicSignalTypes {
		errs = append(errs, newReservedVarError(ctx.filePath, ctx.contextStr, useAPMVarName,
			fmt.Sprintf("%q data stream type or \"dynamic_signal_types: true\"", tracesDataStreamType),
			fmt.Sprintf("%q data stream type", stream.dataStreamType)))
	}

	return errs
}

// validateDatasetVar enforces that data_stream.dataset is of type "text".
func validateDatasetVar(v varDef, ctx varValidationContext) specerrors.ValidationErrors {
	if v.Type != "text" {
		return specerrors.ValidationErrors{
			newReservedVarError(ctx.filePath, ctx.contextStr, datasetVarName,
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
