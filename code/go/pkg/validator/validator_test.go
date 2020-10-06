package validator

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/code/go/internal/validator"
)

func TestValidate(t *testing.T) {
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
				for idx, err := range vErrs {
					expectedErr := errPrefix + test.expectedErrContains[idx]
					require.Contains(t, err.Error(), expectedErr)
				}
			}
		})
	}
}
