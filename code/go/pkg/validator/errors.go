// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"github.com/elastic/package-spec/v2/code/go/pkg/specerrors"
)

// ValidationErrors is an Error that contains a iterable collection of validation error messages.
type ValidationErrors specerrors.ValidationErrors // TODO to be removed in package-spec v3
