// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	spec "github.com/elastic/package-spec/v3"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

func TestValidateFields(t *testing.T) {
	timeSeriesPatch := patch{
		op:   "append",
		file: filepath.Join("data_stream", "test", "manifest.yml"),
		content: map[string]any{
			"elasticsearch": map[string]any{
				"source_mode": "synthetic",
				"index_mode":  "time_series",
			},
		},
	}

	cases := []struct {
		title           string
		packageTemplate string
		specVersion     *semver.Version
		patches         []patch
		fields          any
		expectedErrors  []string
	}{
		// Check that base templates are fine on their own.
		{
			title:           "base integration_v1",
			packageTemplate: "integration_v1",
		},
		{
			title:           "base integration_v1_2_time_series",
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
		},
		{
			title:           "base integration_v3",
			packageTemplate: "integration_v3",
		},
		{
			title:           "base integration_v3_0_2",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.0.2"),
		},
		{
			title:           "base integration_v3_0_3_time_series",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.0.3"),
			patches:         []patch{timeSeriesPatch},
			// This package needs at least a dimension field with this patch.
			fields: []map[string]any{
				{
					"name":      "afield",
					"type":      "keyword",
					"dimension": true,
				},
			},
		},
		{
			title:           "base integration_v3_1",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.1.0"),
		},

		// Generic validations.
		{
			title:           "unknown type",
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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

			title:           "enabled object without object type checked since 3.0.3",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.0.3"),
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

			title:           "disabled object with object type is disallowed since 3.0.3",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.0.3"),
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
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.0.2"),
			fields: []map[string]any{
				{
					"name":        "user_provided_metadata",
					"type":        "object",
					"object_type": "keyword",
					"enabled":     false,
				},
			},
		},

		// Disabling subobjects.
		{
			title:           "disabled subobjects with wildcard",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.1.0"),
			fields: []map[string]any{
				{
					"name":        "prometheus.b.labels.*",
					"type":        "object",
					"object_type": "keyword",
					"subobjects":  false,
				},
			},
		},
		{
			title:           "disabled subobjects without wildcard",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.1.0"),
			fields: []map[string]any{
				{
					"name":        "prometheus.b.labels",
					"type":        "object",
					"object_type": "keyword",
					"subobjects":  false,
				},
			},
		},
		{
			title:           "disabled subobjects cannot be on type group",
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.1.0"),
			fields: []map[string]any{
				{
					"name":       "prometheus.b.labels",
					"type":       "group",
					"subobjects": false,
				},
			},
			expectedErrors: []string{
				`field 0.type: 0.type must be one of the following: "object"`,
			},
		},

		// aggregate_metric_double
		{
			title:           "aggregate_metric_double",
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v3",
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
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
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
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
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
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
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
			packageTemplate: "integration_v3",
			specVersion:     semver.MustParse("3.0.3"),
			patches:         []patch{timeSeriesPatch},
			expectedErrors: []string{
				"time series mode enabled but no dimensions configured",
			},
		},
		{
			title:           "bad time_series: invalid type for time series",
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
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
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
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
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
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
			packageTemplate: "integration_v1",
			specVersion:     semver.MustParse("1.2.0"),
			patches:         []patch{timeSeriesPatch},
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

		// Runtime fields
		{
			title:           "runtime fields: wrong type",
			packageTemplate: "integration_v3",
			fields: []map[string]any{
				{
					"name":    "runtime_boolean",
					"type":    "boolean",
					"runtime": true,
				},
				{
					"name":    "runtime_script",
					"type":    "keyword",
					"runtime": "doc['message'].value().doSomething()",
				},
			},
		},
		{
			title:           "runtime fields: wrong type",
			packageTemplate: "integration_v3",
			fields: []map[string]any{
				{
					"name":    "runtime_field",
					"type":    "byte",
					"runtime": true,
				},
			},
			expectedErrors: []string{
				`field 0: Must not be present`,
			},
		},
		{
			title:           "runtime fields: invalid runtime value",
			packageTemplate: "integration_v3",
			fields: []map[string]any{
				{
					"name":    "runtime_field",
					"type":    "boolean",
					"runtime": 30,
				},
			},
			expectedErrors: []string{
				`field 0.runtime: Invalid type. Expected: string, given: integer`,
			},
		},
		{
			title:           "runtime fields: invalid runtime boolean",
			packageTemplate: "integration_v3",
			fields: []map[string]any{
				{
					"name":    "runtime_field",
					"type":    "binary",
					"runtime": true,
				},
			},
			expectedErrors: []string{
				`field 0: Must not be present`,
			},
		},
		{

			title:           "runtime fields: invalid runtime boolean",
			packageTemplate: "integration_v3",
			fields: []map[string]any{
				{
					"name":    "runtime_field",
					"type":    "binary",
					"runtime": "doc['message'].value().doSomething()",
				},
			},
			expectedErrors: []string{
				`field 0: Must not be present`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			packagePath := createPackageWithFields(t, "testpackage", c.packageTemplate, c.specVersion, c.patches, c.fields)
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

// encodeContent encodes the content as YAML, unless it is already an array of bytes, that is returned as is.
func encodeContent(content any) ([]byte, error) {
	switch content.(type) {
	case nil:
		return nil, nil
	case []byte:
		return content.([]byte), nil
	default:
		var jsonFields bytes.Buffer
		enc := yaml.NewEncoder(&jsonFields)
		enc.SetIndent(2)
		err := enc.Encode(content)
		if err != nil {
			return nil, err
		}
		return jsonFields.Bytes(), nil
	}
}

func createPackageWithFields(t *testing.T, pkgName string, pkgTemplate string, specVersion *semver.Version, patches []patch, fields any) string {
	encodedFields, err := encodeContent(fields)
	require.NoError(t, err)

	return createPackageWithRawFields(t, pkgName, pkgTemplate, specVersion, patches, encodedFields)
}

func createPackageWithRawFields(t *testing.T, pkgName string, pkgTemplate string, specVersion *semver.Version, patches []patch, rawFields []byte) string {
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

	if specVersion != nil {
		err := overrideSpecVersion(pkgDir, specVersion)
		require.NoError(t, err)
	}

	for _, patch := range patches {
		err := patch.apply(pkgDir)
		require.NoError(t, err)
	}

	return pkgDir
}

func overrideSpecVersion(pkgDir string, version *semver.Version) error {
	if version == nil {
		return nil
	}

	manifestPath := filepath.Join(pkgDir, "manifest.yml")
	d, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	versionRegexp := regexp.MustCompile(`(\n|^)format_version: [^\s]+`)
	d = versionRegexp.ReplaceAll(d, []byte(fmt.Sprintf(`format_version: "%s"`, version.String())))
	err = os.WriteFile(manifestPath, d, 0644)
	if err != nil {
		return err
	}

	// Check if package is using an unreleased version of the spec ensure that it is a non GA version.
	if !version.LessThan(semver.MustParse("3.0.1")) {
		specVersion, err := spec.CheckVersion(*version)
		if err != nil {
			return err
		}
		if specVersion.Prerelease() != "" {
			err = setPrereleasePackageVersion(pkgDir, "rc1")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func setPrereleasePackageVersion(pkgDir string, prerelease string) error {
	// Update version in manifest.
	manifestPath := filepath.Join(pkgDir, "manifest.yml")
	d, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	pkgVersionRegexp := regexp.MustCompile(`(\n|^)version: ([^\s]+)`)
	submatch := pkgVersionRegexp.FindSubmatch(d)
	if len(submatch) < 3 {
		return errors.New("no version in manifest?")
	}
	pkgVersion := semver.MustParse(string(submatch[2]))
	prereleaseVersion, err := pkgVersion.SetPrerelease(prerelease)
	if err != nil {
		return err
	}
	d = pkgVersionRegexp.ReplaceAll(d, []byte(fmt.Sprintf(`%sversion: "%s"`, submatch[1], prereleaseVersion.String())))
	err = os.WriteFile(manifestPath, d, 0644)
	if err != nil {
		return err
	}

	// Update version in changelog.
	changelogPath := filepath.Join(pkgDir, "changelog.yml")
	changelog, err := os.ReadFile(changelogPath)
	if err != nil {
		return err
	}
	changelogRegexp := regexp.MustCompile(fmt.Sprintf(`-\s+version:\s+"?%s"?`, pkgVersion.String()))
	changelog = changelogRegexp.ReplaceAll(changelog, []byte(fmt.Sprintf(`- version: "%s"`, prereleaseVersion.String())))

	return os.WriteFile(changelogPath, changelog, 0644)
}

type patch struct {
	op      string
	file    string
	content any
}

func (p patch) apply(pkgDir string) error {
	path := filepath.Join(pkgDir, p.file)

	switch p.op {
	case "append":
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return err
		}
		content, err := encodeContent(p.content)
		if err != nil {
			return err
		}
		_, err = f.WriteString("\n" + string(content))
		if err != nil {
			return err
		}
		defer f.Close()
	default:
		return fmt.Errorf("unknown patch operation %s", p.op)
	}

	return nil
}
