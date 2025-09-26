// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"path"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
	"gopkg.in/yaml.v3"
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
		Type   string `yaml:"type"` // integration or input
		Inputs []struct {
			Title           string     `yaml:"title"`
			PolicyTemplates []struct { // policy templates reference for integration type packages
				Name         string `yaml:"name"`
				TemplatePath string `yaml:"template_path"`
			} `yaml:"policy_templates"`
		} `yaml:"inputs"`

		PolicyTemplates []struct { // policy templates reference for input type packages
			Name         string `yaml:"name"`
			TemplatePath string `yaml:"template_path"` // optional
		} `yaml:"policy_templates"`
	}

	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	switch manifest.Type {
	case "integration":
		for _, input := range manifest.Inputs {
			for _, policyTemplate := range input.PolicyTemplates {
				if policyTemplate.TemplatePath == "" {
					continue // template_path is optional
				}
				err := validateTemplatePath(fsys, policyTemplate.TemplatePath)
				if err != nil {
					errs = append(errs, specerrors.NewStructuredErrorf(
						"file \"%s\" is invalid: policy template \"%s\" references template_path \"%s\" but file \"%s\" does not exist",
						fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.TemplatePath, fsys.Path(policyTemplate.TemplatePath)))
				}
			}
		}
	case "input":
		for _, policyTemplate := range manifest.PolicyTemplates {
			if policyTemplate.TemplatePath == "" {
				continue // template_path is optional
			}
			err := validateTemplatePath(fsys, policyTemplate.TemplatePath)
			if err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\" references template_path \"%s\" but file \"%s\" does not exist",
					fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.TemplatePath, fsys.Path(policyTemplate.TemplatePath)))
			}
		}
	}

	return errs
}

func validateTemplatePath(fsys fspath.FS, tmplPath string) error {
	templatePath := path.Join("agent", "input", tmplPath)
	_, err := fs.Stat(fsys, templatePath)
	if err != nil {
		return err
	}

	return nil
}
