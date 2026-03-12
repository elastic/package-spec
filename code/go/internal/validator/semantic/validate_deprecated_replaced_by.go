// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

type deprecatedInfo struct {
	Since       string `yaml:"since"`
	Description string `yaml:"description"`
	ReplacedBy  *struct {
		Package        string `yaml:"package,omitempty"`
		PolicyTemplate string `yaml:"policy_template,omitempty"`
		Input          string `yaml:"input,omitempty"`
		DataStream     string `yaml:"data_stream,omitempty"`
		Variable       string `yaml:"variable,omitempty"`
	} `yaml:"replaced_by,omitempty"`
}

// ValidateDeprecatedReplacedBy checks that when deprecated.replaced_by is used, the required fields are set.
func ValidateDeprecatedReplacedBy(fsys PackageFS) specerrors.ValidationErrors {
	errs := validatePackageManifestDeprecatedReplacedBy(fsys)
	dsErrs := validateDataStreamsDeprecatedReplacedBy(fsys)

	return append(errs, dsErrs...)
}

func validatePackageManifestDeprecatedReplacedBy(fsys PackageFS) specerrors.ValidationErrors {
	// package manifest structure
	type manifest struct {
		Type            string          `yaml:"type,omitempty"`
		Deprecated      *deprecatedInfo `yaml:"deprecated,omitempty"`
		PolicyTemplates []struct {
			Deprecated *deprecatedInfo `yaml:"deprecated,omitempty"`
			Inputs     []struct {
				Deprecated *deprecatedInfo `yaml:"deprecated,omitempty"`
			} `yaml:"inputs,omitempty"`
		} `yaml:"policy_templates,omitempty"`
	}

	manifestPath := "manifest.yml"
	files, err := fsys.Files(manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}
	if len(files) == 0 {
		return nil
	}
	data, err := files[0].ReadAll()
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
	var errs specerrors.ValidationErrors
	if m.Deprecated != nil && m.Deprecated.ReplacedBy != nil {
		rb := m.Deprecated.ReplacedBy
		if rb.Package == "" {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: deprecated.replaced_by.package must be specified when deprecated.replaced_by is used", fsys.Path(manifestPath)))
		}
	}
	for _, pt := range m.PolicyTemplates {
		if pt.Deprecated != nil && pt.Deprecated.ReplacedBy != nil {
			rb := pt.Deprecated.ReplacedBy
			if rb.PolicyTemplate == "" {
				errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: policy_template deprecated.replaced_by.policy_template must be specified when deprecated.replaced_by is used", fsys.Path(manifestPath)))
			}
		}
		for _, input := range pt.Inputs {
			if input.Deprecated != nil && input.Deprecated.ReplacedBy != nil {
				rb := input.Deprecated.ReplacedBy
				if rb.Input == "" {
					errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: input deprecated.replaced_by.input must be specified when deprecated.replaced_by is used", fsys.Path(manifestPath)))
				}
			}
		}
	}
	return errs

}

func validateDataStreamsDeprecatedReplacedBy(fsys PackageFS) specerrors.ValidationErrors {
	// stream manifest structure
	type streamManifest struct {
		Deprecated *deprecatedInfo `yaml:"deprecated,omitempty"`
		Streams    []struct {
			Vars []struct {
				Deprecated *deprecatedInfo `yaml:"deprecated,omitempty"`
			} `yaml:"vars,omitempty"`
		} `yaml:"streams,omitempty"`
	}

	dsManifestFiles, err := fsys.Files("data_stream/*/manifest.yml")
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("error while searching for data stream manifests: %w", err)}
	}

	var errs specerrors.ValidationErrors
	for _, dsManifestFile := range dsManifestFiles {
		data, err := dsManifestFile.ReadAll()
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(dsManifestFile.Path()), err))
			continue
		}
		var sm streamManifest
		err = yaml.Unmarshal(data, &sm)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(dsManifestFile.Path()), err))
			continue
		}
		if sm.Deprecated != nil && sm.Deprecated.ReplacedBy != nil {
			rb := sm.Deprecated.ReplacedBy
			if rb.DataStream == "" {
				errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: deprecated.replaced_by.data_stream must be specified when deprecated.replaced_by is used", fsys.Path(dsManifestFile.Path())))
			}
		}

		for _, stream := range sm.Streams {
			for _, v := range stream.Vars {
				if v.Deprecated != nil && v.Deprecated.ReplacedBy != nil {
					rb := v.Deprecated.ReplacedBy
					if rb.Variable == "" {
						errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: variable deprecated.replaced_by.variable must be specified when deprecated.replaced_by is used", fsys.Path(dsManifestFile.Path())))
					}
				}
			}
		}
	}
	return errs
}
