// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateDurationVariables checks all duration type variables in the package
// to ensure that their min_duration, max_duration, and default values follow
// the expected constraints: 0 <= min_duration <= default <= max_duration.
//
// It examines both the root manifest.yml file and all data stream manifests
// to find and validate duration variables.
func ValidateDurationVariables(fsys fspath.FS) specerrors.ValidationErrors {
	// Load main manifest vars.
	data, err := fs.ReadFile(fsys, "manifest.yml")
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("%s failed to read manifest: %w", fsys.Path("manifest.yml"), err)}
	}

	var manifest durationManifest
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("%s is invalid: failed to parse manifest: %w", fsys.Path("manifest.yml"), err)}
	}
	annotateFileMetadata(fsys.Path("manifest.yml"), manifest)
	vars := manifest.allVars()

	dsManifests, err := fs.Glob(fsys, "data_stream/*/manifest.yml")
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to list data streams: %w", err)}
	}

	// Load data stream manifest vars.
	for _, path := range dsManifests {
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("%s failed to read data stream manifest: %w", fsys.Path(path), err)}
		}

		var manifest durationDataStreamManifest
		err = yaml.Unmarshal(data, &manifest)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("%s is invalid: failed to parse data stream manifest: %w", fsys.Path(path), err)}
		}
		annotateFileMetadata(fsys.Path(path), manifest)
		vars = append(vars, manifest.allVars()...)
	}

	// Validate duration vars.
	var errs specerrors.ValidationErrors
	for _, v := range vars {
		if err := validateDurationVar(v); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("%s:%d:%d error in variable %q: %w", v.Node.Path, v.Node.Line, v.Node.Col, v.Name, err))
		}
	}

	return errs
}

// nodeMetadata contains location information for YAML nodes in a file.
// This helps pinpoint the exact location of validation errors.
type nodeMetadata struct {
	// Path to the file containing the node
	Path string
	// Line number (1-based)
	Line int
	// Column number (1-based)
	Col int
}

