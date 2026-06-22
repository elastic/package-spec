// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"io/fs"
	"path"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// streamMaterializationEntry captures the fields needed to check input materialization
// in a data stream manifest's streams[] array.
type streamMaterializationEntry struct {
	Input   string `yaml:"input"`
	Package string `yaml:"package"`
}

type dataStreamMaterializationManifest struct {
	Streams []streamMaterializationEntry `yaml:"streams"`
}

// policyTemplateInputMaterialization captures only the fields needed from a policy template input
// to check whether materialization has taken place.
type policyTemplateInputMaterialization struct {
	Type    string `yaml:"type"`
	Package string `yaml:"package"`
}

type policyTemplateMaterialization struct {
	Name   string                               `yaml:"name"`
	Inputs []policyTemplateInputMaterialization `yaml:"inputs"`
}

type packageMaterializationManifest struct {
	Type            string                          `yaml:"type"`
	PolicyTemplates []policyTemplateMaterialization `yaml:"policy_templates"`
}

// ValidateStreamInputBundled errors when build-mode manifests carry
// source-only 'package:' fields that the build process should have bundled:
//
//   - data_stream/*/manifest.yml: each stream must have 'input:' set and must NOT
//     have 'package:' (composable-input pattern, source-only).
//   - manifest.yml: each policy_template input must have 'type:' set and must NOT
//     have 'package:' (package-reference pattern, source-only).
func ValidateStreamInputBundled(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	errs = append(errs, validateDataStreamStreamsBundled(fsys)...)
	errs = append(errs, validatePolicyTemplateInputsBundled(fsys)...)

	return errs
}

// validateDataStreamStreamsBundled checks every data_stream/*/manifest.yml for
// stream entries that carry a source-only 'package:' field or are missing 'input:'.
func validateDataStreamStreamsBundled(fsys fspath.FS) specerrors.ValidationErrors {
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("can't list data streams: %w", err),
		}
	}

	// Sort for deterministic error ordering.
	slices.Sort(dataStreams)

	var errs specerrors.ValidationErrors
	for _, dataStreamName := range dataStreams {
		manifestRelPath := path.Join(dataStreamDir, dataStreamName, "manifest.yml")
		data, err := fs.ReadFile(fsys, manifestRelPath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file %q: failed to read data stream manifest: %w",
				fsys.Path(manifestRelPath), err,
			))
			continue
		}

		var manifest dataStreamMaterializationManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file %q: failed to parse data stream manifest: %w",
				fsys.Path(manifestRelPath), err,
			))
			continue
		}

		fullPath := fsys.Path(manifestRelPath)
		for i, s := range manifest.Streams {
			if s.Package != "" {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file %q: stream[%d] has 'package:' which is source-only; build packages must use 'input:' + 'template_paths:'",
					fullPath, i,
				))
			}
		}
	}

	return errs
}

// validatePolicyTemplateInputsBundled checks the package-level manifest.yml for
// policy_template inputs that carry a source-only 'package:' field instead of 'type:'.
func validatePolicyTemplateInputsBundled(fsys fspath.FS) specerrors.ValidationErrors {
	manifestRelPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestRelPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("reading %q: %w", fsys.Path(manifestRelPath), err),
		}
	}

	var manifest packageMaterializationManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(
				"file %q: failed to parse manifest: %w",
				fsys.Path(manifestRelPath), err,
			),
		}
	}

	if manifest.Type != integrationPackageType {
		return nil
	}

	fullPath := fsys.Path(manifestRelPath)
	var errs specerrors.ValidationErrors
	for _, policyTemplate := range manifest.PolicyTemplates {
		for i, input := range policyTemplate.Inputs {
			if input.Package != "" {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file %q: policy_template %q input[%d] has 'package:' which is source-only; build packages must use 'type:'",
					fullPath, policyTemplate.Name, i,
				))
			}
		}
	}

	return errs
}
