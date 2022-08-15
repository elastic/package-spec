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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFile(t *testing.T) {
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
		"custom_logs":         {},
		"httpjson_input":      {},
		"sql_input":           {},
		"icons_dark_mode":     {},
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
		/* FIXME:
		"missing_image_files": {
			"manifest.yml",
			[]string{
				"field screenshots.0.src: relative path is invalid, target doesn't exist or it exceeds the file size limit",
				"field icons.0.src: relative path is invalid, target doesn't exist or it exceeds the file size limit",
			},
		},
		*/
		"input_template": {},
		"input_groups":   {},
		/* FIXME:
		"input_groups_bad_data_stream": {
			"manifest.yml",
			[]string{
				"field policy_templates.2.data_streams.1: data stream doesn't exist",
			},
		},
		*/
		"bad_github_owner": {
			"manifest.yml",
			[]string{
				"field owner.github: Does not match pattern '^(([a-zA-Z0-9-]+)|([a-zA-Z0-9-]+/[a-zA-Z0-9-]+))$'",
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
		"deploy_custom_agent_invalid_property": {
			"_dev/deploy/agent/custom-agent.yml",
			[]string{
				"field services.docker-custom-agent: Must not validate the schema (not)",
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
