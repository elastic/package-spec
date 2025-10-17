// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
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

	if _, err := semver.NewConstraint(agentVersionCondition); agentVersionCondition != "" && err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(errors.Join(err, errInvalidAgentVersionCondition), specerrors.UnassignedCode)}
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
