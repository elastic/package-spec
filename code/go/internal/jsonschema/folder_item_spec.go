// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jsonschema

import (
	"fmt"
	"io/fs"
	"path"
	"sync"

	"github.com/elastic/gojsonschema"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/elastic/package-spec/code/go/internal/yamlschema"
)

const (
	visibilityTypePublic  = "public"
	visibilityTypePrivate = "private"
)

type ItemSpec struct {
	itemSpec *folderItemSpec
}

func (s *ItemSpec) MaxTotalContents() int {
	return s.itemSpec.Limits.TotalContentsLimit
}
func (s *ItemSpec) MaxTotalSize() spectypes.FileSize {
	return s.itemSpec.Limits.TotalSizeLimit
}
func (s *ItemSpec) MaxFileSize() spectypes.FileSize {
	return s.itemSpec.Limits.SizeLimit
}
func (s *ItemSpec) MaxConfigurationSize() spectypes.FileSize {
	return s.itemSpec.Limits.ConfigurationSizeLimit
}
func (s *ItemSpec) MaxRelativePathSize() spectypes.FileSize {
	return s.itemSpec.Limits.RelativePathSizeLimit
}
func (s *ItemSpec) MaxFieldsPerDataStream() int {
	return s.itemSpec.Limits.FieldsPerDataStreamLimit
}
func (s *ItemSpec) AdditionalContents() bool {
	return s.itemSpec.AdditionalContents
}
func (s *ItemSpec) ContentMediaType() *spectypes.ContentType {
	return s.itemSpec.ContentMediaType
}
func (s *ItemSpec) Contents() []spectypes.ItemSpec {
	result := make([]spectypes.ItemSpec, len(s.itemSpec.Contents))
	for i := range s.itemSpec.Contents {
		result[i] = &ItemSpec{s.itemSpec.Contents[i]}
	}
	return result
}
func (s *ItemSpec) DevelopmentFolder() bool {
	return s.itemSpec.DevelopmentFolder
}
func (s *ItemSpec) ForbiddenPatterns() []string {
	return s.itemSpec.ForbiddenPatterns
}
func (s *ItemSpec) IsDir() bool {
	return s.itemSpec.ItemType == spectypes.ItemTypeFolder
}
func (s *ItemSpec) Name() string {
	return s.itemSpec.Name
}
func (s *ItemSpec) Pattern() string {
	return s.itemSpec.Pattern
}
func (s *ItemSpec) Release() string {
	return s.itemSpec.Release
}
func (s *ItemSpec) Required() bool {
	return s.itemSpec.Required
}
func (s *ItemSpec) Type() string {
	return s.itemSpec.ItemType
}
func (s *ItemSpec) ValidateSchema(fsys fs.FS, itemPath string) ve.ValidationErrors {
	return s.itemSpec.ValidateSchema(fsys, itemPath)
}

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

	schema     *gojsonschema.Schema
	commonSpec `yaml:",inline"`
}

func (s *folderItemSpec) loadSchema(schemaFsys fs.FS, folderSpecPath string) error {
	if s.Ref == "" {
		return nil // item's schema is not defined
	}

	schemaPath := path.Join(folderSpecPath, s.Ref)
	schemaLoader := yamlschema.NewReferenceLoaderFileSystem("file:///"+schemaPath, schemaFsys)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to load schema for %q: %v", schemaPath, err)
	}
	s.schema = schema
	return nil
}

var formatCheckersMutex sync.Mutex

func (s *folderItemSpec) ValidateSchema(fsys fs.FS, itemPath string) ve.ValidationErrors {
	if s.schema == nil {
		return nil
	}

	data, err := loadItemSchema(fsys, itemPath, s.ContentMediaType)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	formatCheckersMutex.Lock()
	defer func() {
		unloadRelativePathFormatChecker()
		unloadDataStreamNameFormatChecker()
		formatCheckersMutex.Unlock()
	}()

	loadRelativePathFormatChecker(fsys, path.Dir(itemPath), s.Limits.RelativePathSizeLimit)
	loadDataStreamNameFormatChecker(fsys, path.Dir(itemPath))
	result, err := s.schema.Validate(gojsonschema.NewBytesLoader(data))
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
