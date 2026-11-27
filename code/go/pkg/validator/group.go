// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Masterminds/semver/v3"

	spec "github.com/elastic/package-spec/v3"
	"github.com/elastic/package-spec/v3/code/go/internal/packages"
	"github.com/elastic/package-spec/v3/code/go/internal/validator"
)

// ValidateGroupDirectory validates the group directory at the given path.
//
// A group directory holds the shared definition of a group of packages: a
// group.yml file, an img folder with its icon, and one directory per member
// package. It is not a package: it is never built, published or installed, so it
// has no manifest and no format_version, and is always validated against the
// current version of the spec. At build time elastic-package copies group.yml
// and its icon into each member package instead.
//
// Member packages are only checked far enough to confirm they are packages of a
// type that can belong to a group. Validating a member in full means validating
// it as a package of its own type, with ValidateFromPath or one of its siblings.
func ValidateGroupDirectory(groupRootPath string, opts ...Option) error {
	// A group directory only ever exists in source, as it is never built, so the
	// mode is not a caller's choice. The validator is built here for its options.
	v, err := New(SourceMode, opts...)
	if err != nil {
		return err
	}

	specVersion, err := latestSpecVersion()
	if err != nil {
		return fmt.Errorf("could not determine the current spec version: %w", err)
	}

	group, err := packages.NewGroupDirectory(groupRootPath, specVersion)
	if err != nil {
		return err
	}

	s, err := validator.NewSpec(specVersion, SourceMode)
	if err != nil {
		return err
	}
	s.WarningsAsErrors = v.warningsAsErrors

	if errs := s.ValidateGroupDirectory(group); len(errs) > 0 {
		return errs
	}
	return nil
}

// latestSpecVersion returns the most recent version defined in the spec
// changelog, with any prerelease tag stripped so that it can be passed to the
// validator. A group directory has no format_version of its own, so it is always
// validated against this version.
func latestSpecVersion() (semver.Version, error) {
	versions, err := spec.VersionsInChangelog()
	if err != nil {
		return semver.Version{}, err
	}
	if len(versions) == 0 {
		return semver.Version{}, errors.New("no versions found in spec changelog")
	}

	// The changelog is written newest-first, but nothing enforces that ordering.
	latest := slices.MaxFunc(versions, func(a, b semver.Version) int {
		return a.Compare(&b)
	})
	released, err := latest.SetPrerelease("")
	if err != nil {
		return semver.Version{}, fmt.Errorf("failed to read latest spec version (%s): %w", latest.String(), err)
	}
	return released, nil
}
