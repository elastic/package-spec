// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
	"gopkg.in/yaml.v3"
)

func ValidateIntegrationInputsDeprecation(fsys fspath.FS) specerrors.ValidationErrors {

	type manifest struct {
		Type       string `yaml:"type,omitempty"`
		Deprecated *struct {
			Since       string `yaml:"since,omitempty"`
			Description string `yaml:"description,omitempty"`
		} `yaml:"deprecated,omitempty"`
		PolicyTemplates []struct {
			Inputs []struct {
				Deprecated *struct {
					Since       string `yaml:"since,omitempty"`
					Description string `yaml:"description,omitempty"`
				} `yaml:"deprecated,omitempty"`
			} `yaml:"inputs,omitempty"`
		} `yaml:"policy_templates,omitempty"`
	}

	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}

	var m manifest
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}
	// skip if not an integration package
	if m.Type != packageTypeIntegration {
		return nil
	}

	// if package is deprecated, skip checks
	if m.Deprecated != nil {
		return nil
	}

	total := 0
	deprecated := 0
	for _, pt := range m.PolicyTemplates {
		for _, input := range pt.Inputs {
			total++
			if input.Deprecated != nil {
				deprecated++
			}
		}
	}
	if deprecated == total && total > 0 {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: all inputs are deprecated but the integration package is not marked as deprecated", fsys.Path(manifestPath)),
		}
	}

	return nil
}
