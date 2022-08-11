// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jsonschema

import (
	"io/fs"
	"io/ioutil"
	"path"

	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// LoadFolderSpec loads a spec from a directory in the given filesystem.
func LoadFolderSpec(fs fs.FS, specPath string) (*ItemSpec, error) {
	var spec folderItemSpec
	return &ItemSpec{&spec}, spec.loadFolderSpec(fs, path.Join(specPath, "spec.yml"))
}

func (s *folderItemSpec) loadFolderSpec(fs fs.FS, specPath string) error {
	specFile, err := fs.Open(specPath)
	if err != nil {
		return errors.Wrap(err, "could not open folder specification file")
	}
	defer specFile.Close()

	data, err := ioutil.ReadAll(specFile)
	if err != nil {
		return errors.Wrap(err, "could not read folder specification file")
	}

	var wrapper struct {
		Spec *commonSpec `yaml:"spec"`
	}
	wrapper.Spec = &s.commonSpec
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return errors.Wrap(err, "could not parse folder specification file")
	}

	err = s.loadContents(fs, specPath)
	if err != nil {
		return err
	}

	err = setDefaultValues(&s.commonSpec)
	if err != nil {
		return errors.Wrap(err, "could not set default values")
	}

	propagateContentLimits(&s.commonSpec)

	return nil
}

func (s *folderItemSpec) loadContents(fs fs.FS, specPath string) error {
	var content *folderItemSpec
	contents := append([]*folderItemSpec{}, s.Contents...)

	for len(contents) > 0 {
		content, contents = contents[0], contents[1:]

		// TODO: Visibility not used at the moment.
		if v := content.Visibility; v != "" && v != visibilityTypePrivate && v != visibilityTypePublic {
			return errors.Errorf("item [%s] visibility is expected to be private or public, not [%s]", path.Join(specPath, content.Name), content.Visibility)
		}

		// All folders inside a development folder are too.
		if s.DevelopmentFolder {
			content.DevelopmentFolder = true
		}

		if content.Ref != "" {
			// Resolve references.
			switch content.ItemType {
			case spectypes.ItemTypeFile:
				if err := content.loadSchema(fs, path.Dir(specPath)); err != nil {
					return errors.Wrapf(err, "could not load schema for %q", path.Dir(specPath))
				}
			case spectypes.ItemTypeFolder:
				p := path.Join(path.Dir(specPath), content.Ref)
				err := content.loadFolderSpec(fs, p)
				if err != nil {
					return errors.Wrapf(err, "could not load spec for %q", p)
				}
			}

		} else if len(content.Contents) > 0 {
			// Walk over folders defined inline.
			contents = append(contents, content.Contents...)
		}
	}

	return nil
}
