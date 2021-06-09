// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFieldGroups_Good(t *testing.T) {
	pkgRoot := "../../../../../test/packages/good"

	errs := ValidateFieldGroups(pkgRoot)
	require.Empty(t, errs)
}

func TestValidateFieldGroups_Bad(t *testing.T) {
	pkgRoot := "../../../../../test/packages/bad_group_unit"

	errs := ValidateFieldGroups(pkgRoot)
	require.Len(t, errs, 3)
	require.Equal(t, `field "bbb" can't have unit property'`, errs[0].Error())
	require.Equal(t, `field "eee" can't have unit property'`, errs[1].Error())
	require.Equal(t, `field "fff" can't have metric type property'`, errs[2].Error())
}