// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	errAgentVersionConditionMissing = fmt.Errorf("agent.version condition is required")
	errAgentVersionIncorrectType    = fmt.Errorf("manifest agent version is not a string")
)

// ValidateMinimumAgentVersion checks that the package manifest includes the agent.version condition.
// This is required for integration packages since version 3.6.0 of the spec.
func ValidateMinimumAgentVersion(fsys fspath.FS) specerrors.ValidationErrors {
	manifest, err := readManifest(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	agentVersionCondition, err := getAgentVersionCondition(*manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	if agentVersionCondition == "" {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(errAgentVersionConditionMissing, specerrors.UnassignedCode)}
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
