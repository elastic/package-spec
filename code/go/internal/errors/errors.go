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

// Error returns the message error
func (e *DanglingObjectIDError) Error() string {
	return fmt.Sprintf("file \"%s\" is invalid: dangling reference found: %s (%s)", e.filePath, e.objectID, e.objectType)
}

// File returns the file path where the was raised
func (e *DanglingObjectIDError) File() string {
	return e.filePath
}

// Code returns a unique code assigned to this error
func (e *DanglingObjectIDError) Code() string {
	return "DanglingObjectIDError"
}

// Severity returns the severity level assigned to this error
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

// JSONSchemaError validation error for dangling object IDs
type JSONSchemaError struct {
	field       string
	description string
	filePath    string
}

// Error returns the message error
func (e *JSONSchemaError) Error() string {
	return fmt.Sprintf("field %s: %s", e.field, e.description)
}

// File returns the file path where the was raised
func (e *JSONSchemaError) File() string {
	return e.filePath
}

// Code returns a unique code assigned to this error
func (e *JSONSchemaError) Code() string {
	return "JsonSchemaError"
}

// Severity returns the severity level assigned to this error
func (e *JSONSchemaError) Severity() int {
	return pve.Critical
}

// NewJSONSchemaError creates a new validation error for JSON schema issues
func NewJSONSchemaError(filePath, field, description string) *JSONSchemaError {
	return &JSONSchemaError{
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

// Error returns the message error
func (e *StructuredError) Error() string {
	return e.err.Error()
}

// Code returns a unique code assigned to this error
func (e *StructuredError) Code() string {
	return e.code
}

// Severity returns the severity level assigned to this error
func (e *StructuredError) Severity() int {
	return e.severity
}

// File returns the file path where the was raised
func (e *StructuredError) File() string {
	return e.filePath
}
