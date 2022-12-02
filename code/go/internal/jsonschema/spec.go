package jsonschema

import (
	"log"
	"path"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"

	"github.com/elastic/package-spec/v2/code/go/internal/spectypes"
)

type RenderedJSONSchema struct {
	Name       string
	JSONSchema []byte
}

func AllJSONSchemas(rootSpec spectypes.ItemSpec) ([]RenderedJSONSchema, error) {
	contents, err := marshalSpec(rootSpec)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func JSONSchema(rootSpec spectypes.ItemSpec, location string) (*RenderedJSONSchema, error) {
	contents, err := marshalSpec(rootSpec)
	if err != nil {
		return nil, err
	}

	var rendered RenderedJSONSchema
	for _, content := range contents {
		matched, err := matchContentWithFile(location, content.Name)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}

		if content.JSONSchema == nil {
			content.JSONSchema = []byte("")
		}

		if len(rendered.JSONSchema) > 0 && len(content.JSONSchema) == 0 {
			// not overwrite rendered with contents that are empty strings (e.g. docs/README.md)
			continue
		}
		rendered = content
		log.Printf("Matched item spec %s for path %s", content.Name, location)
	}
	if rendered.JSONSchema == nil {
		return nil, errors.Errorf("item path not found: %s", location)
	}
	return &rendered, nil
}

func marshalSpec(spec spectypes.ItemSpec) ([]RenderedJSONSchema, error) {
	var allContents []RenderedJSONSchema
	if len(spec.Contents()) == 0 {
		key := spec.Name()
		if key == "" {
			key = spec.Pattern()
		}
		contents, err := spec.Marshal()
		if err != nil {
			return nil, err
		}

		allContents = append(allContents, RenderedJSONSchema{key, contents})
		return allContents, nil
	}
	pending := spec.Contents()
	for _, item := range pending {
		itemsJSON, err := marshalSpec(item)
		if err != nil {
			return nil, err
		}
		if item.IsDir() {
			for c, elem := range itemsJSON {
				itemsJSON[c].Name = path.Join(item.Name(), elem.Name)
			}
		}
		allContents = append(allContents, itemsJSON...)
	}
	return allContents, nil
}

func matchContentWithFile(location, content string) (bool, error) {
	baseLocation := filepath.Base(location)
	baseContent := filepath.Base(content)

	dirLocation := filepath.Dir(location)
	dirContent := filepath.Dir(content)

	if dirLocation != dirContent {
		return false, nil
	}

	r, err := regexp.Compile(baseContent)
	if err != nil {
		log.Printf(" -- Error %+s", err)
		return false, errors.Wrap(err, "failed to compile regex")
	}
	if !r.MatchString(baseLocation) {
		return false, nil
	}

	return true, nil
}
