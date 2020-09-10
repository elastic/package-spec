package validator

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/code/go/internal/validator"
)

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		expectedErrContains []string
	}{
		"good": {},
		"bad_deploy_variants": {
			expectedErrContains: []string{
				"field (root): default is required",
				"field variants: Invalid type. Expected: object, given: array",
			},
		},
	}

	for pkgName, test := range tests {
		t.Run(pkgName, func(t *testing.T) {
			pkgRootPath := filepath.Join("..", "..", "internal", "validator", "test", "packages", pkgName)
			errs := ValidateFromPath(pkgRootPath)
			if test.expectedErrContains == nil {
				require.NoError(t, errs)
			} else {
				require.Error(t, errs)
				require.Len(t, errs, len(test.expectedErrContains))
				vErrs, ok := errs.(validator.ValidationErrors)
				require.True(t, ok)
				for idx, err := range vErrs {
					expectedErr := test.expectedErrContains[idx]
					require.Contains(t, err.Error(), expectedErr)
				}
			}
		})
	}
}
