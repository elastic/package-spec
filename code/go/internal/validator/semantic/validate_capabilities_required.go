// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path"
	"slices"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateCapabilitiesRequired verifies that the required capabilities are added in package manifest
func ValidateCapabilitiesRequired(fsys fspath.FS) specerrors.ValidationErrors {
	err := ensureSecurityRulesHasSecurityCapability(fsys)
	if err != nil {
		return err
	}
	return nil
}

func ensureSecurityRulesHasSecurityCapability(fsys fspath.FS) specerrors.ValidationErrors {
	securityRuleFilePaths := path.Join("kibana", "security_rule", "*.json")
	files, err := pkgpath.Files(fsys, securityRuleFilePaths)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("error finding Kibana security_rule folder: %w", err)}
	}
	if len(files) == 0 {
		return nil
	}

	capabilities, err := readCapabilities(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	if !slices.Contains(capabilities, "security") {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("found security rule assets in package but security capability is missing in package manifest"),
		}
	}

	return nil
}

func readCapabilities(fsys fspath.FS) ([]string, error) {
	manifest, err := readManifest(fsys)
	if err != nil {
		return nil, err
	}

	vals, err := manifest.Values("$.conditions[\"elastic.capabilities\"]")
	if err != nil {
		vals, err = manifest.Values("$.conditions.elastic.capabilities")
		if err != nil {
			return nil, nil
		}
	}

	capabilities, err := toStringSlice(vals)
	if err != nil {
		return nil, fmt.Errorf("can't convert slice entries: %w", err)
	}

	return capabilities, nil
}
