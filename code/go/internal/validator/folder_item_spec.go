package validator

import (
	"encoding/json"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
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

type itemSchema struct {
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

	// loading item schema

	itemSchemaPath := filepath.Join(filepath.Dir(folderSpecPath), s.Ref)
	itemSchemaFile, err := fs.Open(itemSchemaPath)
	if err != nil {
		return ValidationErrors{errors.Wrap(err, "opening schema file failed")}
	}
	defer itemSchemaFile.Close()

	itemSchemaData, err := ioutil.ReadAll(itemSchemaFile)
	if err != nil {
		return ValidationErrors{errors.Wrap(err, "reading schema file failed")}
	}

	if len(itemSchemaData) == 0 {
		return ValidationErrors{errors.New("schema file is empty")}
	}

	var schema itemSchema
	err = yaml.Unmarshal(itemSchemaData, &schema)
	if err != nil {
		return ValidationErrors{errors.Wrapf(err, "schema unmarshalling failed (path: %s)", itemSchemaPath)}
	}

	schemaData, err := json.Marshal(&schema.Spec)
	if err != nil {
		return ValidationErrors{errors.Wrapf(err, "marshalling schema to JSON format failed")}
	}

	// loading item content
	itemData, err := ioutil.ReadFile(itemPath)
	if err != nil {
		return ValidationErrors{errors.Wrap(err, "reading item file failed")}
	}

	if len(itemData) == 0 {
		return ValidationErrors{errors.New("file is empty")}
	}

	switch s.ContentMediaType {
	case "application/x-yaml":
		var c interface{}
		err = yaml.Unmarshal(itemData, &c)
		if err != nil {
			return ValidationErrors{errors.Wrapf(err, "unmarshalling YAML file failed (path: %s)", itemSchemaPath)}
		}

		itemData, err = json.Marshal(&c)
		if err != nil {
			return ValidationErrors{errors.Wrapf(err, "converting YAML file to JSON failed (path: %s)", itemSchemaPath)}
		}
	case "application/json": // no need to convert the item content
	default:
		return ValidationErrors{fmt.Errorf("unsupported file media type (%s)", s.ContentMediaType)}
	}


	// validation
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
