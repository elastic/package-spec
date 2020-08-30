package validator

import (
	"fmt"
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

func (s *folderItemSpec) validate(fs http.FileSystem, folderSpecPath string, itemPath string) error {
	if s.Ref == "" {
		return nil // no item's schema defined
	}

	itemSchemaPath := filepath.Join(filepath.Dir(folderSpecPath), s.Ref)
	itemSchemaFile, err := fs.Open(itemSchemaPath)
	if err != nil {
		return errors.Wrap(err, "opening schema file failed")
	}
	defer itemSchemaFile.Close()

	itemSchemaData, err := ioutil.ReadAll(itemSchemaFile)
	if err != nil {
		return errors.Wrap(err, "reading schema file failed")
	}

	var schema yaml.Node
	err = yaml.Unmarshal(itemSchemaData, &schema)
	if err != nil {
		return errors.Wrapf(err, "schema unmarshalling failed (path: %s)", itemSchemaPath)
	}

	

	fmt.Println("folderItemSpec.validate()", itemSchemaPath, itemPath, s.ContentMediaType)
	return nil
}
