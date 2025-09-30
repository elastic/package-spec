// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateInputPolicyTemplates validates that all referenced template_path files exist for integration and input policy templates
func ValidateInputPolicyTemplates(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := "manifest.yml"

	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err)}
	}

	var manifest struct { // package manifest
		Type string `yaml:"type"` // integration or input

		PolicyTemplates []struct {
			Name         string `yaml:"name"`
			TemplatePath string `yaml:"template_path"` // optional, input type packages
			Inputs       []struct {
				Title        string `yaml:"title"`
				TemplatePath string `yaml:"template_path"` // optional, integration type packages
			} `yaml:"inputs"`
		} `yaml:"policy_templates"`
	}

	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	for _, policyTemplate := range manifest.PolicyTemplates {
		switch manifest.Type {
		case "integration":
			for _, input := range policyTemplate.Inputs {
				if input.TemplatePath == "" {
					continue // template_path is optional
				}
				err := validateTemplatePath(fsys, input.TemplatePath)
				if err != nil {
					errs = append(errs, specerrors.NewStructuredErrorf(
						"file \"%s\" is invalid: policy template \"%s\" references template_path \"%s\": %w",
						fsys.Path(manifestPath), policyTemplate.Name, input.TemplatePath, err))
				}
			}

		case "input":
			if policyTemplate.TemplatePath == "" {
				continue // template_path is optional
			}
			err := validateTemplatePath(fsys, policyTemplate.TemplatePath)
			if err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\" references template_path \"%s\": %w",
					fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.TemplatePath, err))
			}
		}
	}

	return errs
}

func validateTemplatePath(fsys fspath.FS, tmplPath string) error {
	templatePath := filepath.Join("agent", "input", tmplPath)
	_, err := fs.Stat(fsys, templatePath)
	if err != nil {
		return err
	}

	return nil
}
