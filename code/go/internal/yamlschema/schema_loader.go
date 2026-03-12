// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"

	"github.com/Masterminds/semver/v3"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// fsysLoader implements jsonschema.URLLoader, loading YAML schema files
// from an embedded fs.FS and applying version-specific patches.
type fsysLoader struct {
	fsys    fs.FS
	version semver.Version
}

var _ jsonschema.URLLoader = new(fsysLoader)

func (l *fsysLoader) Load(urlStr string) (any, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parsing source failed (source: %s): %w", urlStr, err)
	}
	resourcePath := strings.TrimPrefix(u.Path, "/")

	data, err := fs.ReadFile(l.fsys, resourcePath)
	if err != nil {
		return nil, fmt.Errorf("reading schema file failed: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("schema file is empty")
	}

	var schema itemSchemaSpec
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("schema unmarshalling failed (path: %s): %w", urlStr, err)
	}
	if len(schema.Spec) == 0 {
		return nil, fmt.Errorf("no spec found in schema file (path: %s)", urlStr)
	}

	resolved, err := schema.resolve(l.version)
	if err != nil {
		return nil, fmt.Errorf("resolving schema failed (path: %s): %w", urlStr, err)
	}

	// fixJSONNumbers ensures numbers use json.Number, as expected by the jsonschema library
	// for accurate integer vs. float distinction during validation.
	return fixJSONNumbers(resolved)
}

// fixJSONNumbers converts number types to json.Number by marshaling to JSON and
// decoding again with UseNumber enabled.
func fixJSONNumbers[T any](v T) (result T, err error) {
	d, err := json.Marshal(v)
	if err != nil {
		return result, err
	}
	dec := json.NewDecoder(bytes.NewReader(d))
	dec.UseNumber()
	return result, dec.Decode(&result)
}
