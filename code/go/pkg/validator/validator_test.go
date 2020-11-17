package validator

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/code/go/internal/validator"
)

func TestValidateFile(t *testing.T) {
	tests := map[string]struct {
		invalidPkgFilePath  string
		expectedErrContains []string
	}{
		"good":          {},
		"deploy_docker": {},
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
		"missing_image_files":          {
			"manifest.yml",
			[]string{
				"field screenshots.0.src: relative path is invalid or target doesn't exist",
				"field icons.0.src: relative path is invalid or target doesn't exist",
			},
		},
	}

	for pkgName, test := range tests {
		t.Run(pkgName, func(t *testing.T) {
			pkgRootPath := filepath.Join("..", "..", "internal", "validator", "test", "packages", pkgName)
			errPrefix := fmt.Sprintf("file \"%s/%s\" is invalid: ", pkgRootPath, test.invalidPkgFilePath)

			errs := ValidateFromPath(pkgRootPath)
			if test.expectedErrContains == nil {
				require.NoError(t, errs)
			} else {
				require.Error(t, errs)
				require.Len(t, errs, len(test.expectedErrContains))
				vErrs, ok := errs.(validator.ValidationErrors)
				require.True(t, ok)

				var errMessages []string
				for _, vErr := range vErrs {
					errMessages = append(errMessages, vErr.Error())
				}

				for _, expectedErrMessage := range test.expectedErrContains {
					expectedErr := errPrefix + expectedErrMessage
					require.Contains(t, errMessages, expectedErr)
				}
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
			pkgRootPath := filepath.Join("..", "..", "internal", "validator", "test", "packages", pkgName)

			errs := ValidateFromPath(pkgRootPath)
			require.Error(t, errs)
			vErrs, ok := errs.(validator.ValidationErrors)
			require.True(t, ok)

			var errMessages []string
			for _, vErr := range vErrs {
				errMessages = append(errMessages, vErr.Error())
			}

			var c int
			for itemFolder, invalidItems := range invalidItemsPerFolder {
				for _, invalidItem := range invalidItems {
					c++
					expected := fmt.Sprintf("item [%s] is not allowed in folder [%s/%s]", invalidItem, pkgRootPath, itemFolder)
					require.Contains(t, errMessages, expected)
				}
			}
			require.Len(t, errs, c)
		})
	}
}
