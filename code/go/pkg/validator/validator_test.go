package validator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	pkgRootPath := "../../internal/validator/test/packages/good"
	errs := ValidateFromPath(pkgRootPath)
	require.Len(t, errs, 0)
	fmt.Println(errs)
}
