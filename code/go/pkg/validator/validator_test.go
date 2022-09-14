// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/validator/common"

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
		"good":                {},
		"deploy_custom_agent": {},
		"deploy_docker":       {},
		"deploy_terraform":    {},
		"time_series":         {},
		"missing_data_stream": {},
		"icons_dark_mode":     {},
		"custom_ilm_policy":   {},
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
		"input_template": {},
		"input_groups":   {},
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
		"bad_time_series": {
			"data_stream/example/fields/fields.yml",
			[]string{
				"field \"example.agent.call_duration\" of type histogram can't be a dimension, allowed types for dimensions: constant_keyword, keyword, long, integer, short, byte, double, float, half_float, scaled_float, unsigned_long, ip",
			},
		},
		"bad_fields": {
			"data_stream/foo/fields/fields.yml",
			[]string{
				`field 0.type: 0.type must be one of the following: "alias", "histogram", "constant_keyword", "text", "match_only_text", "keyword", "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "date", "date_nanos", "boolean", "binary", "integer_range", "float_range", "long_range", "double_range", "date_range", "ip_range", "group", "geo_point", "object", "ip", "nested", "flattened", "wildcard", "version", "unsigned_long"`,
			},
		},
		"deploy_custom_agent_invalid_property": {
			"_dev/deploy/agent/custom-agent.yml",
			[]string{
				"field services.docker-custom-agent: Must not validate the schema (not)",
			},
		},
		"invalid_field_for_version": {
			"manifest.yml",
			[]string{
				"field (root): Additional property license is not allowed",
			},
		},
		"bad_custom_ilm_policy": {
			"data_stream/test/manifest.yml",
			[]string{
				fmt.Sprintf("field ilm_policy: ILM policy \"logs-bad_custom_ilm_policy.test-notexists\" not found in package, expected definition in \"%sbad_custom_ilm_policy/data_stream/test/elasticsearch/ilm/notexists.json\"", osTestBasePath),
			},
		},
		"bad_assets_elastic_snapshot_versions":        {},
		"assets_elastic_versions_allowed_snapshot":    {},
		"assets_elastic_snapshot_versions_prerelease": {},
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

func TestValidateMissingReqiredFields(t *testing.T) {
	tests := map[string][]string{
		"good": {},
		"missing_required_fields": {
			`expected type "constant_keyword" for required field "data_stream.dataset", found "keyword" in "../../../../test/packages/missing_required_fields/data_stream/foo/fields/base-fields.yml"`,
			`expected field "data_stream.type" with type "constant_keyword" not found in datastream "foo"`,
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
					t.Errorf("expected error: %q", expectedError)
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
		"bad_duplicated_fields": "field \"event.dataset\" is defined multiple times for data stream \"wrong\", found in: ../../../../test/packages/bad_duplicated_fields/data_stream/wrong/fields/base-fields.yml, ../../../../test/packages/bad_duplicated_fields/data_stream/wrong/fields/ecs.yml",
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

func TestValidateWarnings(t *testing.T) {
	tests := map[string][]string{
		"good": []string{},
		"custom_logs": []string{
			"package with non-stable semantic version and active beta features (enabled in [../../../../test/packages/custom_logs]) can't be released as stable version.",
		},
		"httpjson_input": []string{
			"package with non-stable semantic version and active beta features (enabled in [../../../../test/packages/httpjson_input]) can't be released as stable version.",
		},
		"sql_input": []string{
			"package with non-stable semantic version and active beta features (enabled in [../../../../test/packages/sql_input]) can't be released as stable version.",
		},
		"visualizations_by_reference": []string{
			"references found in dashboard kibana/dashboard/visualizations_by_reference-82273ffe-6acc-4f2f-bbee-c1004abba63d.json: visualizations_by_reference-5e1a01ff-6f9a-41c1-b7ad-326472db42b6 (visualization), visualizations_by_reference-8287a5d5-1576-4f3a-83c4-444e9058439b (lens)",
		},
		"bad_assets_elastic_snapshot_versions": []string{
			"prerelease version found in dashboard kibana/dashboard/bad_assets_elastic_snapshot_versions-overview.json: 8.4.1-SNAPSHOT",
			"prerelease version found in visualization kibana/visualization/bad_assets_elastic_snapshot_versions-visualization1.json: 8.4.1-SNAPSHOT",
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
