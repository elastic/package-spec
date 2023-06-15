// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specpatch

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

// Version represents the JSON patches to be applied in each version
type Version struct {
	// Before is the first version that didn't include this change.
	Before string `json:"before" yaml:"before"`

	// Patch is a list of JSON patch operations as defined by RFC6902.
	Patch []interface{} `json:"patch" yaml:"patch"`
}

// PatchForVersion obtains the JSON patch to be applied given a specific version (e.g. 2.0.0)
// and a list of patches per version (Version struct)
func PatchForVersion(target semver.Version, versions []Version) ([]byte, error) {
	var patch []any
	for _, version := range versions {
		if sv, err := semver.NewVersion(version.Before); err != nil {
			return nil, err
		} else if !target.LessThan(sv) {
			continue
		}

		patch = append(patch, version.Patch...)
	}
	if len(patch) == 0 {
		return nil, nil
	}
	return json.Marshal(patch)
}

// ResolvePatch applies the JSON patch to the given spec
func ResolvePatch(spec any, patchJSON []byte) ([]byte, error) {
	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode patch: %w", err)
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec for patching: %w", err)
	}

	return patch.Apply(specBytes)
}
