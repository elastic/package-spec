// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package packages

import (
	"path"

	"github.com/Masterminds/semver/v3"
)

// GroupType is the name of the folder spec describing a group directory. It is
// not a package type: a group is never built, published or installed.
const GroupType = "group"

// NewGroupDirectory creates a Package representing the group directory at the
// given path.
//
// A group directory has no manifest, so it has no name, version or
// format_version of its own: it takes its name from the folder, the spec version
// it is validated against, and a version below 1.0.0 so that it is never GA.
func NewGroupDirectory(groupRootPath string, specVersion semver.Version) (*Package, error) {
	fsys, err := openRootFolder(groupRootPath, "group")
	if err != nil {
		return nil, err
	}

	return &Package{
		Name:        path.Base(groupRootPath),
		Type:        GroupType,
		Version:     semver.MustParse("0.0.0"),
		SpecVersion: &specVersion,
		fs:          fsys,

		location: groupRootPath,
	}, nil
}
