package semantic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFieldGroupWithoutUnit_Good(t *testing.T) {
	pkgRoot := "../../../../../test/packages/good"

	errs := ValidateFieldGroupWithoutUnit(pkgRoot)
	require.Empty(t, errs)
}

func TestValidateFieldGroupWithoutUnit_Bad(t *testing.T) {
	pkgRoot := "../../../../../test/packages/bad_group_unit"

	errs := ValidateFieldGroupWithoutUnit(pkgRoot)
	require.Len(t, errs, 3)
	require.Equal(t, `field "bbb" can't have unit property'`, errs[0].Error())
	require.Equal(t, `field "eee" can't have unit property'`, errs[1].Error())
	require.Equal(t, `field "fff" can't have unit property'`, errs[2].Error())
}