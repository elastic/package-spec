// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	otelcolInputType string = "otelcol"
)

type inputPolicyTemplateWithDynamic struct {
	Name               string `yaml:"name"`
	Input              string `yaml:"input"`
	DynamicSignalTypes bool   `yaml:"dynamic_signal_types"` // true or false
}

type inputPackageManifestDynamic struct {
	Type            string                           `yaml:"type"`
	PolicyTemplates []inputPolicyTemplateWithDynamic `yaml:"policy_templates"`
}

// ValidateInputDynamicSignalTypes validates that dynamic_signal_types field is only used with otelcol input type
func ValidateInputDynamicSignalTypes(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err)}
	}

	var manifest inputPackageManifestDynamic
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	for _, policyTemplate := range manifest.PolicyTemplates {
		// Check if dynamic_signal_types is explicitly set
		if policyTemplate.DynamicSignalTypes {
			// Must be input package type
			if manifest.Type != inputPackageType {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\": dynamic_signal_types is only allowed for input type packages",
					fsys.Path(manifestPath), policyTemplate.Name))
				continue
			}
			// Must be otelcol input
			if policyTemplate.Input != otelcolInputType {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: policy template \"%s\": dynamic_signal_types is only allowed when input is 'otelcol', got '%s'",
					fsys.Path(manifestPath), policyTemplate.Name, policyTemplate.Input))
			}
		}
	}

	return errs
}
