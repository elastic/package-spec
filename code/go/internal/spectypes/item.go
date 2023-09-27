// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spectypes

import (
	"io/fs"

	"github.com/elastic/package-spec/v2/code/go/pkg/specerrors"
)

const (
	// ItemTypeFile is the type of an item that represents a file.
	ItemTypeFile = "file"

	// ItemTypeFolder is the type of an item that represents a folder.
	ItemTypeFolder = "folder"
)

// LimitsSpec contain the specifications related to limits.
type LimitsSpec interface {
	// MaxTotalContents is the maximum number of files and directories
	// inside a directory and its children directories.
	MaxTotalContents() int

	// MaxTotalSize is the maximum size of a file, or all the files and
	// directories inside a directory.
	MaxTotalSize() FileSize

	// MaxFileSize is the maximum size of an individual file.
	MaxFileSize() FileSize

	// MaxConfigurationSize is the maximum size of a configuration file.
	MaxConfigurationSize() FileSize

	// MaxRelativePathSize is the maximum size of a file indicated with a relative path.
	MaxRelativePathSize() FileSize

	// MaxFieldsPerDataStream is the maxumum number of fields that each data stream can define.
	MaxFieldsPerDataStream() int
}

// ItemSpec is the interface that specifications of items must implement.
type ItemSpec interface {
	LimitsSpec

	// AdditionalContents returns true if the item can contain contents not defined in the spec.
	AdditionalContents() bool

	// ContentMediaType returns the expected content type of a file.
	ContentMediaType() *ContentType

	// Contents returns the definitions of the children elements of this item.
	Contents() []ItemSpec

	// DevelopmentFolder returns true if the item is inside a development folder.
	DevelopmentFolder() bool

	// ForbiddenPatterns returns the list of forbidden patterns for the name of this item.
	ForbiddenPatterns() []string

	// IsDir returns true if the item is a directory.
	IsDir() bool

	// Name returns the name of the item inside its parent.
	Name() string

	// Pattern returns the allowed pattern for the name of this item.
	Pattern() string

	// Release returns 'beta' if the item definition is in beta stage.
	Release() string

	// Required returns true if this item must be defined.
	Required() bool

	// Type returns the type of file ('file' or 'folder').
	Type() string

	// ValidateSchema validates if the indicated file complies with the schema of the item.
	ValidateSchema(fsys fs.FS, itemPath string) specerrors.ValidationErrors
}
