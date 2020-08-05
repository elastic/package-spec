package validator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const ItemTypeFile = "file"
const ItemTypeFolder = "folder"

type folderSpec struct {
	specPath  string
	itemSpecs []folderItemSpec
}

type folderItemSpec struct {
	Description      string `yaml:"description"`
	ItemType         string `yaml:"type"`
	ContentMediaType string `yaml:"contentMediaType"`
	Name             string `yaml:"name"`
	Pattern          string `yaml:"pattern"`
	Required         bool   `yaml:"required"`
	Ref              string `yaml:"$ref"`
}

func newFolderSpec(specPath string) (*folderSpec, error) {
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "no folder specification file found")
	}

	data, err := ioutil.ReadFile(specPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read folder specification file")
	}

	var wrapper struct {
		Spec []folderItemSpec `yaml:",flow"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, errors.Wrap(err, "could not parse folder specification file")
	}

	fs := folderSpec{
		specPath,
		wrapper.Spec,
	}

	return &fs, nil
}

func (fs *folderSpec) Validate(folderPath string) ValidationErrors {
	var errs ValidationErrors
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "could not read folder [%s]", folderPath))
		return errs
	}

	for _, file := range files {
		fileName := file.Name()
		itemSpec, err := fs.findItemSpec(fileName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if itemSpec == nil {
			errs = append(errs, fmt.Errorf("filename [%s] does not match spec for folder [%s]", fileName, folderPath))
			continue
		}

		if file.IsDir() {
			if !itemSpec.isSameType(file) {
				errs = append(errs, fmt.Errorf("[%s] is a folder but is expected to be a file", fileName))
				continue
			}

			if itemSpec.Ref == "" {
				continue
			}

			subFolderSpecPath := path.Join(filepath.Dir(fs.specPath), itemSpec.Ref)
			subFolderSpec, err := newFolderSpec(subFolderSpecPath)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			subFolderPath := path.Join(folderPath, fileName)
			errs = subFolderSpec.Validate(subFolderPath)
			if len(errs) > 0 {
				errs = append(errs, err)
			}
		} else {
			if !itemSpec.isSameType(file) {
				errs = append(errs, fmt.Errorf("[%s] is a file but is expected to be a folder", fileName))
				continue
			}
			// TODO: more validation for file item
		}
	}

	// Validate that required items in spec are all accounted for
	for _, itemSpec := range fs.itemSpecs {
		if !itemSpec.Required {
			continue
		}

		fileFound, err := itemSpec.matchingFileExists(files)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !fileFound {
			var err error
			if itemSpec.Name != "" {
				err = fmt.Errorf("expecting to find %s matching name [%s] in folder [%s]", itemSpec.ItemType, itemSpec.Name, folderPath)
			} else if itemSpec.Pattern != "" {
				err = fmt.Errorf("expecting to find %s matching pattern [%s] in folder [%s]", itemSpec.ItemType, itemSpec.Name, folderPath)
			}
			errs = append(errs, err)
		}
	}
	return errs
}

func (fs *folderSpec) findItemSpec(folderItemName string) (*folderItemSpec, error) {
	for _, itemSpec := range fs.itemSpecs {
		if itemSpec.Name != "" && itemSpec.Name == folderItemName {
			return &itemSpec, nil
		}
		if itemSpec.Pattern != "" {
			isMatch, err := regexp.MatchString(itemSpec.Pattern, folderItemName)
			if err != nil {
				return nil, errors.Wrap(err, "invalid folder item spec pattern")
			}
			if isMatch {
				return &itemSpec, nil
			}
		}
	}

	// No item spec found
	return nil, nil
}

func (is *folderItemSpec) matchingFileExists(files []os.FileInfo) (bool, error) {
	if is.Name != "" {
		for _, file := range files {
			if file.Name() == is.Name {
				return is.isSameType(file), nil
			}
		}
	} else if is.Pattern != "" {
		for _, file := range files {
			isMatch, err := regexp.MatchString(is.Pattern, file.Name())
			if err != nil {
				return false, errors.Wrap(err, "invalid folder item spec pattern")
			}
			if isMatch {
				return is.isSameType(file), nil
			}
		}
	}

	return false, nil
}

func (is *folderItemSpec) isSameType(file os.FileInfo) bool {
	switch is.ItemType {
	case ItemTypeFile:
		return !file.IsDir()
	case ItemTypeFolder:
		return file.IsDir()
	}

	return false
}
