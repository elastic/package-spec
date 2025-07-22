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

// ValidateDeploymentModes ensures that for each deployment mode enabled in a policy template,
// there is at least one input that supports that deployment mode.
func ValidateDeploymentModes(fsys fspath.FS) specerrors.ValidationErrors {
	manifestPath := "manifest.yml"
	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err)}
	}

	var manifest struct {
		PolicyTemplates []struct {
			Name            string `yaml:"name"`
			DeploymentModes struct {
				Default struct {
					Enabled *bool `yaml:"enabled"` // Use pointer to detect if field was set, default is true
				} `yaml:"default"`
				Agentless struct {
					Enabled bool `yaml:"enabled"`
				} `yaml:"agentless"`
			} `yaml:"deployment_modes"`
			Inputs []struct {
				Type            string    `yaml:"type"`
				DeploymentModes *[]string `yaml:"deployment_modes"` // Use pointer to detect if field was set
			} `yaml:"inputs"`
		} `yaml:"policy_templates"`
	}

	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	var errs specerrors.ValidationErrors
	for _, template := range manifest.PolicyTemplates {
		// Collect enabled deployment modes for this policy template
		enabledModes := []string{}
		// Default mode is enabled by default, unless explicitly disabled
		if template.DeploymentModes.Default.Enabled == nil || *template.DeploymentModes.Default.Enabled {
			enabledModes = append(enabledModes, "default")
		}
		if template.DeploymentModes.Agentless.Enabled {
			enabledModes = append(enabledModes, "agentless")
		}

		// Check each enabled deployment mode has at least one supporting input
		for _, enabledMode := range enabledModes {
			hasSupport := false
			for _, input := range template.Inputs {
				// If deployment_modes field was not specified, input supports all modes
				if input.DeploymentModes == nil {
					hasSupport = true
					break
				}
				// Check if this input explicitly supports the deployment mode
				for _, inputMode := range *input.DeploymentModes {
					if inputMode == enabledMode {
						hasSupport = true
						break
					}
				}
				if hasSupport {
					break
				}
			}

			if !hasSupport {
				err := fmt.Errorf("file \"%s\" is invalid: policy template \"%s\" enables deployment mode \"%s\" but no input supports this mode", fsys.Path(manifestPath), template.Name, enabledMode)
				errs = append(errs, specerrors.NewStructuredError(err, specerrors.UnassignedCode))
			}
		}

		// Check that input deployment modes are supported by the policy template
		for _, input := range template.Inputs {
			// If deployment_modes field was not specified, input supports all modes
			if input.DeploymentModes == nil {
				continue
			}
			// Check if the input has any deployment modes that are not enabled by the policy template
			for _, mode := range *input.DeploymentModes {
				found := false
				for _, enabledMode := range enabledModes {
					if mode == enabledMode {
						found = true
						break
					}
				}
				if !found {
					err := fmt.Errorf("file \"%s\" is invalid: input \"%s\" in policy template \"%s\" specifies unsupported deployment mode \"%s\"", fsys.Path(manifestPath), input.Type, template.Name, mode)
					errs = append(errs, specerrors.NewStructuredError(err, specerrors.UnassignedCode))
				}
			}
		}
	}

	return errs
}
