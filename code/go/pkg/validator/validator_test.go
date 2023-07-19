// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/validator/common"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFile(t *testing.T) {
	// Workaround for error messages that contain OS-dependant base paths.
	osTestBasePath := filepath.Join("..", "..", "..", "..", "test", "packages") + string(filepath.Separator)

	tests := map[string]struct {
		invalidPkgFilePath  string
		expectedErrContains []string
	}{
		"good":                               {},
		"good_v2":                            {},
		"good_input":                         {},
		"deploy_custom_agent":                {},
		"deploy_custom_agent_multi_services": {},
		"deploy_docker":                      {},
		"deploy_terraform":                   {},
		"time_series":                        {},
		"missing_data_stream":                {},
		"icons_dark_mode":                    {},
		"ignored_malformed":                  {},
		"custom_ilm_policy":                  {},
		"profiling_symbolizer":               {},
		"bad_additional_content": {
			"bad-bad",
			[]string{
				"directory name inside package bad_additional_content contains -: bad-bad",
			},
		},
		"bad_deploy_variants": {
			"_dev/deploy/variants.yml",
			[]string{
				"field (root): default is required",
				"field variants: Invalid type. Expected: object, given: array",
			},
		},
		"missing_pipeline_dashes": {
			"data_stream/foo/elasticsearch/ingest_pipeline/default.yml",
			[]string{
				"document dashes are required (start the document with '---')",
			},
		},
		"missing_image_files": {
			"manifest.yml",
			[]string{
				"field screenshots.0.src: relative path is invalid, target doesn't exist or it exceeds the file size limit",
				"field icons.0.src: relative path is invalid, target doesn't exist or it exceeds the file size limit",
			},
		},
		"integration_benchmarks": {},
		"input_template":         {},
		"input_groups":           {},
		"input_groups_bad_data_stream": {
			"manifest.yml",
			[]string{
				"field policy_templates.2.data_streams.1: data stream doesn't exist",
			},
		},
		"bad_github_owner": {
			"manifest.yml",
			[]string{
				"field owner.github: Does not match pattern '^(([a-zA-Z0-9-]+)|([a-zA-Z0-9-]+\\/[a-zA-Z0-9-]+))$'",
			},
		},
		"missing_version": {
			"manifest.yml",
			[]string{
				"package version undefined in the package manifest file",
			},
		},
		"bad_aggregate_metric_double": {
			"data_stream/foo/fields/fields.yml",
			[]string{
				`field 0: metrics is required`,
				`field 1: default_metric is required`,
				`field 2.metrics.2: 2.metrics.2 must be one of the following: "min", "max", "sum", "value_count", "avg"`,
				`field 3: Must not be present`,
				`field 3: Must not be present`,
			},
		},
		"bad_time_series": {
			"data_stream/example/fields/fields.yml",
			[]string{
				`field 0.fields.4.type: 0.fields.4.type must be one of the following: "histogram", "aggregate_metric_double", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long"`,
				`field 0.fields.5: type is required`,
				`field 0.fields.8.type: 0.fields.8.type must be one of the following: "histogram", "aggregate_metric_double", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long"`,
				"field \"example.agent.call_duration\" of type histogram can't be a dimension, allowed types for dimensions: constant_keyword, keyword, long, integer, short, byte, double, float, half_float, scaled_float, unsigned_long, ip",
			},
		},
		"bad_metric_type_fields": {
			"data_stream/example/fields/fields.yml",
			[]string{
				`field 0.fields.4.type: 0.fields.4.type must be one of the following: "histogram", "aggregate_metric_double", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long"`,
				`field 0.fields.5: type is required`,
				`field 0.fields.6.type: 0.fields.6.type must be one of the following: "object"`,
				`field 0.fields.7.object_type: 0.fields.7.object_type must be one of the following: "histogram", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long"`,
				`field 0.fields.8.type: 0.fields.8.type must be one of the following: "histogram", "aggregate_metric_double", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long"`,
				"field \"example.agent.call_duration\" of type histogram can't be a dimension, allowed types for dimensions: constant_keyword, keyword, long, integer, short, byte, double, float, half_float, scaled_float, unsigned_long, ip",
			},
		},
		"bad_fields": {
			"data_stream/foo/fields/fields.yml",
			[]string{
				`field 0.type: 0.type must be one of the following: "aggregate_metric_double", "alias", "histogram", "constant_keyword", "text", "match_only_text", "keyword", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "date", "date_nanos", "boolean", "binary", "integer_range", "float_range", "long_range", "double_range", "date_range", "ip_range", "group", "geo_point", "object", "ip", "nested", "flattened", "wildcard", "version", "unsigned_long"`,
				`field "my_custom_date" of type keyword can't set date_format. date_format is allowed for date field type only`,
			},
		},
		"deploy_custom_agent_invalid_property": {
			"_dev/deploy/agent/custom-agent.yml",
			[]string{
				"field services.docker-custom-agent: Must not be present",
			},
		},
		"invalid_field_for_version": {
			"manifest.yml",
			[]string{
				"field (root): Additional property license is not allowed",
			},
		},
		"bad_release_tag": {
			"manifest.yml",
			[]string{
				"field (root): Additional property release is not allowed",
			},
		},
		"bad_custom_ilm_policy": {
			"data_stream/test/manifest.yml",
			[]string{
				fmt.Sprintf("field ilm_policy: ILM policy \"logs-bad_custom_ilm_policy.test-notexists\" not found in package, expected definition in \"%sbad_custom_ilm_policy/data_stream/test/elasticsearch/ilm/notexists.json\"", osTestBasePath),
			},
		},
		"bad_select": {
			"data_stream/foo_stream/manifest.yml",
			[]string{
				"field streams.0.vars.1: options is required",
				"field streams.0.vars.2.options: Invalid type. Expected: array, given: null",
				"field streams.0.vars.3: Must not be present",
			},
		},
		"bad_profiling_symbolizer": {
			"data_stream/example/manifest.yml",
			[]string{
				"profiling data type cannot be used in GA packages",
			},
		},
		"bad_runtime_fields": {
			"data_stream/foo/fields/fields.yml",
			[]string{
				`field 0: Must not be present`,
				`field 1.runtime: Invalid type. Expected: string, given: integer`,
				`field 2: Must not be present`,
				`field 3: Must not be present`,
			},
		},
		"bad_secret_vars": {
			"manifest.yml",
			[]string{
				"field vars.0: Additional property secret is not allowed",
			},
		},
		"bad_lifecycle": {
			"data_stream/test/lifecycle.yml",
			[]string{
				"field (root): Additional property max_age is not allowed",
			},
		},
		"bad_saved_object_tags": {
			"kibana/tags.yml",
			[]string{
				`field 0.asset_types.11: 0.asset_types.11 must be one of the following: "dashboard", "visualization", "search", "map", "lens", "index_pattern", "security_rule", "csp_rule_template", "ml_module", "osquery_pack_asset", "osquery_saved_query"`,
				`field 0.asset_types.12: 0.asset_types.12 must be one of the following: "dashboard", "visualization", "search", "map", "lens", "index_pattern", "security_rule", "csp_rule_template", "ml_module", "osquery_pack_asset", "osquery_saved_query"`,
				`field 1.asset_ids.1: Invalid type. Expected: string, given: integer`,
				`field 2: text is required`,
				`field 3: asset_types is required`,
			},
		},
	}

	for pkgName, test := range tests {
		t.Run(pkgName, func(t *testing.T) {
			pkgRootPath := filepath.Join("..", "..", "..", "..", "test", "packages", pkgName)
			errPrefix := fmt.Sprintf("file \"%s/%s\" is invalid: ", pkgRootPath, test.invalidPkgFilePath)

			errs := ValidateFromPath(pkgRootPath)
			if test.expectedErrContains == nil {
				require.NoError(t, errs)
			} else {
				require.Error(t, errs)
				vErrs, ok := errs.(errors.ValidationErrors)
				if ok {
					require.Len(t, errs, len(test.expectedErrContains))
					var errMessages []string
					for _, vErr := range vErrs {
						errMessages = append(errMessages, vErr.Error())
					}

					for _, expectedErrMessage := range test.expectedErrContains {
						expectedErr := errPrefix + expectedErrMessage
						require.Contains(t, errMessages, expectedErr)
					}
					return
				}
				require.Equal(t, errs.Error(), test.expectedErrContains[0])
			}
		})
	}
}

