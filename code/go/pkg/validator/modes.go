// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	validatorinternal "github.com/elastic/package-spec/v3/code/go/internal/validator"
)

type Mode = validatorinternal.Mode

var (
	ModeLegacy Mode = validatorinternal.ModeLegacy
	ModeSource Mode = validatorinternal.ModeSource
	ModeBuild  Mode = validatorinternal.ModeBuild
)
