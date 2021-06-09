// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"testing"

	"github.com/rakyll/statik/fs"
	"github.com/stretchr/testify/require"

	_ "github.com/elastic/package-spec/code/go/internal/spec"
)

func TestBundledSpecs(t *testing.T) {
	bundle, err := fs.New()
	require.NoError(t, err)

	_, err = bundle.Open("/1/spec.yml")
	require.NoError(t, err)
}
