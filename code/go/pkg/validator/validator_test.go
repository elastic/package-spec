package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	pkgRootPath := "../../internal/validator/test/packages/good"
	errs := ValidateFromPath(pkgRootPath)
	require.NoError(t, errs)
}
