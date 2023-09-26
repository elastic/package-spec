// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"io/fs"
	"path"

	"github.com/elastic/gojsonschema"

	"github.com/tommyers-elastic/package-spec/v2/code/go/internal/spectypes"
)

const (
	// relativePathFormat defines the ID of the relative path format checker. This format checker
	// should be used when a field's value refers to a relative filesystem path. The checker will
	// ensure that the location pointed to by that relative filesystem path actually exists on
	// the filesystem, relative to the file in which the field is defined.
	relativePathFormat = "relative-path"

	// dataStreamNameFormat defines the ID of the data stream name format checker. This format checker
	// should be used when a field's value refers to a data stream name. The checker will ensure
	// that a folder with that data stream name exists on the filesystem.
	dataStreamNameFormat = "data-stream-name"
)

// relativePathChecker is responsible for checking presence of the file path
type relativePathChecker struct {
	fsys        fs.FS
	currentPath string
	sizeLimit   spectypes.FileSize
}

// IsFormat method checks if the path exists.
func (r relativePathChecker) IsFormat(input interface{}) bool {
	asString, ok := input.(string)
	if !ok {
		return false
	}

	path := path.Join(r.currentPath, asString)
	info, err := fs.Stat(r.fsys, path)
	if err != nil {
		return false
	}

	// TODO: It happens that we want the same max size for all the files we reference with
	// relative paths, but it'd be better if we could find a way to parameterize format
	// checkers so we can configure specific max sizes, and we can provide better feedback.
	if r.sizeLimit > 0 && spectypes.FileSize(info.Size()) > r.sizeLimit {
		return false
	}

	return true
}

// loadRelativePathFormatChecker loads the relative-path format checker into the
// json-schema validation library.
func loadRelativePathFormatChecker(fsys fs.FS, currentPath string, sizeLimit spectypes.FileSize) {
	gojsonschema.FormatCheckers.Add(relativePathFormat, relativePathChecker{
		fsys:        fsys,
		currentPath: currentPath,
		sizeLimit:   sizeLimit,
	})
}

// unloadRelativePathFormatChecker unloads the relative-path format checker from the
// json-schema validation library.
func unloadRelativePathFormatChecker() {
	gojsonschema.FormatCheckers.Remove(relativePathFormat)
}

// loadDataStreamNameFormatChecker loads the data-stream-name format checker into the
// json-schema validation library.
func loadDataStreamNameFormatChecker(fsys fs.FS, currentPath string) {
	gojsonschema.FormatCheckers.Add(dataStreamNameFormat, relativePathChecker{
		fsys:        fsys,
		currentPath: path.Join(currentPath, "data_stream"),
	})
}

// unloadDataStreamNameFormatChecker unloads the data-stream-name format checker from the
// json-schema validation library.
func unloadDataStreamNameFormatChecker() {
	gojsonschema.FormatCheckers.Remove(dataStreamNameFormat)
}
