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
	"slices"

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

type sharedTagYML struct {
	Name string `yaml:"text"`
}

// ValidateKibanaTagDuplicates checks for duplicate Kibana tag names
// between the kibana/tags.yml file and the tags defined in the package's kibana/tag/*.json files.
// It returns a list of validation errors if any duplicates are found.
func ValidateKibanaTagDuplicates(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	sharedTagNames, verr := getValidatedSharedKibanaTags(fsys)
	if len(verr) > 0 {
		errs = append(errs, verr...)
	}

	verr = validateKibanaPackageTagsDuplicates(fsys, sharedTagNames)
	if len(verr) > 0 {
		errs = append(errs, verr...)
	}
	return errs
}

// getValidatedSharedKibanaTags reads the kibana/tags.yml file and returns a slice of tag names defined in it.
// It also returns any validation errors encountered during the process if tags are duplicated within the file.
func getValidatedSharedKibanaTags(fsys fspath.FS) ([]string, specerrors.ValidationErrors) {
	tagsPath := path.Join("kibana", "tags.yml")
	// Collect all tags defined in the kibana/tags.yml file.
	b, err := fs.ReadFile(fsys, tagsPath)
	if err != nil {
		// if the file does not exist, return an empty slice without error
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, specerrors.ValidationErrors{specerrors.NewStructuredErrorf("error reading file %s: %v", tagsPath, err)}
	}
	var sharedKibanaTags []sharedTagYML
	err = yaml.Unmarshal(b, &sharedKibanaTags)
	if err != nil {
		return nil, specerrors.ValidationErrors{specerrors.NewStructuredErrorf("error unmarshaling file %s: %v", tagsPath, err)}
	}

	tags := make([]string, 0)
	errs := make(specerrors.ValidationErrors, 0)
	// Check for duplicate tag names in the kibana/tags.yml file.
	for _, tag := range sharedKibanaTags {
		if slices.Contains(tags, tag.Name) {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: duplicate tag name '%s' found", fsys.Path(tagsPath), tag.Name), specerrors.CodeKibanaTagDuplicates))
			continue
		}
		tags = append(tags, tag.Name)
	}
	return tags, errs
}

func validateKibanaPackageTagsDuplicates(fsys fspath.FS, sharedTagNames []string) specerrors.ValidationErrors {

	entries, err := fs.ReadDir(fsys, path.Join("kibana", "tag"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("error reading kibana/tag directory: %v", err)}
	}

	tags := make([]string, 0)
	errs := make(specerrors.ValidationErrors, 0)
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			// skip non-json files and directories
			continue
		}
		filePath := path.Join("kibana", "tag", entry.Name())
		b, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("error reading file %s: %v", fsys.Path(filePath), err))
			continue
		}
		var pkgTag packageSpecTag
		err = json.Unmarshal(b, &pkgTag)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("error unmarshaling file %s: %v", fsys.Path(filePath), err))
			continue
		}
		// skip non-tag types
		if pkgTag.Type != tagType {
			continue
		}

		// validate if the tag is already defined in other json file
		if slices.Contains(tags, pkgTag.Attributes.Name) {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: duplicate package tag name '%s' found", fsys.Path(filePath), pkgTag.Attributes.Name), specerrors.CodeKibanaTagDuplicates))
			continue
		}
		if slices.Contains(sharedTagNames, pkgTag.Attributes.Name) {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: tag name '%s' is already defined in tags.yml", fsys.Path(filePath), pkgTag.Attributes.Name), specerrors.CodeKibanaTagDuplicates))
			continue
		}
		tags = append(tags, pkgTag.Attributes.Name)
	}
	return errs
}
