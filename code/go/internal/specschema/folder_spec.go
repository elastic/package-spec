// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specschema

import (
	"io"
	"io/fs"
	"path"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v2/code/go/internal/spectypes"
)

// FolderSpecLoader loads specs from directories.
type FolderSpecLoader struct {
	fs             fs.FS
	fileSpecLoader spectypes.FileSchemaLoader
	specVersion    semver.Version
}

// NewFolderSpecLoader creates a new `FolderSpecLoader` that loads schemas from the given directories.
// File schemas referenced with `$ref` are loaded using the given `FileSchemaLoader`.
func NewFolderSpecLoader(fs fs.FS, fileLoader spectypes.FileSchemaLoader, version semver.Version) *FolderSpecLoader {
	return &FolderSpecLoader{
		fs:             fs,
		fileSpecLoader: fileLoader,
		specVersion:    version,
	}
}

// Load loads a spec from the given path.
func (l *FolderSpecLoader) Load(specPath string) (*ItemSpec, error) {
	var spec folderItemSpec
	err := l.loadFolderSpec(&spec, path.Join(specPath, "spec.yml"))
	if err != nil {
		return nil, err
	}
	return &ItemSpec{&spec}, nil
}

func (l *FolderSpecLoader) loadFolderSpec(s *folderItemSpec, specPath string) error {
	specFile, err := l.fs.Open(specPath)
	if err != nil {
		return errors.Wrap(err, "could not open folder specification file")
	}
	defer specFile.Close()

	data, err := io.ReadAll(specFile)
	if err != nil {
		return errors.Wrap(err, "could not read folder specification file")
	}

	var folderSpec folderSchemaSpec
	folderSpec.Spec = s
	if err := yaml.Unmarshal(data, &folderSpec); err != nil {
		return errors.Wrap(err, "could not parse folder specification file")
	}

	newSpec, err := folderSpec.resolve(l.specVersion)
	if err != nil {
		return err
	}

	err = l.loadContents(newSpec, l.fs, specPath)
	if err != nil {
		return err
	}

	err = newSpec.setDefaultValues()
	if err != nil {
		return errors.Wrap(err, "could not set default values")
	}

	newSpec.propagateContentLimits()

	// it is required to assign the real values to be able to
	// use all the calculated contents in following iterations
	*s = *newSpec

	return nil
}

func (l *FolderSpecLoader) loadContents(s *folderItemSpec, fs fs.FS, specPath string) error {
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
				if l.fileSpecLoader == nil {
					break
				}
				specPath := path.Join(path.Dir(specPath), content.Ref)
				options := spectypes.FileSchemaLoadOptions{
					SpecVersion: l.specVersion,
					Limits:      &ItemSpec{content},
					ContentType: content.ContentMediaType,
				}
				schema, err := l.fileSpecLoader.Load(fs, specPath, options)
				if err != nil {
					return errors.Wrapf(err, "could not load schema for %q", path.Dir(specPath))
				}
				content.schema = schema
			case spectypes.ItemTypeFolder:
				p := path.Join(path.Dir(specPath), content.Ref)
				err := l.loadFolderSpec(content, p)
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
