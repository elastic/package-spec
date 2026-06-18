// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

// Mode represents the validation context: which semantic rules run and how
// linked (.link) files are handled during package validation.
type Mode string

const (
	// LegacyMode preserves the original validation behavior: linked files are
	// resolved transparently and no rules are mode-gated.
	LegacyMode Mode = "legacy"
	// SourceMode validates a package as a checked-out source tree: linked files
	// are resolved transparently and source-only rules are enforced.
	// The string value must match spectypes.ValidationModeSource.
	SourceMode Mode = "source"
	// BuildMode validates a package as a built artifact: linked files are
	// unconditionally blocked and build-only rules are enforced.
	// The string value must match spectypes.ValidationModeBuild.
	BuildMode Mode = "build"
)

// Valid reports whether m is a recognised validation mode.
func (m Mode) Valid() bool {
	switch m {
	case LegacyMode, SourceMode, BuildMode:
		return true
	}
	return false
}
