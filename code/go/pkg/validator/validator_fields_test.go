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
		{
			title:           "base integration-v3",
			packageTemplate: "integration_v3_0",
			fields:          nil,
			expectedErrors:  nil,
		},
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

		// bad_aggregate_metric_double
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
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			packagePath := createPackageWithFields(t, "testpackage", c.packageTemplate, c.fields)
			err := ValidateFromPath(packagePath)
			if len(c.expectedErrors) == 0 {
				assert.NoError(t, err)
				return
			}

			assert.Error(t, err)

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
