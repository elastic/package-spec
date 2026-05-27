// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package modes

type Mode string

const (
	Legacy Mode = "legacy"
	Source Mode = "source"
	Build  Mode = "build"
)

func (m Mode) Valid() bool {
	switch m {
	case Legacy, Source, Build:
		return true
	}
	return false
}
