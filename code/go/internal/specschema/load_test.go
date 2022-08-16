// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specschema

import (
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/elastic/package-spec/code/go/internal/yamlschema"
)

func TestLoadFolderSpec(t *testing.T) {
	fileSpecLoader := yamlschema.NewFileSchemaLoader()
	loader := NewFolderSpecLoader(os.DirFS("./testdata"), fileSpecLoader, semver.Version{})
	spec, err := loader.Load("simple-spec")
	require.NoError(t, err)

	assert.True(t, spec.AdditionalContents(), "additionalContents")
	assert.Equal(t, 250*spectypes.MegaByte, spec.MaxTotalSize(), "maxTotalSize")
	assert.Equal(t, 150*spectypes.MegaByte, spec.MaxFileSize(), "maxFileSize")
	assert.Len(t, spec.Contents(), 2, "number of contents")

	for _, content := range spec.Contents() {
		switch content.Name() {
		case "manifest.yml":
			t.Run("manifest.yml", func(t *testing.T) {
				item := content.(*ItemSpec).itemSpec
				assert.NotNil(t, item.schema)
			})
		case "docs":
			t.Run("docs", func(t *testing.T) {
				assert.False(t, content.AdditionalContents(), "additionalContents")
				assert.Equal(t, 100*spectypes.MegaByte, content.MaxTotalSize(), "maxTotalSize")
				assert.Equal(t, 150*spectypes.MegaByte, content.MaxFileSize(), "maxFileSize")
				assert.Len(t, content.Contents(), 2)
			})
		default:
			t.Errorf("Unexpected content in the spec with name %q", content.Name())
		}
	}
}