func TestValidateItemNotAllowed(t *testing.T) {
	tests := map[string]map[string][]string{
		"wrong_kibana_filename": {
			"kibana/dashboard": []string{
				"b7e55b73-97cc-44fd-8555-d01b7e13e70d.json",
				"bad-ecs.json",
				"bad-foobar-ecs.json",
				"bad-Foobaz-ECS.json",
			},
			"kibana/map": []string{
				"06149856-cbc1-4988-a93a-815915c4408e.json",
				"another-package-map.json",
			},
			"kibana/search": []string{
				"691240b5-7ec9-4fd7-8750-4ef97944f960.json",
				"another-package-search.json",
			},
			"kibana/visualization": []string{
				"defa1bcc-1ab6-4069-adec-8c997b069a5e.json",
				"another-package-visualization.json",
			},
		},
	}

	for pkgName, invalidItemsPerFolder := range tests {
		t.Run(pkgName, func(t *testing.T) {
			requireErrorMessage(t, pkgName, invalidItemsPerFolder, "item [%s] is not allowed in folder [%s/%s]")
		})
	}
}

func TestValidateItemNotExpected(t *testing.T) {
	tests := map[string]map[string][]string{
		"docs_extra_files": {
			"docs": []string{
				".missing",
			},
		},
	}

	for pkgName, invalidItemsPerFolder := range tests {
		t.Run(pkgName, func(t *testing.T) {
			requireErrorMessage(t, pkgName, invalidItemsPerFolder, "item [%s] is not allowed in folder [%s/%s]")
		})
	}
}

