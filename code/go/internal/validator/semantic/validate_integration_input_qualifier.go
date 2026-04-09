// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

type integrationInputQualifier struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type integrationPolicyTemplateQualifier struct {
	Name   string                      `yaml:"name"`
	Inputs []integrationInputQualifier `yaml:"inputs"`
}

type integrationPackageManifestQualifier struct {
	Type            string                               `yaml:"type"`
	PolicyTemplates []integrationPolicyTemplateQualifier `yaml:"policy_templates"`
}

// ValidateIntegrationInputQualifier checks that if a policy template contains
// multiple inputs of the same type, all of them must have a name set. Without
// names, Fleet cannot distinguish them, leading to the ambiguity this field
// aims to solve.
func ValidateIntegrationInputQualifier(fsys fspath.FS) specerrors.ValidationErrors {
	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err)}
	}

	var manifest integrationPackageManifestQualifier
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	if manifest.Type != integrationPackageType {
		return nil
	}

	return validateInputQualifiers(fsys, manifest, manifestPath)
}

func validateInputQualifiers(fsys fspath.FS, manifest integrationPackageManifestQualifier, manifestPath string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	for _, policyTemplate := range manifest.PolicyTemplates {
		typeCounts := make(map[string]int)
		for _, input := range policyTemplate.Inputs {
			typeCounts[input.Type]++
		}

		reported := make(map[string]bool)
		for _, input := range policyTemplate.Inputs {
			if typeCounts[input.Type] > 1 && input.Name == "" && !reported[input.Type] {
				reported[input.Type] = true
				errs = append(errs, specerrors.NewStructuredError(
					fmt.Errorf("file \"%s\" is invalid: policy template \"%s\": input with type \"%s\" must have a name when multiple inputs of the same type are present",
						fsys.Path(manifestPath), policyTemplate.Name, input.Type),
					specerrors.CodeIntegrationInputQualifierRequired))
			}
		}
	}

	return errs
}