// durationVar represents a variable with duration constraints from a manifest
// file. It includes fields for the variable name, type, duration constraints,
// default value, and location information within the source file.
type durationVar struct {
	// Variable name
	Name string `yaml:"name"`
	// Variable type (must be "duration" for validation)
	Type string `yaml:"type"`
	// Minimum allowed duration (optional)
	MinDuration *string `yaml:"min_duration,omitempty"`
	// Maximum allowed duration (optional)
	MaxDuration *string `yaml:"max_duration,omitempty"`
	// Default value (optional)
	Default any `yaml:"default,omitempty"`

	// Location information in the source file
	Node nodeMetadata `yaml:"-"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for durationVar. It
// unmarshals the YAML data into a durationVar struct and also captures the
// position information from the yaml.Node.
func (v *durationVar) UnmarshalYAML(node *yaml.Node) error {
	// Prevent recursion by creating a type alias that doesn't implement Unmarshaler.
	type notVar durationVar
	x := (*notVar)(v)

	if err := node.Decode(&x); err != nil {
		return err
	}
	v.Node.Line = node.Line
	v.Node.Col = node.Column
	return nil
}

// durationManifest represents the structure of a package manifest file focusing
// on the elements that may contain duration variables.
type durationManifest struct {
	Vars            []durationVar `yaml:"vars"` // Top-level variables
	PolicyTemplates []struct {
		Vars   []durationVar `yaml:"vars"` // Policy template variables
		Inputs []struct {
			Vars []durationVar `yaml:"vars"` // Input variables
		} `yaml:"inputs"`
	} `yaml:"policy_templates"`
}

// allVars collects all duration variables from a manifest including those at the
// top level, within policy templates, and nested inside inputs. It returns a
// flattened slice of all variables for easier processing.
func (m *durationManifest) allVars() []durationVar {
	var out []durationVar
	out = append(out, m.Vars...)
	for _, t := range m.PolicyTemplates {
		out = append(out, t.Vars...)
		for _, i := range t.Inputs {
			out = append(out, i.Vars...)
		}
	}
	return out
}

// durationDataStreamManifest represents the structure of a data stream manifest
// file focusing on the stream elements that may contain duration variables.
type durationDataStreamManifest struct {
	Streams []struct {
		Vars []durationVar `yaml:"vars"` // Stream variables
	} `yaml:"streams"`
}

// allVars collects all duration variables from a data stream manifest. It
// returns a flattened slice of all variables from all streams for easier
// processing.
func (m *durationDataStreamManifest) allVars() []durationVar {
	var out []durationVar
	for _, s := range m.Streams {
		out = append(out, s.Vars...)
	}
	return out
}

// validateDurationVar checks that duration variables follow the required constraints:
//
// 0 <= min_duration <= default <= max_duration, for variables of type "duration".
//
// It parses the duration strings, validates their format, and ensures they meet
// the ordering constraints. Multiple validation errors may be returned together
// using errors.Join if the variable has multiple constraint violations.
func validateDurationVar(v durationVar) error {
	// Only validate variables of the type "duration".
	if v.Type != "duration" {
		return nil
	}

	var minDuration, defaultDuration, maxDuration time.Duration
	var err error

	// Parse min_duration if defined
	if v.MinDuration != nil {
		minDuration, err = time.ParseDuration(*v.MinDuration)
		if err != nil {
			return fmt.Errorf("invalid min_duration value %q: %w", *v.MinDuration, err)
		}
		// Ensure min_duration is not negative
		if minDuration.Nanoseconds() < 0 {
			return fmt.Errorf("negative min_duration value %q", *v.MinDuration)
		}
	}

	// Parse default if defined
	switch s := v.Default.(type) {
	case nil:
	case string:
		defaultDuration, err = time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("invalid default value %q: %w", s, err)
		}
	default:
		return fmt.Errorf("invalid default value type %T: expected a string", s)
	}

	// Parse max_duration if defined
	if v.MaxDuration != nil {
		maxDuration, err = time.ParseDuration(*v.MaxDuration)
		if err != nil {
			return fmt.Errorf("invalid max_duration value %q: %w", *v.MaxDuration, err)
		}
	}

	// Check constraints: 0 <= min_duration <= default <= max_duration
	var errs []error

	// Check min_duration <= default (if both are defined)
	if v.MinDuration != nil && v.Default != nil {
		if minDuration.Nanoseconds() > defaultDuration.Nanoseconds() {
			errs = append(errs, fmt.Errorf("min_duration %q greater than default %q", *v.MinDuration, v.Default))
		}
	}

	// Check default <= max_duration (if both are defined)
	if v.Default != nil && v.MaxDuration != nil {
		if defaultDuration.Nanoseconds() > maxDuration.Nanoseconds() {
			errs = append(errs, fmt.Errorf("default %q greater than max_duration %q", v.Default, *v.MaxDuration))
		}
	}

	// Check min_duration <= max_duration (if both are defined)
	if v.MinDuration != nil && v.MaxDuration != nil {
		if minDuration.Nanoseconds() > maxDuration.Nanoseconds() {
			errs = append(errs, fmt.Errorf("min_duration %q greater than max_duration %q", *v.MinDuration, *v.MaxDuration))
		}
	}

	return errors.Join(errs...)
}

// annotateFileMetadata recursively sets the file path on any nodeMetadata fields
// found within the provided value. This ensures that all node metadata entries
// contain the correct file path for accurate error reporting.
//
// It uses reflection to traverse the object structure and annotate all
// nodeMetadata instances with the provided file path.
func annotateFileMetadata(file string, v any) {
	fileAnnotator{Name: file}.Annotate(reflect.ValueOf(v))
}

// fileAnnotator is a helper type for annotating node metadata with a file name.
// It keeps track of the file name being applied to nodeMetadata instances.
type fileAnnotator struct {
	// Name is the file path to be applied to nodeMetadata instances
	Name string
}

// Annotate recursively traverses the provided reflect.Value and sets the Name
// field on any nodeMetadata objects it finds. It handles various value kinds,
// including pointers, structs, slices, and maps.
func (a fileAnnotator) Annotate(val reflect.Value) {
	// Need an addressable value to edit the metadata value.
	if val.CanAddr() && val.CanSet() {
		if m, ok := val.Addr().Interface().(*nodeMetadata); ok {
			m.Path = a.Name
			return
		}
	}

	switch val.Kind() {
	case reflect.Pointer:
		a.Annotate(val.Elem())
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			valueField := val.Field(i)
			a.Annotate(valueField)
		}
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			a.Annotate(val.Index(i))
		}
	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			// NOTE: This can only edit the map value if it is addressable (i.e., a pointer).
			a.Annotate(iter.Value())
		}
	}
}
