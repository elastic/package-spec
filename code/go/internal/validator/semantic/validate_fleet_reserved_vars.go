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
	useAPMVarName  = "use_apm"
	datasetVarName = "data_stream.dataset"
)

// streamContext is implemented by types that hold the input type and contextual
// description used when reporting reserved variable validation errors.
type streamContext interface {
	getInput() string
	contextString() string
}

type varDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// policyTemplate is used to read the name, input type, and vars from a policy
// template in an input package manifest.
type policyTemplate struct {
	Name  string   `yaml:"name"`
	Input string   `yaml:"input"`
	Vars  []varDef `yaml:"vars"`
}

func (pt policyTemplate) getInput() string { return pt.Input }
func (pt policyTemplate) contextString() string {
	return fmt.Sprintf("policy template %q", pt.Name)
}

// packageManifest captures the package type and policy templates needed to
// validate Fleet-reserved variable definitions in a package manifest.
type packageManifest struct {
	Type            string           `yaml:"type"`
	PolicyTemplates []policyTemplate `yaml:"policy_templates"`
}

// streamEntry is used to read the input type and vars from a stream entry in
// a data stream manifest.
type streamEntry struct {
	Input string   `yaml:"input"`
	Vars  []varDef `yaml:"vars"`
}

func (s streamEntry) getInput() string { return s.Input }
func (s streamEntry) contextString() string {
	return fmt.Sprintf("stream with input type %q", s.Input)
}

// ValidateFleetReservedVars validates that Fleet-reserved variables, when
// explicitly defined in package manifests, conform to Fleet's expectations:
//   - use_apm: must be type "bool", only allowed on otelcol inputs
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

	switch manifest.Type {
	case inputPackageType:
		return validateInputReservedVars(fsys, manifest, manifestPath)
	case integrationPackageType:
		return validateIntegrationReservedVars(fsys)
	}

	return nil
}

func validateInputReservedVars(fsys fspath.FS, manifest packageManifest, manifestPath string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	for _, pt := range manifest.PolicyTemplates {
		for _, v := range pt.Vars {
			errs = append(errs, validateReservedVar(v, pt, fsys.Path(manifestPath))...)
		}
	}

	return errs
}

func validateIntegrationReservedVars(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
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

		var manifest struct {
			Streams []streamEntry `yaml:"streams"`
		}
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		for _, stream := range manifest.Streams {
			for _, v := range stream.Vars {
				errs = append(errs, validateReservedVar(v, stream, fsys.Path(manifestPath))...)
			}
		}
	}

	return errs
}

// validateReservedVar dispatches to per-variable validation based on the
// variable name.
func validateReservedVar(v varDef, ctx streamContext, filePath string) specerrors.ValidationErrors {
	switch v.Name {
	case useAPMVarName:
		return validateUseAPMVar(v, ctx, filePath)
	case datasetVarName:
		return validateDatasetVar(v, ctx, filePath)
	}
	return nil
}

// validateUseAPMVar enforces that use_apm is of type "bool" and only defined
// on otelcol inputs.
func validateUseAPMVar(v varDef, ctx streamContext, filePath string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	if ctx.getInput() != otelcolInputType {
		errs = append(errs, newReservedVarError(filePath, ctx.contextString(), useAPMVarName,
			fmt.Sprintf("%q input", otelcolInputType), fmt.Sprintf("%q", ctx.getInput())))
	}
	if v.Type != "bool" {
		errs = append(errs, newReservedVarError(filePath, ctx.contextString(), useAPMVarName,
			`type "bool"`, fmt.Sprintf("%q", v.Type)))
	}

	return errs
}

// validateDatasetVar enforces that data_stream.dataset is of type "text".
func validateDatasetVar(v varDef, ctx streamContext, filePath string) specerrors.ValidationErrors {
	if v.Type != "text" {
		return specerrors.ValidationErrors{
			newReservedVarError(filePath, ctx.contextString(), datasetVarName,
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
