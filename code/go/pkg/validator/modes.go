package validator

import (
	"io/fs"

	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/internal/validator/modes"
)

// Mode represents the validation context: which rules apply and how linked files
// are handled when creating a Validator from a path.
// Use ModeLegacy, ModeSource, or ModeBuild; do not construct Mode directly.
type Mode struct {
	internal modes.Mode
	// wrapFS builds the filesystem for path-based validation (NewFromPath).
	// It is not applied by NewFromFS, which takes the caller's filesystem as-is.
	wrapFS func(location string, fsys fs.FS) fs.FS
}

var (
	// ModeLegacy preserves byte-for-byte identical behaviour with ValidateFromPath.
	// Linked (.link) files are resolved transparently.
	ModeLegacy = Mode{
		internal: modes.Legacy,
		wrapFS: func(location string, fsys fs.FS) fs.FS {
			return linkedfiles.NewFS(location, fsys)
		},
	}

	// ModeSource validates a package as a checked-out source tree.
	// Linked (.link) files are resolved transparently.
	ModeSource = Mode{
		internal: modes.Source,
		wrapFS: func(location string, fsys fs.FS) fs.FS {
			return linkedfiles.NewFS(location, fsys)
		},
	}

	// ModeBuild validates a package as a build artifact — output of elastic-package
	// build, a zip archive, or a package served by the registry.
	// Linked (.link) files are unconditionally blocked.
	ModeBuild = Mode{
		internal: modes.Build,
		wrapFS: func(_ string, fsys fs.FS) fs.FS {
			return linkedfiles.NewBlockFS(fsys)
		},
	}
)
