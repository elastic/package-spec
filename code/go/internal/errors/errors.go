// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import (
	"fmt"

	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidationPathError is the interface that validation errors related to paths must implement.
type ValidationPathError interface {
	// File returns the file path where the error was raised.
	File() string
}

// StructuredError generic validation error
type StructuredError struct {
	err      error
	filePath string
	code     string // TODO : generate constants and types for each kind of error/code
	severity int
}

// DanglingObjectIDError validation error for dangling object IDs
type DanglingObjectIDError struct {
	objectID   string
	objectType string
	filePath   string
}

func (e *DanglingObjectIDError) Error() string {
	return fmt.Sprintf("file \"%s\" is invalid: dangling reference found: %s (%s)", e.filePath, e.objectID, e.objectType)
}

func (e *DanglingObjectIDError) File() string {
	return e.filePath
}

func (e *DanglingObjectIDError) Code() string {
	return "DanglingObjectIDError"
}

func (e *DanglingObjectIDError) Severity() int {
	return pve.Critical
}

// NewDanglingObjectIDError creates a new validation error for dangling object IDs
func NewDanglingObjectIDError(objectID, objectType, filePath string) *DanglingObjectIDError {
	return &DanglingObjectIDError{
		objectID:   objectID,
		objectType: objectType,
		filePath:   filePath,
	}
}

// JsonSchemaError validation error for dangling object IDs
type JsonSchemaError struct {
	field       string
	description string
	filePath    string
}

func (e *JsonSchemaError) Error() string {
	return fmt.Sprintf("field %s: %s", e.field, e.description)
}

func (e *JsonSchemaError) File() string {
	return e.filePath
}

func (e *JsonSchemaError) Code() string {
	return "JsonSchemaError"
}

func (e *JsonSchemaError) Severity() int {
	return pve.Critical
}

// NewJsonSchemaError creates a new validation error for JSON schema issues
func NewJsonSchemaError(filePath, field, description string) *JsonSchemaError {
	return &JsonSchemaError{
		field:       field,
		description: description,
		filePath:    filePath,
	}
}

// NewStructuredError creates a generic validation error
func NewStructuredError(err error, filePath, code string, level int) *StructuredError {
	return &StructuredError{
		err:      err,
		filePath: filePath,
		code:     code,
		severity: level,
	}
}

func (e *StructuredError) Error() string {
	return e.err.Error()
}

func (e *StructuredError) Code() string {
	return e.code
}

func (e *StructuredError) Severity() int {
	return e.severity
}

func (e *StructuredError) File() string {
	return e.filePath
}
