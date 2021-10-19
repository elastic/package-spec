// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
)

type fields []field

type field struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Unit       string `yaml:"unit"`
	MetricType string `yaml:"metric_type"`
	Dimension  bool   `yaml:"dimension"`

	Fields fields `yaml:"fields"`
}

// ValidateFieldGroups verifies if field groups don't have units and metric types defined.
func ValidateFieldGroups(pkgRoot string) ve.ValidationErrors {
	fieldsFiles, err := listFieldsFiles(pkgRoot)
	if err != nil {
		return ve.ValidationErrors{errors.Wrap(err, "can't list fields files")}
	}

	var vErrs ve.ValidationErrors
	for _, fieldsFile := range fieldsFiles {
		unmarshaled, err := unmarshalFields(fieldsFile)
		if err != nil {
			vErrs = append(vErrs, errors.Wrapf(err, `file "%s" is invalid: can't unmarshal fields`, fieldsFile))
		}

		for _, u := range unmarshaled {
			errs := validateFieldUnit(fieldsFile, u)
			if len(errs) > 0 {
				vErrs = append(vErrs, errs...)
			}
		}
	}
	return vErrs
}

func listFieldsFiles(pkgRoot string) ([]string, error) {
	var fieldsFiles []string

	dataStreamDir := filepath.Join(pkgRoot, "data_stream")
	dataStreams, err := ioutil.ReadDir(dataStreamDir)
	if errors.Is(err, os.ErrNotExist) {
		return fieldsFiles, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't list data streams directory")
	}

	for _, dataStream := range dataStreams {
		fieldsDir := filepath.Join(dataStreamDir, dataStream.Name(), "fields")
		fs, err := ioutil.ReadDir(fieldsDir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "can't list fields directory (path: %s)", fieldsDir)
		}

		for _, f := range fs {
			fieldsFiles = append(fieldsFiles, filepath.Join(fieldsDir, f.Name()))
		}
	}

	return fieldsFiles, nil
}

func unmarshalFields(fieldsPath string) (fields, error) {
	content, err := ioutil.ReadFile(fieldsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "can't read file (path: %s)", fieldsPath)
	}

	var f fields
	err = yaml.Unmarshal(content, &f)
	if err != nil {
		return nil, errors.Wrapf(err, "yaml.Unmarshal failed (path: %s)", fieldsPath)
	}
	return f, nil
}

func validateFieldUnit(fieldsFile string, f field) ve.ValidationErrors {
	if f.Type == "group" && f.Unit != "" {
		return ve.ValidationErrors{fmt.Errorf(`file "%s" is invalid: field "%s" can't have unit property'`, fieldsFile, f.Name)}
	}

	if f.Type == "group" && f.MetricType != "" {
		return ve.ValidationErrors{fmt.Errorf(`file "%s" is invalid: field "%s" can't have metric type property'`, fieldsFile, f.Name)}
	}

	if f.Dimension && !isAllowedDimensionType(f.Type) {
		return ve.ValidationErrors{fmt.Errorf(`file "%s" is invalid: field "%s" of type %s can't be a dimension, allowed types for dimensions: %s`, fieldsFile, f.Name, f.Type, strings.Join(allowedDimensionTypes, ", "))}
	}

	var vErrs ve.ValidationErrors
	for _, aField := range f.Fields {
		errs := validateFieldUnit(fieldsFile, aField)
		if len(errs) > 0 {
			vErrs = append(vErrs, errs...)
		}
	}
	return vErrs
}

var allowedDimensionTypes = []string{
	// Keywords
	"constant_keyword", "keyword",

	// Numeric types
	"long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long",

	// IPs
	"ip",
}

func isAllowedDimensionType(fieldType string) bool {
	for _, allowedType := range allowedDimensionTypes {
		if fieldType == allowedType {
			return true
		}
	}

	return false
}
