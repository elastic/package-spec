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

// ValidateVarGroups validates var_groups definitions in manifests.
// It checks that:
// - vars referenced in options[].vars exist in the manifest vars array
// - var_group names are unique
// - option names within each var_group are unique
// - vars in a var_group must not have required: true (requirement is controlled by var_group)
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
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
}

type varGroupOption struct {
	Name string   `yaml:"name"`
	Vars []string `yaml:"vars"`
}

type varGroup struct {
	Name     string           `yaml:"name"`
	Required bool             `yaml:"required"`
	Options  []varGroupOption `yaml:"options"`
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
	var availableVars []varGroupsManifestVar
	availableVars = append(availableVars, manifest.Vars...)
	for _, template := range manifest.PolicyTemplates {
		availableVars = append(availableVars, template.Vars...)
		for _, input := range template.Inputs {
			availableVars = append(availableVars, input.Vars...)
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
		var availableVars []varGroupsManifestVar
		availableVars = append(availableVars, pkgManifest.Vars...)
		availableVars = append(availableVars, stream.Vars...)

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

func validateVarGroups(filePath string, varGroups []varGroup, availableVars []varGroupsManifestVar) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// Build a map for quick var lookup
	varMap := make(map[string]varGroupsManifestVar)
	for _, v := range availableVars {
		varMap[v.Name] = v
	}

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

			// Validate that referenced vars exist and check required consistency
			for _, varName := range opt.Vars {
				varDef, exists := varMap[varName]
				if !exists {
					errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: var %q referenced in var_group %q option %q is not defined", filePath, varName, vg.Name, opt.Name))
					continue
				}

				// Validate that vars in a var_group do not have required: true
				// The requirement is controlled entirely by the var_group:
				// - If var_group is required, all vars are implicitly required (inferred)
				// - If var_group is not required, the entire group is optional
				if varDef.Required {
					if vg.Required {
						errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: var %q in required var_group %q should not have required: true (requirement is inferred from var_group)", filePath, varName, vg.Name))
					} else {
						errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: var %q in non-required var_group %q should not have required: true (var_group is optional)", filePath, varName, vg.Name))
					}
				}
			}
		}
	}

	return errs
}
