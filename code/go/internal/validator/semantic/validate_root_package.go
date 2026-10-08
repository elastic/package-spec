// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateRootPackage (PROTOTYPE) checks that `requires.integration`, at the top
// level or under `schemas.<name>.requires`, is only used by root packages, i.e.
// packages without policy templates or data streams.
func ValidateRootPackage(fsys fspath.FS) specerrors.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path("manifest.yml"), err)}
	}

	usesIntegrationRequires := false
	if v, err := manifest.Values("$.requires.integration"); err == nil && v != nil {
		usesIntegrationRequires = true
	}
	if schemas, err := manifest.Values("$.schemas"); err == nil && schemas != nil {
		if m, ok := schemas.(map[string]interface{}); ok {
			for name, s := range m {
				if name == "default" {
					continue
				}
				sm, _ := s.(map[string]interface{})
				req, _ := sm["requires"].(map[string]interface{})
				if _, ok := req["integration"]; ok {
					usesIntegrationRequires = true
				}
			}
		}
	}
	if !usesIntegrationRequires {
		return nil
	}

	var errs specerrors.ValidationErrors
	if pt, err := manifest.Values("$.policy_templates"); err == nil && pt != nil {
		if arr, ok := pt.([]interface{}); !ok || len(arr) > 0 {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: requires.integration is only allowed in root packages, which must not define policy_templates",
				fsys.Path("manifest.yml")))
		}
	}
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		errs = append(errs, specerrors.NewStructuredError(err, specerrors.UnassignedCode))
	} else if len(dataStreams) > 0 {
		errs = append(errs, specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: requires.integration is only allowed in root packages, which must not have data streams",
			fsys.Path("manifest.yml")))
	}
	return errs
}
