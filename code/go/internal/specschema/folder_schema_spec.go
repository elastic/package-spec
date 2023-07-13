// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specschema

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v2/code/go/internal/specpatch"
)

type folderSchemaSpec struct {
	Spec     *folderItemSpec     `json:"spec" yaml:"spec"`
	Versions []specpatch.Version `json:"versions" yaml:"versions"`
}

func (f *folderSchemaSpec) resolve(target semver.Version) (*folderItemSpec, error) {
	patchJSON, err := specpatch.PatchForVersion(target, f.Versions)
	if err != nil {
		return nil, err
	}
	if len(patchJSON) == 0 {
		// Nothing to do.
		return f.Spec, nil
	}

	spec, err := specpatch.ResolvePatch(f.Spec, patchJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	var resolved folderItemSpec
	err = json.Unmarshal(spec, &resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal resolved spec: %w", err)
	}
	return &resolved, nil
}

func (f *folderSchemaSpec) patchForVersion(target semver.Version) ([]byte, error) {
	var patch []any
	for _, version := range f.Versions {
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
