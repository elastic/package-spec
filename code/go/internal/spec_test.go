// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	spec "github.com/elastic/package-spec"
)

func TestBundledSpecsForIntegration(t *testing.T) {
	fs := spec.FS()
	_, err := fs.Open("integration/spec.yml")
	require.NoError(t, err)
}

func TestBundledSpecsForInput(t *testing.T) {
	fs := spec.FS()
	_, err := fs.Open("input/spec.yml")
	require.NoError(t, err)
}
