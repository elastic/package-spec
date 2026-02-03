// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	errInvalidAgentVersionCondition = fmt.Errorf("invalid agent.version condition")
	errAgentVersionIncorrectType    = fmt.Errorf("manifest agent version is not a string")
)

// ValidateMinimumAgentVersion checks that the package manifest includes the agent.version condition.
func ValidateMinimumAgentVersion(fsys fspath.FS) specerrors.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	agentVersionCondition, err := getAgentVersionCondition(*manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	if agentVersionCondition != "" {
		if _, err := semver.NewConstraint(agentVersionCondition); err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w: %w", fsys.Path(manifest.Name()), errInvalidAgentVersionCondition, err)}
		}
	}

	return nil
}

func getAgentVersionCondition(manifest pkgpath.File) (string, error) {
	val, err := manifest.Values("$.conditions[\"agent.version\"]")
	if err != nil {
		val, err = manifest.Values("$.conditions.agent.version")
		if err != nil {
			return "", nil
		}
	}

	sVal, ok := val.(string)
	if !ok {
		return "", errAgentVersionIncorrectType
	}

	return sVal, nil
}
