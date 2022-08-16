// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spectypes

import (
	"io/fs"

	"github.com/Masterminds/semver/v3"
	ve "github.com/elastic/package-spec/code/go/internal/errors"
)

// FileSchema defines the expected schema for a file.
type FileSchema interface {
	// Validate checks if the file in the given path complies with the schema.
	Validate(fs fs.FS, path string) ve.ValidationErrors
}

// FileSchemaLoader loads schemas for files.
type FileSchemaLoader interface {
	// Load loads an schema from the given path.
	Load(fs fs.FS, specPath string, opts FileSchemaLoadOptions) (FileSchema, error)
}

// FileSchemaLoadOptions provides additional information for package loading.
type FileSchemaLoadOptions struct {
	ContentType *ContentType
	Limits      LimitsSpec
	SpecVersion semver.Version
}
