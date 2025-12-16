package semantic

import (
	"os"
	"path"
	"testing"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/stretchr/testify/require"
)

func TestGetKibanaTagsYMLMap(t *testing.T) {
	t.Run("no tags.yml file", func(t *testing.T) {
		tmpDir := t.TempDir()
		fsys := fspath.DirFS(tmpDir)

		tagMap, errs := getKibanaTagsYMLMap(fsys)
		require.Empty(t, errs)
		require.Empty(t, tagMap)
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
		tagMap, errs := getKibanaTagsYMLMap(fsys)
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "duplicate tag name 'tag1' found in kibana/tags.yml")
		require.Len(t, tagMap, 2)
		require.Contains(t, tagMap, "tag1")
		require.Contains(t, tagMap, "tag2")
	})
}

func TestValidateKibanaJSONTags(t *testing.T) {
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
		tagMap := map[string]struct{}{
			"tagB": {},
		}
		errs := validateKibanaJSONTags(fsys, tagMap)
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "duplicate tag name 'tagA' found in package tag")
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
		tagMap := map[string]struct{}{
			"tagB": {},
		}
		errs := validateKibanaJSONTags(fsys, tagMap)
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "tag name 'tagB' used in package tag")
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
		tagMap := map[string]struct{}{
			"tagC": {},
		}
		errs := validateKibanaJSONTags(fsys, tagMap)
		require.Empty(t, errs)
	})
}
