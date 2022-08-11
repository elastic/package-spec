// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spectypes

import (
	"io/fs"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
)

const (
	ItemTypeFile   = "file"
	ItemTypeFolder = "folder"
)

type LimitsSpec interface {
	MaxTotalContents() int
	MaxTotalSize() FileSize
	MaxFileSize() FileSize
	MaxConfigurationSize() FileSize
	MaxRelativePathSize() FileSize
	MaxFieldsPerDataStream() int
}

type ItemSpec interface {
	LimitsSpec

	AdditionalContents() bool
	ContentMediaType() *ContentType
	Contents() []ItemSpec
	DevelopmentFolder() bool
	ForbiddenPatterns() []string
	IsDir() bool
	Name() string
	Pattern() string
	Release() string
	Required() bool
	Type() string

	// Schema validation
	ValidateSchema(fsys fs.FS, itemPath string) ve.ValidationErrors
}
