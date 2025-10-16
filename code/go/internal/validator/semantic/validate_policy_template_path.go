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
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	defaultStreamTemplatePath = "stream.yml.hbs"
)

var (
	errRequiredTemplatePath  = errors.New("template_path is required for input type packages")
	errFailedToReadManifest  = errors.New("failed to read manifest")
	errFailedToParseManifest = errors.New("failed to parse manifest")
	errTemplateNotFound      = errors.New("template file not found")
)

type policyTemplateInput struct {
	Type         string `yaml:"type"`
	TemplatePath string `yaml:"template_path"` // optional for integration packages
}

type policyTemplate struct {
	Name         string                `yaml:"name"`
	TemplatePath string                `yaml:"template_path"` // input type packages require this field
	Inputs       []policyTemplateInput `yaml:"inputs"`        // integration type packages
}

type packageManifest struct { // package manifest
	Type            string           `yaml:"type"` // integration or input
	PolicyTemplates []policyTemplate `yaml:"policy_templates"`
}

type stream struct {
	Input        string `yaml:"input"`
	TemplatePath string `yaml:"template_path"`
}

type streamManifest struct {
	Streams []stream `yaml:"streams"`
}

// ValidatePolicyTemplates validates that all referenced template_path files exist for integration and input policy templates
func ValidatePolicyTemplates(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %ww", fsys.Path(manifestPath), errFailedToReadManifest)}
	}

	var manifest packageManifest
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), errFailedToParseManifest)}
	}

	for _, policyTemplate := range manifest.PolicyTemplates {
		switch manifest.Type {
		case "integration":
			err := validateIntegrationPackagePolicyTemplate(fsys, policyTemplate)
			if err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\" references input template_path: %w",
					fsys.Path(manifestPath), policyTemplate.Name, err))
			}
		case "input":
			err := validateInputPackagePolicyTemplate(fsys, policyTemplate)
			if err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\" references template_path \"%s\": %w",
					fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.TemplatePath, err))
			}
		}
	}

	return errs
}

// validateInputPackagePolicyTemplate validates the template_path at the policy template level for input type packages
// if the template_path is empty, it returns an error as this field is required for input type packages
func validateInputPackagePolicyTemplate(fsys fspath.FS, policyTemplate policyTemplate) error {
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

// validateIntegrationPackagePolicyTemplate validates the template_path at the inputs level for integration type packages
// if the template_path is empty, it looks up at the data stream manifest for the stream input that matches the input type of the policy template
// and uses its template_path to look for the corresponding template file at the data stream stream directory
// if no matching stream input is found, it returns an error as at least one stream input must match the input type of the policy template
// if a matching stream input is found but its template_path file does not exist, it returns an error
func validateIntegrationPackagePolicyTemplate(fsys fspath.FS, policyTemplate policyTemplate) error {
	for _, input := range policyTemplate.Inputs {
		if input.TemplatePath != "" {
			err := validateAgentInputTemplatePath(fsys, input.TemplatePath)
			if err != nil {
				return err
			}
			continue
		}

		var found bool
		// when an input.TemplatePath is empty, lookup at the data stream manifest
		err := fs.WalkDir(
			fsys,
			path.Join("data_stream"),
			func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if found {
					return fs.SkipAll
				}
				// read the data stream manifest and look for the stream input that matches the input.type of the policy template
				if !d.IsDir() && d.Name() == "manifest.yml" {
					data, err := fs.ReadFile(fsys, p)
					if err != nil {
						return err
					}
					var sm streamManifest
					err = yaml.Unmarshal(data, &sm)
					if err != nil {
						return err
					}
					for _, stream := range sm.Streams {
						// skip if the stream input type does not match the policy template input type
						if stream.Input == input.Type {
							streamName := path.Base(path.Dir(p))
							// as template_path is optional at the stream level, default to "stream.yml.hbs" if not set
							templatePath := stream.TemplatePath
							if templatePath == "" {
								templatePath = defaultStreamTemplatePath
							}

							// look for the template_path file at the data stream stream directory
							err := fs.WalkDir(
								fsys,
								path.Join("data_stream", streamName, "agent", "stream"),
								func(p string, d fs.DirEntry, err error) error {
									if err != nil {
										return err
									}
									if !d.IsDir() && d.Name() != "" && strings.HasSuffix(d.Name(), templatePath) {
										found = true
										return fs.SkipAll
									}
									return nil
								})
							if err != nil {
								return err
							}
						}
					}
				}
				return nil
			})
		if err != nil {
			return err
		}
		if !found {
			return errTemplateNotFound
		}
	}
	return nil
}
