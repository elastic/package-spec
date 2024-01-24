// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

func TestValidateFields(t *testing.T) {
	cases := []struct {
		title           string
		packageTemplate string
		fields          any
		expectedErrors  []string
	}{
		// Check that base templates are fine on their own.
		{
			title:           "base integration_v1_2_time_series",
			packageTemplate: "integration_v1_2_time_series",
		},
		{
			title:           "base integration_v3_0",
			packageTemplate: "integration_v3_0",
		},
		{
			title:           "base integration_v3_0_2",
			packageTemplate: "integration_v3_0_2",
		},
		{
			title:           "base integration_v3_0_3_time_series",
			packageTemplate: "integration_v3_0_3_time_series",
			// This package needs at least a dimension field.
			fields: []map[string]any{
				{
					"name":      "afield",
					"type":      "keyword",
					"dimension": true,
				},
			},
		},

		// Generic validations.
		{
			title:           "unknown type",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name": "afield",
					"type": "unknown",
				},
			},
			expectedErrors: []string{
				`field 0.type: 0.type must be one of the following: "aggregate_metric_double", "alias", "histogram", "constant_keyword", "text", "match_only_text", "keyword", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "date", "date_nanos", "boolean", "binary", "integer_range", "float_range", "long_range", "double_range", "date_range", "ip_range", "group", "geo_point", "object", "ip", "nested", "flattened", "wildcard", "version", "unsigned_long"`,
			},
		},
		{
			title:           "array",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name": "afield",
					"type": "array",
				},
			},
			expectedErrors: []string{
				`field 0.type: 0.type must be one of the following: "aggregate_metric_double", "alias", "histogram", "constant_keyword", "text", "match_only_text", "keyword", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "date", "date_nanos", "boolean", "binary", "integer_range", "float_range", "long_range", "double_range", "date_range", "ip_range", "group", "geo_point", "object", "ip", "nested", "flattened", "wildcard", "version", "unsigned_long"`,
			},
		},
		{
			title:           "invalid custom date",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":        "my_custom_date",
					"type":        "keyword",
					"date_format": "yyy-MM-dd",
				},
			},
			expectedErrors: []string{
				`field "my_custom_date" of type keyword can't set date_format. date_format is allowed for date field type only`,
			},
		},
		{
			title:           "required object type",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name": "object_without_object_type.*",
					"type": "object",
				},
			},
			expectedErrors: []string{
				`field 0: object_type is required`,
			},
		},
		{
			title:           "object with subfields should be of type group",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":        "object_with_subfields",
					"type":        "object",
					"object_type": "keyword",
					"fields": []map[string]any{
						{
							"name": "foo",
							"type": "keyword",
						},
					},
				},
			},
			expectedErrors: []string{
				`field 0.type: 0.type must be one of the following: "group", "nested"`,
			},
		},
		{

			title:           "enabled object without object type",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":    "someobject",
					"type":    "object",
					"enabled": true,
				},
			},
			expectedErrors: []string{
				`field 0.enabled: 0.enabled does not match: false`,
			},
		},
		{

			title:           "disabled object with object type",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":        "someobject",
					"type":        "object",
					"object_type": "keyword",
					"enabled":     false,
				},
			},
			expectedErrors: []string{
				`field 0.enabled: 0.enabled does not match: true`,
			},
		},
		{
			title:           "bad disabled object was allowed till 3.0.2",
			packageTemplate: "integration_v3_0_2",
			fields: []map[string]any{
				{
					"name":        "user_provided_metadata",
					"type":        "object",
					"object_type": "keyword",
					"enabled":     false,
				},
			},
		},

		// aggregate_metric_double
		{
			title:           "aggregate_metric_double",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":           "metrics",
					"type":           "aggregate_metric_double",
					"metrics":        []string{"min", "max", "sum", "value_count"},
					"default_metric": "max",
				},
			},
		},
		{
			title:           "bad aggregate_metric_double: required metrics",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":           "no_metrics",
					"type":           "aggregate_metric_double",
					"default_metric": "max",
				},
			},
			expectedErrors: []string{
				`field 0: metrics is required`,
			},
		},
		{
			title:           "bad aggregate_metric_double: required default metric",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":    "no_default_metric",
					"type":    "aggregate_metric_double",
					"metrics": []string{"min", "max", "sum", "value_count"},
				},
			},
			expectedErrors: []string{
				`field 0: default_metric is required`,
			},
		},
		{
			title:           "bad aggregate_metric_double: wrong metrics",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":           "no_default_metric",
					"type":           "aggregate_metric_double",
					"metrics":        []string{"min", "max", "floor"},
					"default_metric": "max",
				},
			},
			expectedErrors: []string{
				`field 0.metrics.2: 0.metrics.2 must be one of the following: "min", "max", "sum", "value_count", "avg"`,
			},
		},
		{
			title:           "bad aggregate_metric_double: aggregate_metric_double params in long field",
			packageTemplate: "integration_v3_0",
			fields: []map[string]any{
				{
					"name":           "no_default_metric",
					"type":           "long",
					"metrics":        []string{"min", "max", "sum", "value_count"},
					"default_metric": "max",
				},
			},
			expectedErrors: []string{
				`field 0: Must not be present`,
				`field 0: Must not be present`,
			},
		},

		// time series
		{
			title:           "time_series: dimension",
			packageTemplate: "integration_v1_2_time_series",
			fields: []map[string]any{
				{
					"name":      "agent.id",
					"type":      "keyword",
					"dimension": true,
				},
			},
		},
		{
			title:           "counter",
			packageTemplate: "integration_v1_2_time_series",
			fields: []map[string]any{
				{
					"name":        "agent.call_count",
					"type":        "long",
					"metric_type": "counter",
				},
			},
		},
		{
			title:           "gauge",
			packageTemplate: "integration_v1_2_time_series",
			fields: []map[string]any{
				{
					"name":        "agent.current_count",
					"type":        "long",
					"metric_type": "gauge",
				},
			},
		},
		{
			title:           "missing dimensions",
			packageTemplate: "integration_v3_0_3_time_series",
			expectedErrors: []string{
				"time series mode enabled but no dimensions configured",
			},
		},
		{
			title:           "bad time_series: invalid type for time series",
			packageTemplate: "integration_v1_2_time_series",
			fields: []map[string]any{
				{
					"name":        "no_valid_type",
					"type":        "boolean",
					"metric_type": "gauge",
				},
			},
			expectedErrors: []string{
				`field 0.type: 0.type must be one of the following: "histogram", "aggregate_metric_double", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long"`,
			},
		},
		{
			title:           "bad time_series: no type",
			packageTemplate: "integration_v1_2_time_series",
			fields: []map[string]any{
				{
					"name":        "no_type",
					"metric_type": "gauge",
				},
			},
			expectedErrors: []string{
				`field 0: type is required`,
			},
		},
		{
			title:           "bad time_series: histogram cannot be a dimension",
			packageTemplate: "integration_v1_2_time_series",
			fields: []map[string]any{
				{
					"name":        "example.agent.call_duration",
					"type":        "histogram",
					"metric_type": "gauge",
					"dimension":   true,
				},
			},
			expectedErrors: []string{
				`field "example.agent.call_duration" of type histogram can't be a dimension, allowed types for dimensions: constant_keyword, keyword, long, integer, short, byte, double, float, half_float, scaled_float, unsigned_long, ip`,
			},
		},
		{
			title:           "bad time_series: ambiguous object cannot be a dimension",
			packageTemplate: "integration_v1_2_time_series",
			fields: []map[string]any{
				{
					"name":        "field_object",
					"type":        "object",
					"metric_type": "gauge",
				},
			},
			expectedErrors: []string{
				`field 0.type: 0.type must be one of the following: "histogram", "aggregate_metric_double", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long"`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			packagePath := createPackageWithFields(t, "testpackage", c.packageTemplate, c.fields)
			err := ValidateFromPath(packagePath)
			if len(c.expectedErrors) == 0 {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)

			errs, ok := err.(specerrors.ValidationErrors)
			require.True(t, ok)
			assert.Len(t, errs, len(c.expectedErrors))

			for _, foundError := range errs {
				// Trim the part of the error that refers to the file where the error was found.
				_, trimmedError, found := strings.Cut(foundError.Error(), ": ")
				if assert.True(t, found) {
					assert.Contains(t, c.expectedErrors, trimmedError)
				}
			}
		})
	}
}

func createPackageWithFields(t *testing.T, pkgName string, pkgTemplate string, fields any) string {
	var encodedFields []byte
	switch fields.(type) {
	case nil:
	case []byte:
		encodedFields = fields.([]byte)
	default:
		var jsonFields bytes.Buffer
		enc := yaml.NewEncoder(&jsonFields)
		err := enc.Encode(fields)
		require.NoError(t, err)
		encodedFields = jsonFields.Bytes()
	}

	return createPackageWithRawFields(t, pkgName, pkgTemplate, encodedFields)
}

func createPackageWithRawFields(t *testing.T, pkgName string, pkgTemplate string, rawFields []byte) string {
	basePackageDir := filepath.Join("testdata", "templates", pkgTemplate)
	require.DirExists(t, basePackageDir, "checking if template package exists")

	pkgDir := filepath.Join(t.TempDir(), pkgName)
	err := cp.Copy(basePackageDir, pkgDir)
	require.NoError(t, err, "copying template package")

	if len(rawFields) > 0 {
		fieldsPath := filepath.Join(pkgDir, "data_stream", "test", "fields", "fields.yml")
		err := os.WriteFile(fieldsPath, rawFields, 0644)
		require.NoError(t, err, "creating fields file")
	}

	return pkgDir
}
