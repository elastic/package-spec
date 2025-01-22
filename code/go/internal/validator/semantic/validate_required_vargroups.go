// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"path"
	"slices"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
	"gopkg.in/yaml.v3"
)

// ValidateRequiredVarGroups validates lists of optional required variables.
func ValidateRequiredVarGroups(fsys fspath.FS) specerrors.ValidationErrors {
	// Validate main manifest.
	errs := validateRequiredVarGroups(fsys, "manifest.yml")

	// Validate data stream manifests.
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to list data streams: %w", err)}
	}
	for _, ds := range dataStreams {
		errs = append(errs, validateDataStreamRequiredVarGroups(fsys, path.Join("data_stream", ds, "manifest.yml"))...)
	}

	return errs
}

func validateRequiredVarGroups(fsys fspath.FS, path string) specerrors.ValidationErrors {
	d, err := fs.ReadFile(fsys, path)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(path), err)}
	}

	var manifest requiredVarsManifest
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(path), err)}
	}

	return validateRequiredVarGroupsManifest(fsys.Path(path), manifest)
}

type requiredVarsManifestVar struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
}

type requiredVarsManifest struct {
	Vars            []requiredVarsManifestVar `yaml:"vars"`
	PolicyTemplates []struct {
		Vars   []requiredVarsManifestVar `yaml:"vars"`
		Inputs []struct {
			Vars         []requiredVarsManifestVar            `yaml:"vars"`
			RequiredVars map[string][]requiredVarsManifestVar `yaml:"required_vars"`
		} `yaml:"inputs"`
	} `yaml:"policy_templates"`
}

func validateRequiredVarGroupsManifest(path string, manifest requiredVarsManifest) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	for _, template := range manifest.PolicyTemplates {
		var vars []requiredVarsManifestVar
		vars = append(vars, manifest.Vars...)
		vars = append(vars, template.Vars...)
		for _, input := range template.Inputs {
			vars := append(slices.Clone(vars), input.Vars...)
			for _, varGroup := range input.RequiredVars {
				errs = append(errs,
					validateRequiredVarsDefined(path, vars, varGroup)...)
			}
		}
	}
	return errs
}

func validateDataStreamRequiredVarGroups(fsys fspath.FS, path string) specerrors.ValidationErrors {
	d, err := fs.ReadFile(fsys, path)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(path), err)}
	}

	var manifest requiredVarsDataStreamManifest
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(path), err)}
	}

	return validateDataStreamRequiredVarGroupsManifest(fsys.Path(path), manifest)
}

type requiredVarsDataStreamManifest struct {
	Streams []struct {
		Vars         []requiredVarsManifestVar            `yaml:"vars"`
		RequiredVars map[string][]requiredVarsManifestVar `yaml:"required_vars"`
	} `yaml:"streams"`
}

func validateDataStreamRequiredVarGroupsManifest(path string, manifest requiredVarsDataStreamManifest) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	for _, stream := range manifest.Streams {
		for _, varGroup := range stream.RequiredVars {
			errs = append(errs,
				validateRequiredVarsDefined(path, stream.Vars, varGroup)...)
		}
	}
	return errs
}

func validateRequiredVarsDefined(path string, vars, requiredVars []requiredVarsManifestVar) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	for _, requiredVar := range requiredVars {
		if requiredVar.Name == "" {
			continue
		}
		i := slices.IndexFunc(vars, func(v requiredVarsManifestVar) bool {
			return requiredVar.Name == v.Name
		})
		if i < 0 {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: required var %q in optional group is not defined", path, requiredVar.Name))
			continue
		}
		if vars[i].Required {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: required var %q in optional group is defined as always required", path, requiredVar.Name))
		}
	}
	return errs
}
