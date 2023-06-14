// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specschema

import (
	"fmt"
	"io/fs"
	"reflect"

	"github.com/creasty/defaults"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/spectypes"
)

const (
	visibilityTypePublic  = "public"
	visibilityTypePrivate = "private"
)

// ItemSpec implements the `spectype.ItemSpec` interface for a folder item spec.
type ItemSpec struct {
	itemSpec *folderItemSpec
}

// MaxTotalContents is the maximum number of files and directories
// inside a directory and its children directories.
func (s *ItemSpec) MaxTotalContents() int {
	return s.itemSpec.SpecLimits.TotalContentsLimit
}

// MaxTotalSize is the maximum size of a file, or all the files and
// directories inside a directory.
func (s *ItemSpec) MaxTotalSize() spectypes.FileSize {
	return s.itemSpec.SpecLimits.TotalSizeLimit
}

// MaxFileSize is the maximum size of an individual file.
func (s *ItemSpec) MaxFileSize() spectypes.FileSize {
	return s.itemSpec.SpecLimits.SizeLimit
}

// MaxConfigurationSize is the maximum size of a configuration file.
func (s *ItemSpec) MaxConfigurationSize() spectypes.FileSize {
	return s.itemSpec.SpecLimits.ConfigurationSizeLimit
}

// MaxRelativePathSize is the maximum size of a file indicated with a relative path.
func (s *ItemSpec) MaxRelativePathSize() spectypes.FileSize {
	return s.itemSpec.SpecLimits.RelativePathSizeLimit
}

// MaxFieldsPerDataStream is the maxumum number of fields that each data stream can define.
func (s *ItemSpec) MaxFieldsPerDataStream() int {
	return s.itemSpec.SpecLimits.FieldsPerDataStreamLimit
}

// AdditionalContents returns true if the item can contain contents not defined in the spec.
func (s *ItemSpec) AdditionalContents() bool {
	return s.itemSpec.AdditionalContents
}

// ContentMediaType returns the expected content type of a file.
func (s *ItemSpec) ContentMediaType() *spectypes.ContentType {
	return s.itemSpec.ContentMediaType
}

// Contents returns the definitions of the children elements of this item.
func (s *ItemSpec) Contents() []spectypes.ItemSpec {
	result := make([]spectypes.ItemSpec, len(s.itemSpec.Contents))
	for i := range s.itemSpec.Contents {
		result[i] = &ItemSpec{s.itemSpec.Contents[i]}
	}
	return result
}

// DevelopmentFolder returns true if the item is inside a development folder.
func (s *ItemSpec) DevelopmentFolder() bool {
	return s.itemSpec.DevelopmentFolder
}

// ForbiddenPatterns returns the list of forbidden patterns for the name of this item.
func (s *ItemSpec) ForbiddenPatterns() []string {
	return s.itemSpec.ForbiddenPatterns
}

// IsDir returns true if the item is a directory.
func (s *ItemSpec) IsDir() bool {
	return s.itemSpec.ItemType == spectypes.ItemTypeFolder
}

// Name returns the name of the item inside its parent.
func (s *ItemSpec) Name() string {
	return s.itemSpec.Name
}

// Pattern returns the allowed pattern for the name of this item.
func (s *ItemSpec) Pattern() string {
	return s.itemSpec.Pattern
}

// Release returns 'beta' if the item definition is in beta stage.
func (s *ItemSpec) Release() string {
	return s.itemSpec.Release
}

// Required returns true if this item must be defined.
func (s *ItemSpec) Required() bool {
	return s.itemSpec.Required
}

// Type returns the type of file ('file' or 'folder').
func (s *ItemSpec) Type() string {
	return s.itemSpec.ItemType
}

// ValidateSchema validates if the indicated file complies with the schema of the item.
func (s *ItemSpec) ValidateSchema(fsys fs.FS, itemPath string) ve.ValidationErrors {
	return s.itemSpec.ValidateSchema(fsys, itemPath)
}

type folderItemSpec struct {
	Description       string                 `json:"description" yaml:"description"`
	ItemType          string                 `json:"type" yaml:"type"`
	ContentMediaType  *spectypes.ContentType `json:"contentMediaType" yaml:"contentMediaType"`
	ForbiddenPatterns []string               `json:"forbiddenPatterns" yaml:"forbiddenPatterns"`
	Name              string                 `json:"name" yaml:"name"`
	Pattern           string                 `json:"pattern" yaml:"pattern"`
	Required          bool                   `json:"required" yaml:"required"`
	Ref               string                 `json:"$ref" yaml:"$ref"`
	Visibility        string                 `json:"visibility" yaml:"visibility" default:"public"`

	AdditionalContents bool              `json:"additionalContents" yaml:"additionalContents"`
	Contents           []*folderItemSpec `json:"contents" yaml:"contents"`
	DevelopmentFolder  bool              `json:"developmentFolder" yaml:"developmentFolder"`

	// As it is required to be inline both in yaml and json, this struct must be public embedded field
	SpecLimits `yaml:",inline"`

	// Release type of the spec: beta, ga.
	// Packages using beta features won't be able to go GA.
	// Default release: ga
	Release string `json:"release" yaml:"release"`

	schema spectypes.FileSchema
}

func (s *folderItemSpec) setDefaultValues() error {
	err := defaults.Set(s)
	if err != nil {
		return fmt.Errorf("could not set default values: %w", err)
	}

	for _, content := range s.Contents {
		err = content.setDefaultValues()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *folderItemSpec) propagateContentLimits() {
	for _, content := range s.Contents {
		content.SpecLimits.update(s.SpecLimits)
		content.propagateContentLimits()
	}
}

// SpecLimits represents limits related to an item
type SpecLimits struct {
	// Limit to the total number of elements in a directory.
	TotalContentsLimit int `json:"totalContentsLimit" yaml:"totalContentsLimit"`

	// Limit to the total size of files in a directory.
	TotalSizeLimit spectypes.FileSize `json:"totalSizeLimit" yaml:"totalSizeLimit"`

	// Limit to individual files.
	SizeLimit spectypes.FileSize `json:"sizeLimit" yaml:"sizeLimit"`

	// Limit to individual configuration files (yaml files).
	ConfigurationSizeLimit spectypes.FileSize `json:"configurationSizeLimit" yaml:"configurationSizeLimit"`

	// Limit to files referenced as relative paths (images).
	RelativePathSizeLimit spectypes.FileSize `json:"relativePathSizeLimit" yaml:"relativePathSizeLimit"`

	// Maximum number of fields per data stream, can only be set at the root level spec.
	FieldsPerDataStreamLimit int `json:"fieldsPerDataStreamLimit" yaml:"fieldsPerDataStreamLimit"`
}

func (l *SpecLimits) update(o SpecLimits) {
	target := reflect.ValueOf(l).Elem()
	source := reflect.ValueOf(&o).Elem()
	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)
		if field.IsZero() {
			field.Set(source.Field(i))
		}
	}
}

func (s *folderItemSpec) ValidateSchema(fsys fs.FS, itemPath string) ve.ValidationErrors {
	if s.schema == nil {
		return nil
	}
	return s.schema.Validate(fsys, itemPath)
}
