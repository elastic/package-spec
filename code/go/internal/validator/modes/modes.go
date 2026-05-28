// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package modes

// Mode represents the validation mode used when validating a package.
type Mode string

// Validation modes. The public API re-exports these as validator.Mode* constants.
const (
	Legacy Mode = "legacy"
	Source Mode = "source"
	Build  Mode = "build"
)

// Valid reports whether m is a recognised validation mode.
func (m Mode) Valid() bool {
	switch m {
	case Legacy, Source, Build:
		return true
	}
	return false
}
