// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	defaultStreamTemplatePath = "stream.yml.hbs"
	packageTypeIntegration    = "integration"
)

type policyTemplateInput struct {
	Type         string `yaml:"type"`
	TemplatePath string `yaml:"template_path"` // optional for integration packages
}

type integrationPolicyTemplate struct {
	Name   string                `yaml:"name"`
	Inputs []policyTemplateInput `yaml:"inputs"`
}

type integrationPackageManifest struct { // package manifest
	Type            string                      `yaml:"type"` // integration or input
	PolicyTemplates []integrationPolicyTemplate `yaml:"policy_templates"`
}

type stream struct {
	Input        string `yaml:"input"`
	TemplatePath string `yaml:"template_path"`
}

type dataStreamManifest struct {
	Streams []stream `yaml:"streams"`
}

// ValidateIntegrationPolicyTemplates validates the template_path fields at the policy template level for integration type packages
func ValidateIntegrationPolicyTemplates(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %ww", fsys.Path(manifestPath), errFailedToReadManifest)}
	}

	var manifest integrationPackageManifest
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), errFailedToParseManifest)}
	}

	// only validate integration type packages
	if manifest.Type != packageTypeIntegration {
		return nil
	}

	// read at once all data stream manifests
	dataStreamsManifestMap, err := readDataStreamsManifests(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}

	for _, policyTemplate := range manifest.PolicyTemplates {
		err = validateIntegrationPackagePolicyTemplate(fsys, policyTemplate, dataStreamsManifestMap)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: policy template \"%s\" references input template_path: %w",
				fsys.Path(manifestPath), policyTemplate.Name, err))
		}
	}

	return errs
}

// validateIntegrationPackagePolicyTemplate validates the template_path fields at the policy template level for integration type packages
func validateIntegrationPackagePolicyTemplate(fsys fspath.FS, policyTemplate integrationPolicyTemplate, dsManifestMap map[string]dataStreamManifest) error {
	for _, input := range policyTemplate.Inputs {
		if input.TemplatePath != "" {
			// validate the provided template_path file exists
			err := validateAgentInputTemplatePath(fsys, input.TemplatePath)
			if err != nil {
				return fmt.Errorf("error validating input \"%s\": %w", input.Type, err)
			}
			continue
		}

		err := validateInputWithStreams(fsys, input.Type, dsManifestMap)
		if err != nil {
			return fmt.Errorf("error validating input from streams \"%s\": %w", input.Type, err)
		}
	}
	return nil
}

// readDataStreamsManifests reads all data stream manifests and returns a map of data stream directory to its manifest relevant content
func readDataStreamsManifests(fsys fspath.FS) (map[string]dataStreamManifest, error) {
	// map of data stream directory to its manifest
	dsManifestMap := make(map[string]dataStreamManifest, 0)

	dsManifests, err := fs.Glob(fsys, "data_stream/*/manifest.yml")
	if err != nil {
		return nil, err
	}
	for _, file := range dsManifests {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}
		var m dataStreamManifest
		err = yaml.Unmarshal(data, &m)
		if err != nil {
			return nil, err
		}

		dsDir := path.Dir(file)
		dsManifestMap[dsDir] = m
	}

	return dsManifestMap, nil
}

// validateInputWithStreams validates that for the given input type, the streams of each dataset related to it have valid template_path files
// an input is related to a data_stream if any of its streams has the same input type as input
func validateInputWithStreams(fsys fspath.FS, input string, dsMap map[string]dataStreamManifest) error {
	for dsDir, manifest := range dsMap {
		for _, stream := range manifest.Streams {
			// only consider streams that match the input type of the policy template
			if stream.Input != input {
				continue
			}
			// if template_path is not set at the stream level, default to "stream.yml.hbs"
			if stream.TemplatePath == "" {
				stream.TemplatePath = defaultStreamTemplatePath
			}

			foundFile, err := findPathWithPattern(fsys, dsDir, stream.TemplatePath)
			if err != nil {
				return err
			}
			if foundFile == "" {
				return errTemplateNotFound
			}

			_, err = fs.ReadFile(fsys, foundFile)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return errTemplateNotFound
				}
				return err
			}
		}
	}

	return nil
}

// findPathWithPattern looks for a file matching the templatePath in the given data stream directory
// It checks for exact matches, files ending with the templatePath, or templatePath + ".link"
func findPathWithPattern(fsys fspath.FS, dsDir, templatePath string) (string, error) {
	// Check for exact match, files ending with stream.TemplatePath, or stream.TemplatePath + ".link"
	pattern := path.Join(dsDir, "agent", "stream", "*"+templatePath+"*")
	matches, err := fs.Glob(fsys, pattern)
	if err != nil {
		return "", err
	}

	// Filter matches to ensure they match our criteria:
	// 1. Exact name match
	// 2. Ends with stream.TemplatePath
	// 3. Equals stream.TemplatePath + ".link"
	var foundFile string
	for _, match := range matches {
		base := filepath.Base(match)
		if base == templatePath || base == templatePath+".link" {
			foundFile = match
			break
		}
		// fallbak to check for suffix match, in case the path is prefixed
		if strings.HasSuffix(match, templatePath) {
			foundFile = match
			break
		}
	}
	return foundFile, nil
}
