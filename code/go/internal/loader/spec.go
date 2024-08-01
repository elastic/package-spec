// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loader

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/specschema"
	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
	"github.com/elastic/package-spec/v3/code/go/internal/yamlschema"
)

// LoadSpec loads a package specification for the given version and type.
func LoadSpec(fsys fs.FS, version semver.Version, pkgType string) (spectypes.ItemSpec, error) {
	fileSpecLoader := yamlschema.NewFileSchemaLoader()
	loader := specschema.NewFolderSpecLoader(fsys, fileSpecLoader, version)
	spec, err := loader.Load(pkgType)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("package type %q not supported (%w)", pkgType, err)
	}
	if err != nil {
		return nil, err
	}
	return spec, nil
}
