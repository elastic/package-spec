// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loader

import (
	"io/fs"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/specschema"
	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
	"github.com/elastic/package-spec/v3/code/go/internal/yamlschema"
)

// LoadSpec loads a package specification for the given version and type.
func LoadSpec(fsys fs.FS, version semver.Version, pkgType string) (spectypes.ItemSpec, error) {
	fileSpecLoader := yamlschema.NewFileSchemaLoader()
	loader := specschema.NewFolderSpecLoader(fsys, fileSpecLoader, version)
	return loader.Load(pkgType)
}
