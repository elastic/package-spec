// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	spec "github.com/elastic/package-spec"
)

func TestBundledSpecs(t *testing.T) {
	fs := spec.FS()
	_, err := fs.Open("1/spec.yml")
	require.NoError(t, err)
}
