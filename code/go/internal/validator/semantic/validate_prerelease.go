// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	// Prereleases allowed as literals.
	literalPrereleases = []string{
		// For convenience with previous recommendations
		"next",
		"SNAPSHOT",
	}

	// Prereleases allowed, potentially with additional numbering.
	numberedPrereleases = []string{
		"beta",
		"rc",
		"preview",
	}
)

// ValidatePrerelease validates additional restrictions on the prerelease tags.
func ValidatePrerelease(fsys fspath.FS) specerrors.ValidationErrors {
	manifestVersion, err := readManifestVersion(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	err = validatePrerelease(manifestVersion)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	return nil
}

func validatePrerelease(manifestVersion string) error {
	version, err := semver.NewVersion(manifestVersion)
	if err != nil {
		return err
	}

	if version.Major() == 0 && version.Prerelease() != "" {
		return fmt.Errorf("versions below 1.0.0 are considered technical previews, please remove prerelease tag (version: %s)", manifestVersion)
	}

	if err := validatePrereleaseTag(version.Prerelease()); err != nil {
		return err
	}

	return nil
}

// prereleaseNumberPattern is the pattern that the number after the prerelease tag must match.
// It has to start with a number, hyphen or dot, and end with number or letter.
const prereleaseNumberPattern = "(([0-9]|[.-][0-9A-Za-z])([0-9A-Za-z-.]*[0-9A-Za-z])?)?"

func validatePrereleaseTag(tag string) error {
	if tag == "" {
		return nil
	}

	for _, literal := range literalPrereleases {
		if tag == literal {
			return nil
		}
	}

	for _, numbered := range numberedPrereleases {
		if tag == numbered {
			return nil
		}

		pattern := regexp.MustCompile(fmt.Sprintf("^%s%s$", numbered, prereleaseNumberPattern))
		if pattern.MatchString(tag) {
			return nil
		}
	}

	return fmt.Errorf("prerelease tag (%s) should be one of [%s], or one of [%s] followed by numbers",
		tag,
		strings.Join(literalPrereleases, ", "),
		strings.Join(numberedPrereleases, ", "),
	)
}
