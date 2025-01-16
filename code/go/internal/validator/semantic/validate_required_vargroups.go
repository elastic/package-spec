// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"slices"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
	"gopkg.in/yaml.v3"
)

// ValidateRequiredVarGroups validates lists of optional required variables.
func ValidateRequiredVarGroups(fsys fspath.FS) specerrors.ValidationErrors {
	d, err := fs.ReadFile(fsys, "manifest.yml")
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to read manifest: %w", err)}
	}

	var manifest requiredVarsManifest
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to parse manifest: %w", err)}
	}

	return validateRequiredVarGroups(manifest)
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
			RequiredVars map[string][]struct {
				Name string `yaml:"name"`
			} `yaml:"required_vars"`
		} `yaml:"inputs"`
	} `yaml:"policy_templates"`
}

func validateRequiredVarGroups(manifest requiredVarsManifest) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	for _, template := range manifest.PolicyTemplates {
		var vars []requiredVarsManifestVar
		vars = append(vars, manifest.Vars...)
		vars = append(vars, template.Vars...)
		for _, input := range template.Inputs {
			for _, varGroup := range input.RequiredVars {
				for _, requiredVar := range varGroup {
					i := slices.IndexFunc(vars, func(v requiredVarsManifestVar) bool {
						return requiredVar.Name == v.Name
					})
					if i < 0 {
						errs = append(errs, specerrors.NewStructuredErrorf("required var %q in optional group is not defined", requiredVar.Name))
						continue
					}
					if vars[i].Required {
						errs = append(errs, specerrors.NewStructuredErrorf("required var %q in optional group is defined as always required", requiredVar.Name))
					}
				}
			}
		}
	}
	return errs
}
