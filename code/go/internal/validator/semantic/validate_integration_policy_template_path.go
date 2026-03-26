// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
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
	Type          string   `yaml:"type"`
	TemplatePath  string   `yaml:"template_path"`
	TemplatePaths []string `yaml:"template_paths"`
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
	Input         string   `yaml:"input"`
	TemplatePath  string   `yaml:"template_path"`
	TemplatePaths []string `yaml:"template_paths"`
}

type dataStreamManifest struct {
	Streams []stream `yaml:"streams"`
}

// dataStreamManifestReadError records which data_stream/<name>/manifest.yml failed to read or parse.
type dataStreamManifestReadError struct {
	relPath string
	err     error
}

func (e *dataStreamManifestReadError) Error() string { return e.err.Error() }
func (e *dataStreamManifestReadError) Unwrap() error { return e.err }

// ValidateIntegrationPolicyTemplates validates agent input and stream template files for
// integration packages, following Fleet/EPM resolution (template_paths before template_path;
// stream default stream.yml.hbs when neither is set on a stream).
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

	if manifest.Type != packageTypeIntegration {
		return nil
	}

	dataStreamsManifestMap, err := readDataStreamsManifests(fsys)
	if err != nil {
		var dsReadErr *dataStreamManifestReadError
		if errors.As(err, &dsReadErr) {
			return specerrors.ValidationErrors{
				specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(dsReadErr.relPath), dsReadErr.err)}
		}
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("invalid data stream manifests: %w", err)}
	}

	errs = append(errs, validateAllDataStreamStreamTemplates(fsys, dataStreamsManifestMap)...)

	for _, policyTemplate := range manifest.PolicyTemplates {
		if err := validateIntegrationPolicyTemplateInputs(fsys, policyTemplate); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: policy template \"%s\": %w",
				fsys.Path(manifestPath), policyTemplate.Name, err))
		}
	}

	return errs
}

// validateIntegrationPolicyTemplateInputs validates policy template inputs[] template files
// under agent/input when template_paths or template_path is set (Fleet: template_paths first).
func validateIntegrationPolicyTemplateInputs(fsys fspath.FS, policyTemplate integrationPolicyTemplate) error {
	for _, input := range policyTemplate.Inputs {
		if len(input.TemplatePaths) > 0 {
			for _, tp := range input.TemplatePaths {
				if err := validateAgentInputTemplatePath(fsys, tp); err != nil {
					return fmt.Errorf("failed validation for policy input %q: %w", input.Type, err)
				}
			}
			continue
		}
		if input.TemplatePath != "" {
			if err := validateAgentInputTemplatePath(fsys, input.TemplatePath); err != nil {
				return fmt.Errorf("failed validation for policy input %q: %w", input.Type, err)
			}
		}
	}
	return nil
}

// validateAllDataStreamStreamTemplates validates every stream row in every data stream manifest.
func validateAllDataStreamStreamTemplates(fsys fspath.FS, dsMap map[string]dataStreamManifest) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	dsDirs := make([]string, 0, len(dsMap))
	for d := range dsMap {
		dsDirs = append(dsDirs, d)
	}
	slices.Sort(dsDirs)

	for _, dsDir := range dsDirs {
		dsManifestPath := path.Join(dsDir, "manifest.yml")
		manifest := dsMap[dsDir]
		for i, s := range manifest.Streams {
			if err := validateSingleDataStreamStreamTemplates(fsys, dsDir, s); err != nil {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: data stream \"%s\" stream %d (input %q): %w",
					fsys.Path(dsManifestPath), dsDir, i+1, streamInputLabel(s.Input), err))
			}
		}
	}
	return errs
}

func streamInputLabel(input string) string {
	if input == "" {
		return ""
	}
	return input
}

// validateSingleDataStreamStreamTemplates checks stream template files under dsDir/agent/stream
// using Fleet parseAndVerifyStreams / compile precedence (template_paths first, else template_path
// or default stream.yml.hbs).
func validateSingleDataStreamStreamTemplates(fsys fspath.FS, dsDir string, s stream) error {
	dir := path.Join(dsDir, "agent", "stream")

	if len(s.TemplatePaths) > 0 {
		for _, tp := range s.TemplatePaths {
			if err := validateStreamTemplateFile(fsys, dir, tp); err != nil {
				return err
			}
		}
		return nil
	}

	tp := s.TemplatePath
	if tp == "" {
		tp = defaultStreamTemplatePath
	}
	return validateStreamTemplateFile(fsys, dir, tp)
}

func validateStreamTemplateFile(fsys fspath.FS, dir, templatePath string) error {
	foundFile, err := findPathAtDirectory(fsys, dir, templatePath)
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

// readDataStreamsManifests reads all data stream manifests and returns a map of data stream directory to its manifest relevant content
func readDataStreamsManifests(fsys fspath.FS) (map[string]dataStreamManifest, error) {
	dsManifestMap := make(map[string]dataStreamManifest, 0)

	dsManifests, err := fs.Glob(fsys, "data_stream/*/manifest.yml")
	if err != nil {
		return nil, err
	}
	for _, file := range dsManifests {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, &dataStreamManifestReadError{relPath: file, err: err}
		}
		var m dataStreamManifest
		err = yaml.Unmarshal(data, &m)
		if err != nil {
			return nil, &dataStreamManifestReadError{relPath: file, err: err}
		}

		dsDir := path.Dir(file)
		dsManifestMap[dsDir] = m
	}

	return dsManifestMap, nil
}

// findPathAtDirectory looks for a file matching the templatePath in the given directory (dir)
// It checks for exact matches, files ending with the templatePath, or templatePath + ".link"
func findPathAtDirectory(fsys fspath.FS, dir, templatePath string) (string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return "", err
	}

	var foundFile string
	for _, entry := range entries {
		name := entry.Name()
		if name == templatePath || name == templatePath+".link" {
			foundFile = path.Join(dir, name)
			break
		}
		if strings.HasSuffix(name, templatePath) || strings.HasSuffix(name, templatePath+".link") {
			foundFile = path.Join(dir, name)
			break
		}
	}
	return foundFile, nil
}
