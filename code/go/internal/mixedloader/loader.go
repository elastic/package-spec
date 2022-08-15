// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mixedloader

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/elastic/package-spec/code/go/internal/cueschema"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/elastic/package-spec/code/go/internal/yamlschema"
)

// FileSchemaLoader can load schemas from different formats based on the extension of the file.
type FileSchemaLoader struct {
	cueschema  *cueschema.FileSchemaLoader
	yamlschema *yamlschema.FileSchemaLoader
}

// NewFileSchemaLoader builds a new FileSchemaLoader.
func NewFileSchemaLoader() *FileSchemaLoader {
	return &FileSchemaLoader{
		cueschema:  cueschema.NewFileSchemaLoader(),
		yamlschema: yamlschema.NewFileSchemaLoader(),
	}
}

// Load loads a schema from a file in the given filesystem. It uses a different decoder depending on the
// extension of the file.
func (f *FileSchemaLoader) Load(fs fs.FS, schemaPath string, options spectypes.FileSchemaLoadOptions) (spectypes.FileSchema, error) {
	parts := strings.SplitN(schemaPath, "#", 2)
	switch path.Ext(parts[0]) {
	case ".yml", ".spec.yml":
		return f.yamlschema.Load(fs, schemaPath, options)
	case ".cue", ".spec.cue":
		return f.cueschema.Load(fs, schemaPath, options)
	}
	return nil, fmt.Errorf("not implemented loading for %q (decided by extension: %q)", schemaPath, path.Ext(parts[0]))
}
