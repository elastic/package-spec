// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/packages"
)

const dataStreamDir = "data_stream"

type fields []field

type field struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Unit       string `yaml:"unit"`
	DateFormat string `yaml:"date_format"`
	MetricType string `yaml:"metric_type"`
	Dimension  bool   `yaml:"dimension"`
	External   string `yaml:"external"`

	Fields fields `yaml:"fields"`
}

type fieldFileMetadata struct {
	packageType  string
	packageName  string
	dataStream   string
	filePath     string
	fullFilePath string
}

type validateFunc func(fileMetadata fieldFileMetadata, f field) ve.ValidationErrors

func validateFields(fsys fspath.FS, validate validateFunc) ve.ValidationErrors {

	fieldsFilesMetadata, err := listFieldsFiles(fsys)
	if err != nil {
		return ve.ValidationErrors{errors.Wrap(err, "can't list fields files")}
	}

	var vErrs ve.ValidationErrors
	for _, metadata := range fieldsFilesMetadata {
		unmarshaled, err := unmarshalFields(fsys, metadata.filePath)
		if err != nil {
			vErrs = append(vErrs, errors.Wrapf(err, `file "%s" is invalid: can't unmarshal fields`, fsys.Path(metadata.filePath)))
		}

		errs := validateNestedFields("", metadata, unmarshaled, validate)
		if len(errs) > 0 {
			vErrs = append(vErrs, errs...)
		}
	}
	return vErrs
}

func validateNestedFields(parent string, metadata fieldFileMetadata, fields fields, validate validateFunc) ve.ValidationErrors {
	var result ve.ValidationErrors
	for _, field := range fields {
		if len(parent) > 0 {
			field.Name = parent + "." + field.Name
		}
		errs := validate(metadata, field)
		if len(errs) > 0 {
			result = append(result, errs...)
		}
		if len(field.Fields) > 0 {
			errs := validateNestedFields(field.Name, metadata, field.Fields, validate)
			if len(errs) > 0 {
				result = append(result, errs...)
			}
		}
	}
	return result
}

func listFieldsFiles(fsys fspath.FS) ([]fieldFileMetadata, error) {
	var fieldsFilesMetadata []fieldFileMetadata

	pkg, err := packages.NewPackageFromFS(fsys.Path(), fsys)
	if err != nil {
		return nil, ve.ValidationErrors{err}
	}

	// integration packages
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return nil, err
	}

	for _, dataStream := range dataStreams {
		fieldsDir := path.Join(dataStreamDir, dataStream, "fields")
		integrationFieldsFiles, err := readFieldsFolder(fsys, fieldsDir)
		if err != nil {
			return nil, fmt.Errorf("cannot read fields file from integration packages: %w", err)
		}

		for _, file := range integrationFieldsFiles {
			fieldsFilesMetadata = append(
				fieldsFilesMetadata,
				fieldFileMetadata{
					filePath:     file,
					fullFilePath: fsys.Path(file),
					dataStream:   dataStream,
					packageName:  pkg.Name,
					packageType:  pkg.Type,
				})
		}
	}

	// input packages
	inputFieldsFiles, err := readFieldsFolder(fsys, "fields")
	if err != nil {
		return nil, fmt.Errorf("cannot read fields file from input packages: %w", err)
	}

	for _, file := range inputFieldsFiles {
		fieldsFilesMetadata = append(
			fieldsFilesMetadata,
			fieldFileMetadata{
				filePath:     file,
				fullFilePath: fsys.Path(file),
				dataStream:   "",
				packageName:  pkg.Name,
				packageType:  pkg.Type,
			})
	}

	return fieldsFilesMetadata, nil
}

func readFieldsFolder(fsys fspath.FS, fieldsDir string) ([]string, error) {
	var fieldsFiles []string
	fs, err := fs.ReadDir(fsys, fieldsDir)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "can't list fields directory (path: %s)", fsys.Path(fieldsDir))
	}

	for _, f := range fs {
		fieldsFiles = append(fieldsFiles, path.Join(fieldsDir, f.Name()))
	}
	return fieldsFiles, nil
}

func unmarshalFields(fsys fspath.FS, fieldsPath string) (fields, error) {
	content, err := fs.ReadFile(fsys, fieldsPath)
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

func listDataStreams(fsys fspath.FS) ([]string, error) {
	dataStreams, err := fs.ReadDir(fsys, dataStreamDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't list data streams directory")
	}

	list := make([]string, len(dataStreams))
	for i, dataStream := range dataStreams {
		list[i] = dataStream.Name()
	}
	return list, nil
}
