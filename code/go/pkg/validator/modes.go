// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"io/fs"

	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/internal/validator/modes"
)

// Mode represents the validation context: which rules apply and how linked files
// are handled when creating a Validator from a path.
// Use ModeLegacy, ModeSource, or ModeBuild.
type Mode struct {
	internal modes.Mode
	// wrapFS builds the filesystem for path-based validation (NewFromPath).
	// It is not applied by NewFromFS, which takes the caller's filesystem as-is.
	wrapFS func(location string, fsys fs.FS) fs.FS
}

var (
	// ModeLegacy preserves the original validation behavior: linked (.link) files
	// are resolved transparently and no rules are mode-gated.
	// Use this mode when backward compatibility with existing callers is required.
	ModeLegacy = Mode{
		internal: modes.Legacy,
		wrapFS: func(location string, fsys fs.FS) fs.FS {
			return linkedfiles.NewFS(location, fsys)
		},
	}

	// ModeSource validates a package as a checked-out source tree.
	// Linked (.link) files are resolved transparently.
	// Source-only rules (e.g. dev-folder checks) are enforced; build-only rules are skipped.
	ModeSource = Mode{
		internal: modes.Source,
		wrapFS: func(location string, fsys fs.FS) fs.FS {
			return linkedfiles.NewFS(location, fsys)
		},
	}

	// ModeBuild validates a package as a built artifact — the output of
	// elastic-package build, a zip archive, or a package served by the registry.
	// Linked (.link) files are unconditionally blocked.
	// Build-only rules are enforced; source-only rules are skipped.
	ModeBuild = Mode{
		internal: modes.Build,
		wrapFS: func(_ string, fsys fs.FS) fs.FS {
			return linkedfiles.NewBlockFS(fsys)
		},
	}
)
