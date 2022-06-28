// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sync"

	"github.com/pkg/errors"
	"github.com/elastic/gojsonschema"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/elastic/package-spec/code/go/internal/validator/semantic"
	"github.com/elastic/package-spec/code/go/internal/yamlschema"
)

type folderItemSpec struct {
	Description       string                 `yaml:"description"`
	ItemType          string                 `yaml:"type"`
	ContentMediaType  *spectypes.ContentType `yaml:"contentMediaType"`
	ForbiddenPatterns []string               `yaml:"forbiddenPatterns"`
	Name              string                 `yaml:"name"`
	Pattern           string                 `yaml:"pattern"`
	Required          bool                   `yaml:"required"`
	Ref               string                 `yaml:"$ref"`
	Visibility        string                 `yaml:"visibility" default:"public"`
	commonSpec        `yaml:",inline"`
}

var formatCheckersMutex sync.Mutex

func (s *folderItemSpec) matchingFileExists(files []fs.DirEntry) (bool, error) {
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

// sameTypeChecker is the interface that parameters of isSameType should implement,
// this is intended to accept both fs.DirEntry and fs.FileInfo.
type sameTypeChecker interface {
	IsDir() bool
}

func (s *folderItemSpec) isSameType(file sameTypeChecker) bool {
	switch s.ItemType {
	case itemTypeFile:
		return !file.IsDir()
	case itemTypeFolder:
		return file.IsDir()
	}

	return false
}

func (s *folderItemSpec) validate(schemaFsys fs.FS, fsys fs.FS, folderSpecPath string, itemPath string) ve.ValidationErrors {
	err := validateMaxSize(fsys, itemPath, s.Limits)
	if err != nil {
		return ve.ValidationErrors{err}
	}
	if s.ContentMediaType != nil {
		err := validateContentType(fsys, itemPath, *s.ContentMediaType)
		if err != nil {
			return ve.ValidationErrors{err}
		}
		err = validateContentTypeSize(fsys, itemPath, *s.ContentMediaType, s.Limits)
		if err != nil {
			return ve.ValidationErrors{err}
		}
	}

	errs := s.validateSchema(schemaFsys, fsys, folderSpecPath, itemPath)
	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (s *folderItemSpec) validateSchema(schemaFsys fs.FS, fsys fs.FS, folderSpecPath, itemPath string) ve.ValidationErrors {
	if s.Ref == "" {
		return nil // item's schema is not defined
	}

	schemaPath := path.Join(path.Dir(folderSpecPath), s.Ref)
	schemaLoader := yamlschema.NewReferenceLoaderFileSystem("file:///"+schemaPath, schemaFsys)

	// validation with schema
	itemData, err := loadItemSchema(fsys, itemPath, s.ContentMediaType)
	if err != nil {
		return ve.ValidationErrors{err}
	}
	documentLoader := gojsonschema.NewBytesLoader(itemData)

	formatCheckersMutex.Lock()
	defer func() {
		semantic.UnloadRelativePathFormatChecker()
		semantic.UnloadDataStreamNameFormatChecker()
		formatCheckersMutex.Unlock()
	}()

	semantic.LoadRelativePathFormatChecker(fsys, path.Dir(itemPath), s.Limits.RelativePathSizeLimit)
	semantic.LoadDataStreamNameFormatChecker(fsys, path.Dir(itemPath))
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	if !result.Valid() {
		var errs ve.ValidationErrors
		for _, re := range result.Errors() {
			errs = append(errs, fmt.Errorf("field %s: %s", re.Field(), adjustErrorDescription(re.Description())))
		}
		return errs
	}

	return nil // item content is valid according to the loaded schema
}
