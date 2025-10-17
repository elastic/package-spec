// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	inputPackageType packageType = "input"
)

var (
	errRequiredTemplatePath  = errors.New("template_path is required for input type packages")
	errFailedToReadManifest  = errors.New("failed to read manifest")
	errFailedToParseManifest = errors.New("failed to parse manifest")
	errTemplateNotFound      = errors.New("template file not found")
	errInvalidPackageType    = errors.New("invalid package type")
)

type packageType string

type inputPolicyTemplate struct {
	Name         string `yaml:"name"`
	TemplatePath string `yaml:"template_path"` // input type packages require this field
}

type inputPackageManifest struct { // package manifest
	Type            packageType           `yaml:"type"`
	PolicyTemplates []inputPolicyTemplate `yaml:"policy_templates"`
}

// ValidateInputPackagesPolicyTemplates
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

// validateInputPackagePolicyTemplate validates the template_path at the policy template level for input type packages
// if the template_path is empty, it returns an error as this field is required for input type packages
func validateInputPackagePolicyTemplate(fsys fspath.FS, policyTemplate inputPolicyTemplate) error {
	if policyTemplate.TemplatePath == "" {
		return errRequiredTemplatePath
	}
	return validateAgentInputTemplatePath(fsys, policyTemplate.TemplatePath)
}

func validateAgentInputTemplatePath(fsys fspath.FS, tmplPath string) error {
	templatePath := path.Join("agent", "input", tmplPath)
	_, err := fs.Stat(fsys, templatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errTemplateNotFound
		}
		return fmt.Errorf("failed to stat template file %s: %w", fsys.Path(templatePath), err)
	}

	return nil
}
