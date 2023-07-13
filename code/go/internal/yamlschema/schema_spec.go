// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v2/code/go/internal/specpatch"
)

type itemSchemaSpec struct {
	Spec     map[string]interface{} `json:"spec" yaml:"spec"`
	Versions []specpatch.Version    `json:"versions" yaml:"versions"`
}

func (i *itemSchemaSpec) resolve(target semver.Version) (map[string]interface{}, error) {
	patchJSON, err := specpatch.PatchForVersion(target, i.Versions)
	if err != nil {
		return nil, err
	}
	if len(patchJSON) == 0 {
		// Nothing to do.
		return i.Spec, nil
	}
	spec, err := specpatch.ResolvePatch(i.Spec, patchJSON)
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
	var patch []any
	for _, version := range i.Versions {
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
