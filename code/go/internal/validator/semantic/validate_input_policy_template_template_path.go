// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	errRequiredTemplatePath  = errors.New("template_path is required for input type packages")
	errFailedToReadManifest  = errors.New("failed to read manifest")
	errFailedToParseManifest = errors.New("failed to parse manifest")
	errTemplateNotFound      = errors.New("template file not found")
	errInvalidPackageType    = errors.New("invalid package type")
)

type inputPolicyTemplate struct {
	Name          string   `yaml:"name"`
	TemplatePath  string   `yaml:"template_path"`  // input type packages require this field or template_paths
	TemplatePaths []string `yaml:"template_paths"` // alternative to template_path for multiple templates
}

type inputPackageManifest struct { // package manifest
	Type            string                `yaml:"type"`
	PolicyTemplates []inputPolicyTemplate `yaml:"policy_templates"`
}

// ValidateInputPackagesPolicyTemplates validates the policy template entries of an input package
func ValidateInputPackagesPolicyTemplates(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %ww", fsys.Path(manifestPath), errFailedToReadManifest)}
	}

	var manifest inputPackageManifest
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), errFailedToParseManifest)}
	}

	if manifest.Type != inputPackageType {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: expected package type \"%s\", got \"%s\": %w",
				fsys.Path(manifestPath), inputPackageType, manifest.Type, errInvalidPackageType)}
	}

	for _, policyTemplate := range manifest.PolicyTemplates {
		err := validateInputPackagePolicyTemplate(fsys, policyTemplate)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: policy template \"%s\" references template_path \"%s\": %w",
				fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.TemplatePath, err))
		}
	}

	return errs
}

// validateInputPackagePolicyTemplate validates the template_path or template_paths at the policy template level for input type packages
// if both template_path and template_paths are empty, it returns an error as at least one is required for input type packages
func validateInputPackagePolicyTemplate(fsys fspath.FS, policyTemplate inputPolicyTemplate) error {
	if policyTemplate.TemplatePath == "" && len(policyTemplate.TemplatePaths) == 0 {
		return errRequiredTemplatePath
	}

	// Validate template_path if provided
	if policyTemplate.TemplatePath != "" {
		if err := validateAgentInputTemplatePath(fsys, policyTemplate.TemplatePath); err != nil {
			return err
		}
	}

	// Validate template_paths if provided
	for _, tmplPath := range policyTemplate.TemplatePaths {
		if err := validateAgentInputTemplatePath(fsys, tmplPath); err != nil {
			return err
		}
	}

	return nil
}

func validateAgentInputTemplatePath(fsys fspath.FS, tmplPath string) error {
	dir := path.Join("agent", "input")
	foundFile, err := findPathAtDirectory(fsys, dir, tmplPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errTemplateNotFound
		}
		return err
	}
	if foundFile == "" {
		return errTemplateNotFound
	}

	return nil
}
