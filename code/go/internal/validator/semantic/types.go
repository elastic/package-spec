// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const dataStreamDir = "data_stream"

type fields []field

type runtimeField struct {
	enabled bool
	script  string
}

// Ensure runtime implements these interfaces.
var (
	_ json.Unmarshaler = new(runtimeField)
	_ yaml.Unmarshaler = new(runtimeField)
)

func (r *runtimeField) isEnabled() bool {
	if r.enabled {
		return true
	}
	if r.script != "" {
		return true
	}
	return false
}

func (r runtimeField) String() string {
	if r.script != "" {
		return r.script
	}
	return strconv.FormatBool(r.enabled)
}

func (r *runtimeField) unmarshalString(text string) error {
	value, err := strconv.ParseBool(text)
	if err == nil {
		r.enabled = value
		return nil
	}

	// JSONSchema already checks about the type of this field (e.g. int or float)
	r.enabled = true
	r.script = text
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for field
func (r *runtimeField) UnmarshalJSON(data []byte) error {
	var alias interface{}
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	switch v := alias.(type) {
	case bool:
		r.enabled = v
	case string:
		r.enabled = true
		r.script = v
	default:
		// JSONSchema already checks about the type of this field (e.g. int or float)
		r.enabled = true
		r.script = string(data)
	}
	return nil
}

// UnmarshalYAML implements the yaml.Marshaler interface for runtime.
func (r *runtimeField) UnmarshalYAML(value *yaml.Node) error {
	// For some reason go-yaml doesn't like the UnmarshalJSON function above.
	return r.unmarshalString(value.Value)
}

type position struct {
	file   string
	line   int
	column int
}

func (p position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.file, p.line, p.column)
}

type processor struct {
	Type       string
	Attributes map[string]any
	OnFailure  []processor

	position position
}

func (p *processor) UnmarshalYAML(value *yaml.Node) error {
	var procMap map[string]struct {
		Attributes map[string]any `yaml:",inline"`
		OnFailure  []processor    `yaml:"on_failure"`
	}
	if err := value.Decode(&procMap); err != nil {
		return err
	}

	for k, v := range procMap {
		p.Type = k
		p.Attributes = v.Attributes
		p.OnFailure = v.OnFailure
		break
	}

	p.position.line = value.Line
	p.position.column = value.Column

	return nil
}

type ingestPipeline struct {
	Processors []processor `yaml:"processors"`
}

type field struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Unit       string `yaml:"unit"`
	DateFormat string `yaml:"date_format"`
	MetricType string `yaml:"metric_type"`
	Dimension  bool   `yaml:"dimension"`
	External   string `yaml:"external"`

	Runtime runtimeField `yaml:"runtime"`

	Fields fields `yaml:"fields"`
}

type fieldFileMetadata struct {
	dataStream   string
	transform    string
	filePath     string
	fullFilePath string
}

type pipelineFileMetadata struct {
	dataStream   string
	filePath     string
	fullFilePath string
}

type validateFunc func(fileMetadata fieldFileMetadata, f field) specerrors.ValidationErrors

func validateFields(fsys fspath.FS, validate validateFunc) specerrors.ValidationErrors {
	fieldsFilesMetadata, err := listFieldsFiles(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("can't list fields files: %w", err),
		}
	}

	var vErrs specerrors.ValidationErrors
	for _, metadata := range fieldsFilesMetadata {
		unmarshaled, err := unmarshalFields(fsys, metadata.filePath)
		if err != nil {
			anError := specerrors.NewStructuredErrorf(`file "%s" is invalid: can't unmarshal fields: %w`, metadata.filePath, err)
			vErrs = append(vErrs, anError)
		}

		errs := validateNestedFields("", metadata, unmarshaled, validate)
		if len(errs) > 0 {
			vErrs = append(vErrs, errs...)
		}
	}
	return vErrs
}

func validateNestedFields(parent string, metadata fieldFileMetadata, fields fields, validate validateFunc) specerrors.ValidationErrors {
	var result specerrors.ValidationErrors
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
					transform:    "",
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
				transform:    "",
			})
	}

	// transform definitions
	transforms, err := listTransforms(fsys)
	if err != nil {
		return nil, err
	}

	for _, transform := range transforms {
		fieldsDir := path.Join("elasticsearch", "transform", transform, "fields")
		transformFieldsFiles, err := readFieldsFolder(fsys, fieldsDir)
		if err != nil {
			return nil, fmt.Errorf("cannot read fields file from integration packages: %w", err)
		}

		for _, file := range transformFieldsFiles {
			fieldsFilesMetadata = append(
				fieldsFilesMetadata,
				fieldFileMetadata{
					filePath:     file,
					fullFilePath: fsys.Path(file),
					dataStream:   "",
					transform:    transform,
				})
		}
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
		return nil, fmt.Errorf("can't list fields directory (path: %s): %w", fsys.Path(fieldsDir), err)
	}

	for _, f := range fs {
		fieldsFiles = append(fieldsFiles, path.Join(fieldsDir, f.Name()))
	}
	return fieldsFiles, nil
}

func readPipelinesFolder(fsys fspath.FS, pipelinesDir string) ([]string, error) {
	var pipelineFiles []string
	entries, err := fs.ReadDir(fsys, pipelinesDir)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("can't list pipelines directory (path: %s): %w", fsys.Path(pipelinesDir), err)
	}

	for _, v := range entries {
		pipelineFiles = append(pipelineFiles, path.Join(pipelinesDir, v.Name()))
	}

	return pipelineFiles, nil
}

func unmarshalFields(fsys fspath.FS, fieldsPath string) (fields, error) {
	content, err := fs.ReadFile(fsys, fieldsPath)
	if err != nil {
		return nil, fmt.Errorf("can't read file (path: %s): %w", fieldsPath, err)
	}

	var f fields
	err = yaml.Unmarshal(content, &f)
	if err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal failed (path: %s): %w", fieldsPath, err)
	}
	return f, nil
}

func listDataStreams(fsys fspath.FS) ([]string, error) {
	dataStreams, err := fs.ReadDir(fsys, dataStreamDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("can't list data streams directory: %w", err)
	}

	list := make([]string, len(dataStreams))
	for i, dataStream := range dataStreams {
		list[i] = dataStream.Name()
	}
	return list, nil
}

func listTransforms(fsys fspath.FS) ([]string, error) {
	transformDirectory := path.Join("elasticsearch", "transform")
	transforms, err := fs.ReadDir(fsys, transformDirectory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("can't list transforms directory: %w", err)
	}

	list := make([]string, len(transforms))
	for i, transform := range transforms {
		list[i] = transform.Name()
	}
	return list, nil

}

func listPipelineFiles(fsys fspath.FS, dataStream string) ([]pipelineFileMetadata, error) {
	var pipelineFileMetadatas []pipelineFileMetadata

	ingestPipelineDir := path.Join(dataStreamDir, dataStream, "elasticsearch", "ingest_pipeline")
	pipelineFiles, err := readPipelinesFolder(fsys, ingestPipelineDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read pipeline files from integration package: %w", err)
	}

	for _, file := range pipelineFiles {
		pipelineFileMetadatas = append(pipelineFileMetadatas, pipelineFileMetadata{
			filePath:     file,
			fullFilePath: fsys.Path(file),
			dataStream:   dataStream,
		})
	}

	return pipelineFileMetadatas, nil
}
