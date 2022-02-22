// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"errors"
	"io/fs"
	"os"

	"github.com/elastic/package-spec/code/go/internal/validator"
)

// ValidateFromPath validates a package located at the given path against the
// appropriate specification and returns any errors.
func ValidateFromPath(packageRootPath string) error {
	return ValidateFromFS(packageRootPath, os.DirFS(packageRootPath))
}

// ValidateFromFS validates a package against the appropiate specification and returns any errors.
// Package files are obtained throug the given filesystem.
func ValidateFromFS(rootPath string, fsys fs.FS) error {
	pkg, err := validator.NewPackageFromFS(rootPath, fsys)
	if err != nil {
		return err
	}

	if pkg.SpecVersion == nil {
		return errors.New("could not determine specification version for package")
	}

	spec, err := validator.NewSpec(*pkg.SpecVersion)
	if err != nil {
		return err
	}

	if errs := spec.ValidatePackage(*pkg); errs != nil && len(errs) > 0 {
		return errs
	}

	return nil
}
