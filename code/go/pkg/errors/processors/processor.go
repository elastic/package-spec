// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import "github.com/elastic/package-spec/v2/code/go/pkg/errors"

// Processor is the interface for every processor.
type Processor interface {
	Process(errors.ValidationErrors) (ProcessResult, error)
	Name() string
}

// ProcessResult represents the errors that have been processed and removed
type ProcessResult struct {
	Processed errors.ValidationErrors
	Removed   errors.ValidationErrors
}