func TestValidateBadKibanaIDs(t *testing.T) {
	tests := map[string]map[string][]string{
		"bad_kibana_ids": {
			"kibana/dashboard": []string{
				"bad_kibana_ids-bar-baz.json",
			},
			"kibana/security_rule": []string{
				"bad_kibana_ids-bar-baz.json",
			},
		},
	}

	for pkgName, invalidItemsPerFolder := range tests {
		t.Run(pkgName, func(t *testing.T) {
			pkgRootPath := filepath.Join("..", "..", "..", "..", "test", "packages", pkgName)

			errs := ValidateFromPath(pkgRootPath)
			require.Error(t, errs)
			vErrs, ok := errs.(errors.ValidationErrors)
			require.True(t, ok)

			var errMessages []string
			for _, vErr := range vErrs {
				errMessages = append(errMessages, vErr.Error())
			}

			var c int
			for itemFolder, invalidItems := range invalidItemsPerFolder {
				for _, invalidItem := range invalidItems {
					objectFilePath := path.Join(pkgRootPath, itemFolder, invalidItem)
					expected := fmt.Sprintf("kibana object file [%s] defines non-matching ID", objectFilePath)

					var found bool
					for _, e := range errMessages {
						if strings.HasPrefix(e, expected) {
							found = true
						}
					}
					require.True(t, found, "Missing item: "+expected)
					c++
				}
			}
			require.Equal(t, c, len(errMessages))
		})
	}
}

