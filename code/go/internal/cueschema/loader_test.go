// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cueschema

import (
	"testing"

	spec "github.com/elastic/package-spec"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/stretchr/testify/require"
)

func TestLoadIntegrationManifest(t *testing.T) {
	loader := NewFileSchemaLoader()
	options := spectypes.FileSchemaLoadOptions{}
	_, err := loader.Load(spec.FS(), "integration/manifest.spec.cue", options)
	require.NoError(t, err)
}
