// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFieldGroups_Good(t *testing.T) {
	pkgRoot := "../../../../../test/packages/good"

	errs := ValidateFieldGroups(pkgRoot)
	require.Empty(t, errs)
}

func TestValidateFieldGroups_Bad(t *testing.T) {
	pkgRoot := "../../../../../test/packages/bad_group_unit"

	fileError := func(name string, expected string) string {
		return fmt.Sprintf(`file "%s" is invalid: %s`,
			filepath.Join(pkgRoot, name),
			expected)
	}

	errs := ValidateFieldGroups(pkgRoot)
	if assert.Len(t, errs, 3) {
		assert.Equal(t, fileError("data_stream/bar/fields/hello-world.yml", `field "bbb" can't have unit property'`), errs[0].Error())
		assert.Equal(t, fileError("data_stream/bar/fields/hello-world.yml", `field "eee" can't have unit property'`), errs[1].Error())
		assert.Equal(t, fileError("data_stream/foo/fields/bad-file.yml", `field "fff" can't have metric type property'`), errs[2].Error())
	}
}