func TestValidateBadRuleIDs(t *testing.T) {
	tests := map[string]string{
		"bad_rule_ids": "kibana object ID [saved_object_id] should start with rule ID [rule_id]",
	}

	for pkgName, expectedError := range tests {
		t.Run(pkgName, func(t *testing.T) {
			errs := ValidateFromPath(filepath.Join("..", "..", "..", "..", "test", "packages", pkgName))
			require.Error(t, errs)
			vErrs, ok := errs.(errors.ValidationErrors)
			require.True(t, ok)

			var errMessages []string
			for _, vErr := range vErrs {
				errMessages = append(errMessages, vErr.Error())
			}
			require.Contains(t, errMessages, expectedError)
		})
	}
}

func TestValidateMissingRequiredFields(t *testing.T) {
	tests := map[string][]string{
		"good":    {},
		"good_v2": {},
		"missing_required_fields": {
			`expected type "constant_keyword" for required field "data_stream.dataset", found "keyword" in "../../../../test/packages/missing_required_fields/data_stream/foo/fields/base-fields.yml"`,
			`expected field "data_stream.type" with type "constant_keyword" not found in datastream "foo"`,
		},
		"missing_required_fields_input": {
			`expected type "constant_keyword" for required field "data_stream.dataset", found "keyword" in "../../../../test/packages/missing_required_fields_input/fields/base-fields.yml"`,
			`expected field "data_stream.type" with type "constant_keyword" not found`,
		},
	}

	for pkgName, expectedErrors := range tests {
		t.Run(pkgName, func(t *testing.T) {
			pkgRootPath := path.Join("..", "..", "..", "..", "test", "packages", pkgName)
			err := ValidateFromPath(pkgRootPath)
			if len(expectedErrors) == 0 {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)

			errs, ok := err.(errors.ValidationErrors)
			require.True(t, ok)
			assert.Len(t, errs, len(expectedErrors))

			for _, expectedError := range expectedErrors {
				found := false
				for _, foundError := range errs {
					if foundError.Error() == expectedError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error: %q (%v)", expectedError, errs)
				}
			}
		})
	}
}

func TestValidateVersionIntegrity(t *testing.T) {
	tests := map[string]string{
		"inconsistent_version": "current manifest version doesn't have changelog entry",
		"same_version_twice":   "versions in changelog must be unique, found at least two same versions (0.0.2)",
	}

	for pkgName, expectedErrorMessage := range tests {
		t.Run(pkgName, func(t *testing.T) {
			errs := ValidateFromPath(filepath.Join("..", "..", "..", "..", "test", "packages", pkgName))
			require.Error(t, errs)
			vErrs, ok := errs.(errors.ValidationErrors)
			require.True(t, ok)

			var errMessages []string
			for _, vErr := range vErrs {
				errMessages = append(errMessages, vErr.Error())
			}
			require.Contains(t, errMessages, expectedErrorMessage)
		})
	}
}

func TestValidateDuplicatedFields(t *testing.T) {
	tests := map[string]string{
		"bad_duplicated_fields":       "field \"event.dataset\" is defined multiple times for data stream \"wrong\", found in: ../../../../test/packages/bad_duplicated_fields/data_stream/wrong/fields/base-fields.yml, ../../../../test/packages/bad_duplicated_fields/data_stream/wrong/fields/ecs.yml",
		"bad_duplicated_fields_input": "field \"event.dataset\" is defined multiple times for data stream \"\", found in: ../../../../test/packages/bad_duplicated_fields_input/fields/base-fields.yml, ../../../../test/packages/bad_duplicated_fields_input/fields/ecs.yml",
	}

	for pkgName, expectedErrorMessage := range tests {
		t.Run(pkgName, func(t *testing.T) {
			errs := ValidateFromPath(path.Join("..", "..", "..", "..", "test", "packages", pkgName))
			require.Error(t, errs)
			vErrs, ok := errs.(errors.ValidationErrors)
			require.True(t, ok)

			assert.Len(t, vErrs, 1)

			var errMessages []string
			for _, vErr := range vErrs {
				errMessages = append(errMessages, vErr.Error())
			}
			require.Contains(t, errMessages, expectedErrorMessage)
		})
	}

}

