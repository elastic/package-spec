// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
	"gopkg.in/yaml.v3"
)

const (
	tagType = "tag"
)

type packageSpecTag struct {
	Attributes struct {
		Name string `yaml:"name"`
	} `yaml:"attributes"`
	Type string `yaml:"type"`
}

type sharedTag struct {
	Name string `yaml:"text"`
}

func ValidateKibanaTagDuplicates(fsys fspath.FS) specerrors.ValidationErrors {
	tagMap, errs := getKibanaTagsYMLMap(fsys)
	if len(errs) > 0 {
		return errs
	}

	errs = validateKibanaJSONTags(fsys, tagMap)
	return errs
}

func getKibanaTagsYMLMap(fsys fspath.FS) (map[string]struct{}, specerrors.ValidationErrors) {
	tagMap := make(map[string]struct{})
	tagsPath := path.Join("kibana", "tags.yml")
	// Collect all tags defined in the kibana/tags.yml file.
	b, err := fs.ReadFile(fsys, tagsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return tagMap, nil
		}
		return nil, specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}
	var sharedKibanaTags []sharedTag
	err = yaml.Unmarshal(b, &sharedKibanaTags)
	if err != nil {
		return nil, specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	errs := make(specerrors.ValidationErrors, 0)
	// Check for duplicate tag names in the kibana/tags.yml file.
	for _, tag := range sharedKibanaTags {
		if _, exists := tagMap[tag.Name]; exists {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("duplicate tag name '%s' found in %s", tag.Name, tagsPath), specerrors.UnassignedCode))
			continue
		}
		tagMap[tag.Name] = struct{}{}
	}
	return tagMap, errs
}

func validateKibanaJSONTags(fsys fspath.FS, tagMap map[string]struct{}) specerrors.ValidationErrors {
	// Collect all tags used in the package.
	tagDir := path.Join("kibana", "tag")
	pkgTagNames := make(map[string]struct{})
	errs := make(specerrors.ValidationErrors, 0)
	err := fs.WalkDir(fsys, tagDir, func(filePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if !errors.Is(walkErr, fs.ErrNotExist) {
				return walkErr
			}
			return nil
		}
		if d.IsDir() || path.Ext(filePath) != ".json" {
			return nil
		}
		b, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return err
		}
		var pkgTag packageSpecTag
		err = json.Unmarshal(b, &pkgTag)
		if err != nil {
			return err
		}
		if pkgTag.Type != tagType {
			return nil
		}
		// check if the tag is already used in the package.
		if _, exists := pkgTagNames[pkgTag.Attributes.Name]; exists {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("duplicate tag name '%s' found in package tag %s", pkgTag.Attributes.Name, filePath), specerrors.UnassignedCode))
			return nil
		}
		// check if the tag is defined in the kibana/tags.yml file.
		if _, exists := tagMap[pkgTag.Attributes.Name]; exists {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("tag name '%s' used in package tag %s is already defined in tags.yml", pkgTag.Attributes.Name, filePath), specerrors.UnassignedCode))
			return nil
		}
		pkgTagNames[pkgTag.Attributes.Name] = struct{}{}
		return nil
	})
	if err != nil {
		if ve, ok := err.(specerrors.ValidationErrors); ok {
			return ve
		}
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}
	return errs
}
