// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import "io/fs"

// ValidateFromPath validates a package located at the given path against the
// appropriate specification and returns any errors.
//
// Deprecated: use NewFromPath(ModeLegacy, packageRootPath).Validate() instead.
func ValidateFromPath(packageRootPath string) error {
	v, err := NewFromPath(ModeLegacy, packageRootPath)
	if err != nil {
		return err
	}
	return v.Validate()
}

// ValidateFromZip validates a package in zip format.
//
// Deprecated: use NewFromZip(packagePath).Validate() instead.
func ValidateFromZip(packagePath string) error {
	v, err := newFromZip(ModeLegacy, packagePath)
	if err != nil {
		return err
	}
	return v.Validate()
}

// ValidateFromFS validates a package against the appropriate specification and
// returns any errors. Package files are obtained through the given filesystem.
//
// Deprecated: use NewFromFS(ModeLegacy, location, fsys).Validate() instead.
func ValidateFromFS(location string, fsys fs.FS) error {
	v, err := NewFromFS(ModeLegacy, location, fsys)
	if err != nil {
		return err
	}
	return v.Validate()
}
