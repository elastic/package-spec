package validator

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

type folderItemSpec struct {
	Description      string `yaml:"description"`
	ItemType         string `yaml:"type"`
	ContentMediaType string `yaml:"contentMediaType"`
	Name             string `yaml:"name"`
	Pattern          string `yaml:"pattern"`
	Required         bool   `yaml:"required"`
	Ref              string `yaml:"$ref"`
	Visibility       string `yaml:"visibility" default:"public"`
	commonSpec       `yaml:",inline"`
}

type itemSchemaSpec struct {
	Spec map[string]interface{} `json:"spec" yaml:"spec"`
}

func (s *folderItemSpec) matchingFileExists(files []os.FileInfo) (bool, error) {
	if s.Name != "" {
		for _, file := range files {
			if file.Name() == s.Name {
				return s.isSameType(file), nil
			}
		}
	} else if s.Pattern != "" {
		for _, file := range files {
			isMatch, err := regexp.MatchString(s.Pattern, file.Name())
			if err != nil {
				return false, errors.Wrap(err, "invalid folder item spec pattern")
			}
			if isMatch {
				return s.isSameType(file), nil
			}
		}
	}

	return false, nil
}

func (s *folderItemSpec) isSameType(file os.FileInfo) bool {
	switch s.ItemType {
	case itemTypeFile:
		return !file.IsDir()
	case itemTypeFolder:
		return file.IsDir()
	}

	return false
}

func (s *folderItemSpec) validate(fs http.FileSystem, folderSpecPath string, itemPath string) ValidationErrors {
	if s.Ref == "" {
		return nil // no item's schema defined
	}

	schemaPath := filepath.Join(filepath.Dir(folderSpecPath), s.Ref)
	schemaData, err := loadItemSchema(fs, schemaPath)
	if err != nil {
		return ValidationErrors{errors.Wrapf(err, "loading item schema failed (path :s)", schemaPath)}
	}

	// loading item content
	itemData, err := loadItemContent(itemPath, s.ContentMediaType)
	if err != nil {
		return ValidationErrors{errors.Wrapf(err, "loading item content failed (path :s)", itemPath)}
	}

	// validation with schema
	errs := validateData(schemaData, itemData)
	if errs != nil {
		return errs
	}
	return nil
}

func loadItemSchema(fs http.FileSystem, itemSchemaPath string) ([]byte, error) {
	itemSchemaFile, err := fs.Open(itemSchemaPath)
	if err != nil {
		return nil, errors.Wrap(err, "opening schema file failed")
	}
	defer itemSchemaFile.Close()

	itemSchemaData, err := ioutil.ReadAll(itemSchemaFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading schema file failed")
	}

	if len(itemSchemaData) == 0 {
		return nil, errors.New("schema file is empty")
	}

	var schema itemSchemaSpec
	err = yaml.Unmarshal(itemSchemaData, &schema)
	if err != nil {
		return nil, errors.Wrapf(err, "schema unmarshalling failed (path: %s)", itemSchemaPath)
	}

	schemaData, err := json.Marshal(&schema.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "marshalling schema to JSON format failed")
	}
	return schemaData, nil
}

func loadItemContent(itemPath, mediaType string) ([]byte, error) {
	itemData, err := ioutil.ReadFile(itemPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading item file failed")
	}

	if len(itemData) == 0 {
		return nil, errors.New("file is empty")
	}

	switch mediaType {
	case "application/x-yaml":
		var c interface{}
		err = yaml.Unmarshal(itemData, &c)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshalling YAML file failed (path: %s)", itemPath)
		}

		itemData, err = json.Marshal(&c)
		if err != nil {
			return nil, errors.Wrapf(err, "converting YAML file to JSON failed (path: %s)", itemPath)
		}
	case "application/json": // no need to convert the item content
	default:
		return nil, fmt.Errorf("unsupported media type (%s)", mediaType)
	}
	return itemData, nil
}

func validateData(schemaData, itemData []byte) ValidationErrors {
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	documentLoader := gojsonschema.NewBytesLoader(itemData)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return ValidationErrors{err}
	}

	if result.Valid() {
		return nil // item content is valid according to the loaded schema
	}

	var errs ValidationErrors
	for _, re := range result.Errors() {
		errs = append(errs, fmt.Errorf("field %s: %s", re.Field(), re.Description()))
	}
	return errs
}