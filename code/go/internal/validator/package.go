// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Package represents an Elastic Package Registry package
type Package struct {
	Name        string
	Type        string
	SpecVersion *semver.Version

	fs       fs.FS
	location string
}

// Open opens a file in the package filesystem.
func (p *Package) Open(name string) (fs.File, error) {
	return p.fs.Open(name)
}

// Path returns a path meaningful for the user.
func (p *Package) Path(names ...string) string {
	return filepath.Join(append([]string{p.location}, names...)...)
}

// NewPackage creates a new Package from a path to the package's root folder
func NewPackage(pkgRootPath string) (*Package, error) {
	info, err := os.Stat(pkgRootPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "no package found at path [%v]", pkgRootPath)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("no package folder found at path [%v]", pkgRootPath)
	}

	return NewPackageFromFS(pkgRootPath, os.DirFS(pkgRootPath))
}

// NewPackageFromFS creates a new package from a given filesystem. A root path can be indicated
// to help building paths meaningful for the users.
func NewPackageFromFS(location string, fsys fs.FS) (*Package, error) {
	pkgManifestPath := "manifest.yml"
	_, err := fs.Stat(fsys, pkgManifestPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "no package manifest file found at path [%v]", pkgManifestPath)
	}

	data, err := fs.ReadFile(fsys, pkgManifestPath)
	if err != nil {
		return nil, fmt.Errorf("could not read package manifest file [%v]", pkgManifestPath)
	}

	var manifest struct {
		Name        string `yaml:"name"`
		Type        string `yaml:"type"`
		SpecVersion string `yaml:"format_version"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, errors.Wrapf(err, "could not parse package manifest file [%v]", pkgManifestPath)
	}

	if manifest.Type == "" {
		return nil, errors.New("package type undefined in the package manifest file")
	}

	specVersion, err := semver.NewVersion(manifest.SpecVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read specification version from package manifest file [%v]", pkgManifestPath)
	}

	// Instantiate Package object and return it
	p := Package{
		Name:        manifest.Name,
		Type:        manifest.Type,
		SpecVersion: specVersion,
		fs:          fsys,

		location: location,
	}

	return &p, nil
}
