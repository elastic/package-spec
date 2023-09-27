// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specerrors

// Processor is the interface for every processor.
type Processor interface {
	Process(ValidationErrors) (ProcessResult, error)
	Name() string
}

// ProcessResult represents the errors that have been processed and removed
type ProcessResult struct {
	Processed ValidationErrors
	Removed   ValidationErrors
}
