// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path"
	"testing"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetValidatedSharedKibanaTags(t *testing.T) {
	t.Run("no tags.yml file", func(t *testing.T) {
		tmpDir := t.TempDir()
		fsys := fspath.DirFS(tmpDir)

		tags, errs := getValidatedSharedKibanaTags(fsys)
		require.Empty(t, errs)
		assert.Empty(t, tags)
	})

	t.Run("with tags.yml file and duplicates", func(t *testing.T) {
		tmpDir := t.TempDir()
		kibanaDir := path.Join(tmpDir, "kibana")
		err := os.MkdirAll(kibanaDir, 0o755)
		require.NoError(t, err)

		tagsYMLPath := path.Join(kibanaDir, "tags.yml")
		tagsYMLContent := `- text: tag1
- text: tag2
- text: tag1
`
		err = os.WriteFile(tagsYMLPath, []byte(tagsYMLContent), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(tmpDir)
		tags, errs := getValidatedSharedKibanaTags(fsys)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "duplicate tag name 'tag1' found (SVR00007)")
		require.Len(t, tags, 2)
		assert.Contains(t, tags, "tag1")
		assert.Contains(t, tags, "tag2")
	})
}

func TestValidateKibanaPackageTagsDuplicates(t *testing.T) {
	t.Run("with duplicate tags in JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()
		kibanaTagDir := path.Join(tmpDir, "kibana", "tag")
		err := os.MkdirAll(kibanaTagDir, 0o755)
		require.NoError(t, err)

		tag1Path := path.Join(kibanaTagDir, "tag1.json")
		tag1Content := `{
  "attributes": {
	"name": "tagA"
  },
  "type": "tag"
}`
		err = os.WriteFile(tag1Path, []byte(tag1Content), 0o644)
		require.NoError(t, err)

		tag2Path := path.Join(kibanaTagDir, "tag2.json")
		tag2Content := `{
  "attributes": {
	"name": "tagA"
  },
  "type": "tag"
}`
		err = os.WriteFile(tag2Path, []byte(tag2Content), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(tmpDir)
		tags := []string{"tagB"}
		errs := validateKibanaPackageTagsDuplicates(fsys, tags)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "duplicate package tag name 'tagA'")
	})

	t.Run("with tag in JSON already defined in tags.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		kibanaTagDir := path.Join(tmpDir, "kibana", "tag")
		err := os.MkdirAll(kibanaTagDir, 0o755)
		require.NoError(t, err)

		tag1Path := path.Join(kibanaTagDir, "tag1.json")
		tag1Content := `{
  "attributes": {
	"name": "tagB"
  },
  "type": "tag"
}`
		err = os.WriteFile(tag1Path, []byte(tag1Content), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(tmpDir)
		tags := []string{"tagB"}
		errs := validateKibanaPackageTagsDuplicates(fsys, tags)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "tag name 'tagB' is already defined in tags.yml (SVR00007)")
	})

	t.Run("with unique tags in JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()
		kibanaTagDir := path.Join(tmpDir, "kibana", "tag")
		err := os.MkdirAll(kibanaTagDir, 0o755)
		require.NoError(t, err)

		tag1Path := path.Join(kibanaTagDir, "tag1.json")
		tag1Content := `{
  "attributes": {
	"name": "tagA"
  },
  "type": "tag"
}`
		err = os.WriteFile(tag1Path, []byte(tag1Content), 0o644)
		require.NoError(t, err)

		tag2Path := path.Join(kibanaTagDir, "tag2.json")
		tag2Content := `{
  "attributes": {
	"name": "tagB"
  },
  "type": "tag"
}`
		err = os.WriteFile(tag2Path, []byte(tag2Content), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(tmpDir)
		tags := []string{"tagC"}
		errs := validateKibanaPackageTagsDuplicates(fsys, tags)
		require.Empty(t, errs)
	})
}