func TestValidateMinimumKibanaVersions(t *testing.T) {
	tests := map[string][]string{
		"good":       []string{},
		"good_input": []string{},
		"good_v2":    []string{},
		"custom_logs": []string{
			"conditions.kibana.version must be ^8.8.0 or greater for non experimental input packages (version > 1.0.0)",
		},
		"httpjson_input": []string{
			"conditions.kibana.version must be ^8.8.0 or greater for non experimental input packages (version > 1.0.0)",
		},
		"sql_input": []string{
			"conditions.kibana.version must be ^8.8.0 or greater for non experimental input packages (version > 1.0.0)",
		},
		"bad_runtime_kibana_version": []string{
			"conditions.kibana.version must be ^8.10.0 or greater to include runtime fields",
		},
		"bad_saved_object_tags_kibana_version": []string{
			"conditions.kibana.version must be ^8.10.0 or greater to include saved object tags file: kibana/tags.yml",
		},
	}

	for pkgName, expectedErrorMessages := range tests {
		t.Run(pkgName, func(t *testing.T) {
			err := ValidateFromPath(path.Join("..", "..", "..", "..", "test", "packages", pkgName))
			if len(expectedErrorMessages) == 0 {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)

			errs, ok := err.(errors.ValidationErrors)
			require.True(t, ok)
			assert.Len(t, errs, len(expectedErrorMessages))

			for _, expectedError := range expectedErrorMessages {
				found := false
				for _, foundError := range errs {
					if foundError.Error() == expectedError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error: %q, found: %q", expectedError, errs)
				}
			}
		})
	}

}

func TestValidateWarnings(t *testing.T) {
	tests := map[string][]string{
		"good":    []string{},
		"good_v2": []string{},
		"visualizations_by_reference": []string{
			"references found in dashboard kibana/dashboard/visualizations_by_reference-82273ffe-6acc-4f2f-bbee-c1004abba63d.json: visualizations_by_reference-5e1a01ff-6f9a-41c1-b7ad-326472db42b6 (visualization), visualizations_by_reference-8287a5d5-1576-4f3a-83c4-444e9058439b (lens)",
		},
	}
	if err := common.EnableWarningsAsErrors(); err != nil {
		require.NoError(t, err)
	}
	defer common.DisableWarningsAsErrors()

	for pkgName, expectedWarnContains := range tests {
		t.Run(pkgName, func(t *testing.T) {
			warnPrefix := fmt.Sprintf("Warning: ")

			pkgRootPath := path.Join("..", "..", "..", "..", "test", "packages", pkgName)
			errs := ValidateFromPath(pkgRootPath)
			if len(expectedWarnContains) == 0 {
				require.NoError(t, errs)
			} else {
				require.Error(t, errs)
				vErrs, ok := errs.(errors.ValidationErrors)
				if ok {
					require.Len(t, errs, len(expectedWarnContains))
					var warnMessages []string
					for _, vErr := range vErrs {
						warnMessages = append(warnMessages, vErr.Error())
					}

					for _, expectedWarnMessage := range expectedWarnContains {
						expectedWarn := warnPrefix + expectedWarnMessage
						require.Contains(t, warnMessages, expectedWarn)
					}
					return
				}
				require.Equal(t, errs.Error(), expectedWarnContains[0])
			}
		})
	}
}

