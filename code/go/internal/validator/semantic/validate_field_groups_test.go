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

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func TestValidateFieldGroups_Good(t *testing.T) {
	pkgRoot := "../../../../../test/packages/good"

	errs := ValidateFieldGroups(fspath.DirFS(pkgRoot))
	require.Empty(t, errs)
}

func TestValidateFieldGroups_Bad(t *testing.T) {
	pkgRoot := "../../../../../test/packages/bad_group_unit"

	fileError := func(name string, expected string) string {
		return fmt.Sprintf(`file "%s" is invalid: %s`,
			filepath.Join(pkgRoot, name),
			expected)
	}

	errs := ValidateFieldGroups(fspath.DirFS(pkgRoot))
	if assert.Len(t, errs, 3) {
		assert.Equal(t, fileError("data_stream/bar/fields/hello-world.yml", `field "aaa.bbb" can't have unit property'`), errs[0].Error())
		assert.Equal(t, fileError("data_stream/bar/fields/hello-world.yml", `field "ddd.eee" can't have unit property'`), errs[1].Error())
		assert.Equal(t, fileError("data_stream/foo/fields/bad-file.yml", `field "fff" can't have metric type property'`), errs[2].Error())
	}
}
