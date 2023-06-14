// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specschema

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path"

	"github.com/Masterminds/semver/v3"
	jsonpatch "github.com/evanphx/json-patch/v5"
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

type folderSchemaSpec struct {
	Spec     *folderItemSpec       `json:"spec" yaml:"spec"`
	Versions []folderSchemaVersion `json:"versions" yaml:"versions"`
}

type folderSchemaVersion struct {
	// Before is the first version that didn't include this change.
	Before string `json:"before" yaml:"before"`

	// Patch is a list of JSON patch operations as defined by RFC6902.
	Patch []interface{} `json:"patch" yaml:"patch"`
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
	specBytes, _ := json.Marshal(spec)
	log.Printf(">>> Load Method --> spec:\n%s", specBytes)
	return &ItemSpec{&spec}, nil
}

func (l *FolderSpecLoader) loadFolderSpec(s *folderItemSpec, specPath string) error {
	log.Printf("Reading file: %s", specPath)
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

	// TODO add comment
	*s = *newSpec

	specBytes, _ := json.Marshal(s)
	log.Printf("Final spec:\n%s", specBytes)

	return nil
}

func (f *folderSchemaSpec) resolve(target semver.Version) (*folderItemSpec, error) {
	patchJSON, err := f.patchForVersion(target)
	if err != nil {
		return nil, err
	}
	if len(patchJSON) == 0 {
		// Nothing to do.
		spec, err := json.Marshal(f.Spec)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec for patching: %w", err)
		}

		log.Printf("no versions -> applying spec:\n%s", string(spec))

		return f.Spec, nil
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode patch: %w", err)
	}

	spec, err := json.Marshal(f.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec for patching: %w", err)
	}

	log.Printf("spec before applying:\n%s", string(spec))
	log.Printf("Applied patchJson:\n%s", string(patchJSON))
	log.Printf("---")
	spec, err = patch.Apply(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}
	log.Printf("spec after applying:\n%s", string(spec))

	var resolved folderItemSpec
	err = json.Unmarshal(spec, &resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal resolved spec: %w", err)
	}
	return &resolved, nil
}

func (f *folderSchemaSpec) patchForVersion(target semver.Version) ([]byte, error) {
	var patch []any
	for _, version := range f.Versions {
		if sv, err := semver.NewVersion(version.Before); err != nil {
			return nil, err
		} else if !target.LessThan(sv) {
			continue
		}

		patch = append(patch, version.Patch...)
	}
	if len(patch) == 0 {
		return nil, nil
	}
	return json.Marshal(patch)
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
				var err error
				err = l.loadFolderSpec(content, p)
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
