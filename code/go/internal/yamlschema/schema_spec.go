// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

type itemSchemaSpec struct {
	Spec     map[string]interface{} `json:"spec" yaml:"spec"`
	Versions []itemSchemaVersion    `json:"versions" yaml:"versions"`
}

type itemSchemaVersion struct {
	// Before is the first version that didn't include this change.
	Before string `json:"before" yaml:"before"`

	// Patch is a list of JSON patch operations as defined by RFC6902.
	Patch []interface{} `json:"patch" yaml:"patch"`
}

func (i *itemSchemaSpec) resolve(target semver.Version) (map[string]interface{}, error) {
	patchJSON, err := i.patchForVersion(target)
	if err != nil {
		return nil, err
	}
	if len(patchJSON) == 0 {
		// Nothing to do.
		return i.Spec, nil
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode patch: %w", err)
	}

	spec, err := json.Marshal(i.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec for patching: %w", err)
	}

	spec, err = patch.Apply(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	var resolved map[string]interface{}
	err = json.Unmarshal(spec, &resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal resolved spec: %w", err)
	}
	return resolved, nil
}

func (i *itemSchemaSpec) patchForVersion(target semver.Version) ([]byte, error) {
	var patch []byte
	for _, version := range i.Versions {
		if sv, err := semver.NewVersion(version.Before); err != nil {
			return nil, err
		} else if !target.LessThan(sv) {
			continue
		}

		patchData, err := json.Marshal(version.Patch)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal patch: %w", err)
		}

		if len(patch) == 0 {
			patch = patchData
			continue
		}

		patch, err = jsonpatch.MergeMergePatches(patch, patchData)
		if err != nil {
			return nil, fmt.Errorf("failed to combine patch: %w", err)
		}
	}
	return patch, nil
}
