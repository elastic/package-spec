// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package modes

// Mode represents the validation context: which semantic rules run and how
// linked (.link) files are handled during package validation.
type Mode string

const (
	// Legacy preserves the original validation behavior: linked files are
	// resolved transparently and no rules are mode-gated.
	Legacy Mode = "legacy"
	// Source validates a package as a checked-out source tree: linked files
	// are resolved transparently and source-only rules are enforced.
	Source Mode = "source"
	// Build validates a package as a built artifact: linked files are
	// unconditionally blocked and build-only rules are enforced.
	Build Mode = "build"
)

// Valid reports whether m is a recognised validation mode.
func (m Mode) Valid() bool {
	switch m {
	case Legacy, Source, Build:
		return true
	}
	return false
}
