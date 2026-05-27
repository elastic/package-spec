// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import "github.com/elastic/package-spec/v3/code/go/internal/validator/modes"

// Mode represents the validation mode used when creating a Validator.
type Mode = modes.Mode

const (
	// ModeLegacy is the default validation mode; it preserves byte-for-byte
	// identical behaviour with the existing ValidateFrom* functions.
	ModeLegacy = modes.Legacy

	// ModeSource validates a package as a checked-out source tree.
	// Linked (.link) files are resolved transparently.
	ModeSource = modes.Source

	// ModeBuild validates a package as a build artifact (e.g. a directory
	// produced by elastic-package build). Linked files are blocked.
	ModeBuild = modes.Build
)
