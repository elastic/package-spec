// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

// SimulatedIndexTemplate contains the result of simulating an index template,
// with the component templates resolved.
type SimulatedIndexTemplate struct {
	types.Template
}

// FieldMapping looks for the definition of a field in the simulated index template.
func (t *SimulatedIndexTemplate) FieldMapping(name string) (any, error) {
	if runtimeField, isRuntime := t.Mappings.Runtime[name]; isRuntime {
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

		// Using reflect because Property type is defined as any and there are too many
		// possible matching types.
		value := reflect.ValueOf(property).Elem()
		if value.Kind() != reflect.Struct {
			return nil, fmt.Errorf("property %q not found in index template, unexpected parent kind %s", name, value.Kind())
		}
		nextPropertiesValue := value.FieldByName("Properties")
		if nextPropertiesValue.IsZero() {
			return nil, fmt.Errorf("property %q not found in index template, zero properties", name)
		}
		nextProperties, ok := nextPropertiesValue.Interface().(map[string]types.Property)
		if !ok {
			return nil, fmt.Errorf("property %q not found in index template, cannot convert properties", name)
		}
		properties = nextProperties
	}
	return nil, fmt.Errorf("property %q not found in index template", name)
}
