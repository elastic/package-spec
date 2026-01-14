// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateVarGroups validates var_groups definitions in manifests.
// It checks that:
// - vars referenced in options[].vars exist in the manifest vars array
// - var_group names are unique
// - option names within each var_group are unique
func ValidateVarGroups(fsys fspath.FS) specerrors.ValidationErrors {
	// Validate main manifest.
	d, err := fs.ReadFile(fsys, "manifest.yml")
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path("manifest.yml"), err)}
	}

	var manifest varGroupsManifest
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path("manifest.yml"), err)}
	}
	errs := validateVarGroupsManifest(fsys.Path("manifest.yml"), manifest)

	// Validate data stream manifests.
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to list data streams: %w", err)}
	}
	for _, ds := range dataStreams {
		errs = append(errs, validateDataStreamVarGroups(fsys, path.Join("data_stream", ds, "manifest.yml"), manifest)...)
	}

	return errs
}

type varGroupsManifestVar struct {
	Name string `yaml:"name"`
}

type varGroupOption struct {
	Name string   `yaml:"name"`
	Vars []string `yaml:"vars"`
}

type varGroup struct {
	Name    string           `yaml:"name"`
	Options []varGroupOption `yaml:"options"`
}

type varGroupsManifest struct {
	Vars            []varGroupsManifestVar `yaml:"vars"`
	VarGroups       []varGroup             `yaml:"var_groups"`
	PolicyTemplates []struct {
		Vars   []varGroupsManifestVar `yaml:"vars"`
		Inputs []struct {
			Vars []varGroupsManifestVar `yaml:"vars"`
		} `yaml:"inputs"`
	} `yaml:"policy_templates"`
}

type varGroupsStream struct {
	Title     string                 `yaml:"title"`
	Input     string                 `yaml:"input"`
	Vars      []varGroupsManifestVar `yaml:"vars"`
	VarGroups []varGroup             `yaml:"var_groups"`
}

type varGroupsDataStreamManifest struct {
	Streams []varGroupsStream `yaml:"streams"`
}

func validateVarGroupsManifest(filePath string, manifest varGroupsManifest) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// Collect all available vars from package, policy templates, and inputs
	var availableVars []string
	for _, v := range manifest.Vars {
		availableVars = append(availableVars, v.Name)
	}
	for _, template := range manifest.PolicyTemplates {
		for _, v := range template.Vars {
			availableVars = append(availableVars, v.Name)
		}
		for _, input := range template.Inputs {
			for _, v := range input.Vars {
				availableVars = append(availableVars, v.Name)
			}
		}
	}

	errs = append(errs, validateVarGroups(filePath, manifest.VarGroups, availableVars)...)

	return errs
}

func validateDataStreamVarGroups(fsys fspath.FS, filePath string, pkgManifest varGroupsManifest) specerrors.ValidationErrors {
	d, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		// File might not exist, which is fine
		return nil
	}

	var manifest varGroupsDataStreamManifest
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(filePath), err)}
	}

	var errs specerrors.ValidationErrors

	// Validate var_groups in each stream
	for i, stream := range manifest.Streams {
		if len(stream.VarGroups) == 0 {
			continue
		}

		// Collect available vars from both package manifest and stream-level vars
		var availableVars []string
		for _, v := range pkgManifest.Vars {
			availableVars = append(availableVars, v.Name)
		}
		for _, v := range stream.Vars {
			availableVars = append(availableVars, v.Name)
		}

		streamID := stream.Title
		if streamID == "" {
			streamID = stream.Input
		}
		if streamID == "" {
			streamID = fmt.Sprintf("stream[%d]", i)
		}

		streamErrs := validateVarGroups(
			fmt.Sprintf("%s (stream: %s)", fsys.Path(filePath), streamID),
			stream.VarGroups,
			availableVars,
		)
		errs = append(errs, streamErrs...)
	}

	return errs
}

func validateVarGroups(filePath string, varGroups []varGroup, availableVars []string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// Check for duplicate var_group names
	seenGroupNames := make(map[string]bool)
	for _, vg := range varGroups {
		if seenGroupNames[vg.Name] {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: duplicate var_group name %q", filePath, vg.Name))
		}
		seenGroupNames[vg.Name] = true

		// Check for duplicate option names within each var_group
		seenOptionNames := make(map[string]bool)
		for _, opt := range vg.Options {
			if seenOptionNames[opt.Name] {
				errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: duplicate option name %q in var_group %q", filePath, opt.Name, vg.Name))
			}
			seenOptionNames[opt.Name] = true

			// Validate that referenced vars exist
			for _, varName := range opt.Vars {
				if !slices.Contains(availableVars, varName) {
					errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: var %q referenced in var_group %q option %q is not defined", filePath, varName, vg.Name, opt.Name))
				}
			}
		}
	}

	return errs
}
