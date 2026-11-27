// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// groupDefinitionFile is the group definition copied into each member package.
const groupDefinitionFile = "group.yml"

type groupMembershipManifest struct {
	Group string `yaml:"group"`
}

// ValidateGroupFileRequiresGroupField errors when a built package carries a
// group.yml but its manifest does not declare the group it belongs to.
//
// Both are written by the same build step: elastic-package copies group.yml and
// its icon out of the parent group directory and sets group to the name of that
// directory. A package carrying the file without the field means only half of
// that step ran, and since packages are grouped by the manifest field, the
// copied definition would never be used.
//
// The reverse is allowed on purpose: a package may declare a group without
// carrying a group.yml. That is how packages outside a group directory join a
// group today, and those definitions come from elsewhere.
func ValidateGroupFileRequiresGroupField(fsys fspath.FS) specerrors.ValidationErrors {
	if _, err := fs.Stat(fsys, groupDefinitionFile); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("reading %q: %w", fsys.Path(groupDefinitionFile), err),
		}
	}

	const manifestRelPath = "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestRelPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("reading %q: %w", fsys.Path(manifestRelPath), err),
		}
	}

	var manifest groupMembershipManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(
				"file %q: failed to parse manifest: %w", fsys.Path(manifestRelPath), err),
		}
	}

	if manifest.Group == "" {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(
				"file %q: package contains a %q file but does not define the group it belongs to in the manifest",
				fsys.Path(manifestRelPath), groupDefinitionFile),
		}
	}

	return nil
}
