// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateGroupPackage (PROTOTYPE) checks integration groups:
//   - `requires.integration` is only used by packages without policy templates
//     or data streams.
//   - `schemas` requires `requires.integration`, each schema's integration is
//     listed there, and exactly one schema is the default.
func ValidateGroupPackage(fsys fspath.FS) specerrors.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path("manifest.yml"), err)}
	}
	manifestPath := fsys.Path("manifest.yml")

	required := map[string]bool{}
	usesIntegrationRequires := false
	if v, err := manifest.Values("$.requires.integration"); err == nil && v != nil {
		usesIntegrationRequires = true
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if name, ok := m["package"].(string); ok {
						required[name] = true
					}
				}
			}
		}
	}

	var errs specerrors.ValidationErrors
	if schemas, err := manifest.Values("$.schemas"); err == nil && schemas != nil {
		m, _ := schemas.(map[string]interface{})
		defaults := 0
		for name, s := range m {
			sm, _ := s.(map[string]interface{})
			if d, _ := sm["default"].(bool); d {
				defaults++
			}
			integration, _ := sm["integration"].(string)
			if integration != "" && !required[integration] {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: schemas.%s.integration %q must be listed in requires.integration",
					manifestPath, name, integration))
			}
		}
		if len(m) > 0 && defaults != 1 {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: exactly one schema must set default: true, found %d",
				manifestPath, defaults))
		}
	}

	if !usesIntegrationRequires {
		return errs
	}

	if pt, err := manifest.Values("$.policy_templates"); err == nil && pt != nil {
		if arr, ok := pt.([]interface{}); !ok || len(arr) > 0 {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: requires.integration is only allowed in integration groups, which must not define policy_templates",
				manifestPath))
		}
	}
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		errs = append(errs, specerrors.NewStructuredError(err, specerrors.UnassignedCode))
	} else if len(dataStreams) > 0 {
		errs = append(errs, specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: requires.integration is only allowed in integration groups, which must not have data streams",
			manifestPath))
	}
	return errs
}
