// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/specpatch"
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

	// Using UseNumber here ensures numbers are decoded as json.Number for accurate
	// integer vs. float validation.
	dec := json.NewDecoder(bytes.NewReader(spec))
	dec.UseNumber()

	var resolved map[string]interface{}
	err = dec.Decode(&resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal resolved spec: %w", err)
	}
	return resolved, nil
}
