// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// IndexTemplate contains the result of getting an index template, decoded
// from json into a map.
type IndexTemplate struct {
	IndexPatterns []string `json:"index_patterns"`
}

// SimulatedIndexTemplate contains the result of simulating an index template,
// with the component templates resolved.
type SimulatedIndexTemplate struct {
	Settings struct {
		Index struct {
			Mapping struct {
				Source struct {
					Mode string `json:"mode"`
				} `json:"source"`
			} `json:"mapping"`
		} `json:"index"`
	} `json:"settings"`
	Mappings struct {
		Source struct {
			Mode string `json:"mode"`
		} `json:"_source"`
		Runtime    map[string]MappingProperty `json:"runtime"`
		Properties map[string]MappingProperty `json:"properties"`
	} `json:"mappings"`
}

// MappingProperty is the definition of a property in an index template.
type MappingProperty map[string]any

// CheckCondition checks if a property satisfies a condition. Conditions are in the
// form key:value, where the key and the value are compared with attributes of the
// property.
func (p MappingProperty) CheckCondition(condition string) bool {
	key, value, ok := strings.Cut(condition, ":")
	if !ok {
		panic("cannot understand condition " + condition)
	}

	v, ok := p[strings.TrimSpace(key)]
	if !ok {
		return false
	}

	switch v := v.(type) {
	case string:
		return strings.TrimSpace(value) == strings.TrimSpace(v)
	case bool:
		expected, err := strconv.ParseBool(value)
		return err != nil || expected == v
	case json.Number:
		expected, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return false
		}
		n, err := v.Int64()
		if err != nil {
			return false
		}
		return expected == n
	}

	return false
}

// Properties returns the child properties of this property.
func (p MappingProperty) Properties() (map[string]MappingProperty, error) {
	properties, ok := p["properties"]
	if !ok {
		return nil, nil
	}
	mapProperties, ok := properties.(map[string]any)
	if !ok {
		return nil, errors.New("not a map")
	}

	result := make(map[string]MappingProperty)
	for k, v := range mapProperties {
		anyMap, ok := v.(map[string]any)
		if !ok {
			return nil, errors.New("not a map")
		}
		result[k] = MappingProperty(anyMap)
	}

	return result, nil
}

// FieldMapping looks for the definition of a field in the simulated index template.
func (t *SimulatedIndexTemplate) FieldMapping(name string) (MappingProperty, error) {
	if runtimeField, isRuntime := t.Mappings.Runtime[name]; isRuntime {
		// TODO: Look for some solution to don't need to modify the properties.
		runtimeField["runtime"] = true
		return runtimeField, nil
	}

	parts := strings.Split(name, ".")
	properties := t.Mappings.Properties
	for i, part := range parts {
		property, found := properties[part]
		if !found {
			return nil, fmt.Errorf("property %q not found in index template", name)
		}
		if i+1 == len(parts) {
			return property, nil
		}

		nextProperties, err := property.Properties()
		if err != nil {
			return nil, fmt.Errorf("property %q not found in index template: %w", name, err)
		}
		properties = nextProperties
	}
	return nil, fmt.Errorf("property %q not found in index template", name)
}