func TestValidateExternalFieldsWithoutDevFolder(t *testing.T) {

	pkgName := "bad_external_without_dev_build"
	tests := []struct {
		title               string
		invalidPkgFilePath  string
		buildFileContents   string
		expectedErrContains []string
	}{
		{
			"valid definition",
			"data_stream/foo/fields/ecs.yml",
			"dependencies:\n  ecs:\n    reference: git@v8.6.0\n",
			[]string{},
		},
		{
			"build file not exist",
			"data_stream/foo/fields/ecs.yml",
			"",
			[]string{
				"field container.id with external key defined (\"ecs\") but no _dev/build/build.yml found",
			},
		},
		{
			"ecs not defined",
			"data_stream/foo/fields/ecs.yml",
			"dependencies: {}\n",
			[]string{
				"field container.id with external key defined (\"ecs\") but no definition found for it (_dev/build/build.yml)",
			},
		},
	}

	pkgRootPath := filepath.Join("..", "..", "..", "..", "test", "packages", pkgName)

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			tempDir := t.TempDir()

			devFolderPath := filepath.Join(tempDir, "_dev")
			buildFolderPath := filepath.Join(devFolderPath, "build")
			buildFilePath := filepath.Join(buildFolderPath, "build.yml")

			errPrefix := fmt.Sprintf("file \"%s/%s\" is invalid: ", tempDir, test.invalidPkgFilePath)

			err := cp.Copy(pkgRootPath, tempDir)
			require.NoError(t, err)

			err = os.RemoveAll(devFolderPath)
			require.NoError(t, err)

			if test.buildFileContents != "" {
				err := os.MkdirAll(buildFolderPath, 0755)
				require.NoError(t, err)

				err = os.WriteFile(buildFilePath, []byte(test.buildFileContents), 0644)
				require.NoError(t, err)
			}

			errs := ValidateFromPath(tempDir)
			if len(test.expectedErrContains) == 0 {
				require.NoError(t, errs)
			} else {
				require.Error(t, errs)
				vErrs, ok := errs.(errors.ValidationErrors)
				if ok {
					require.Len(t, errs, len(test.expectedErrContains))
					var errMessages []string
					for _, vErr := range vErrs {
						errMessages = append(errMessages, vErr.Error())
					}

					for _, expectedErrMessage := range test.expectedErrContains {
						expectedErr := errPrefix + expectedErrMessage
						require.Contains(t, errMessages, expectedErr)
					}
					return
				}
				require.Equal(t, errs.Error(), test.expectedErrContains[0])
			}
		})
	}
}

func TestValidateRoutingRules(t *testing.T) {
	tests := map[string][]string{
		"good":    []string{},
		"good_v2": []string{},
		"bad_routing_rules": []string{
			`routing rules defined in data stream "rules" but dataset field is missing: dataset field is required in data stream "rules"`,
		},
		"bad_routing_rules_wrong_spec": []string{
			`item [routing_rules.yml] is not allowed in folder [../../../../test/packages/bad_routing_rules_wrong_spec/data_stream/rules]`,
		},
	}

	for pkgName, expectedErrorMessages := range tests {
		t.Run(pkgName, func(t *testing.T) {
			err := ValidateFromPath(path.Join("..", "..", "..", "..", "test", "packages", pkgName))
			if len(expectedErrorMessages) == 0 {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)

			errs, ok := err.(errors.ValidationErrors)
			require.True(t, ok)
			assert.Len(t, errs, len(expectedErrorMessages))

			for _, foundError := range errs {
				require.Contains(t, expectedErrorMessages, foundError.Error())
			}
		})
	}

}

func requireErrorMessage(t *testing.T, pkgName string, invalidItemsPerFolder map[string][]string, expectedErrorMessage string) {
	pkgRootPath := filepath.Join("..", "..", "..", "..", "test", "packages", pkgName)

	errs := ValidateFromPath(pkgRootPath)
	require.Error(t, errs)
	vErrs, ok := errs.(errors.ValidationErrors)
	require.True(t, ok)

	var errMessages []string
	for _, vErr := range vErrs {
		errMessages = append(errMessages, vErr.Error())
	}

	var c int
	for itemFolder, invalidItems := range invalidItemsPerFolder {
		for _, invalidItem := range invalidItems {
			c++
			expected := fmt.Sprintf(expectedErrorMessage, invalidItem, pkgRootPath, itemFolder)
			require.Contains(t, errMessages, expected)
		}
	}
	require.Len(t, errs, c)
}
