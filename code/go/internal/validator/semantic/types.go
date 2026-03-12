// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// PackageFS is the filesystem interface that validators use to access package
// files. It is satisfied by *pkgpath.CachedFS, which caches file access.
type PackageFS interface {
	// Files finds files matching the glob pattern.
	Files(glob string) ([]pkgpath.File, error)

	// Path returns a path for the given names, based on the location of
	// the underlying filesystem. Used for error messages.
	Path(names ...string) string

	// LoadOrStore returns the cached value for key, or calls compute,
	// stores and returns the result. Useful for caching derived data.
	LoadOrStore(key string, compute func() (any, error)) (any, error)
}

const (
	dataStreamDir = "data_stream"

	inputPackageType       = "input"
	integrationPackageType = "integration"
)

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

func (p *processor) GetAttributeString(key string) (string, bool) {
	s, ok := p.Attributes[key].(string)
	if !ok {
		return "", false
	}

	return s, true
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
	OnFailure  []processor `yaml:"on_failure"`
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

func validateFields(fsys PackageFS, validate validateFunc) specerrors.ValidationErrors {
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

func listFieldsFiles(fsys PackageFS) ([]fieldFileMetadata, error) {
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

func readFieldsFolder(fsys PackageFS, fieldsDir string) ([]string, error) {
	entries, err := fsys.Files(fieldsDir + "/*")
	if err != nil {
		return nil, fmt.Errorf("can't list fields directory (path: %s): %w", fsys.Path(fieldsDir), err)
	}

	var fieldsFiles []string
	for _, f := range entries {
		fieldsFiles = append(fieldsFiles, f.Path())
	}
	return fieldsFiles, nil
}

func readPipelinesFolder(fsys PackageFS, pipelinesDir string) ([]string, error) {
	entries, err := fsys.Files(pipelinesDir + "/*")
	if err != nil {
		return nil, fmt.Errorf("can't list pipelines directory (path: %s): %w", fsys.Path(pipelinesDir), err)
	}

	var pipelineFiles []string
	for _, v := range entries {
		pipelineFiles = append(pipelineFiles, v.Path())
	}
	return pipelineFiles, nil
}

func unmarshalFields(fsys PackageFS, fieldsPath string) (fields, error) {
	key := "unmarshalFields:" + fieldsPath
	result, err := fsys.LoadOrStore(key, func() (any, error) {
		files, err := fsys.Files(fieldsPath)
		if err != nil {
			return nil, fmt.Errorf("can't read file (path: %s): %w", fieldsPath, err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("can't read file (path: %s): file not found", fieldsPath)
		}

		content, err := files[0].ReadAll()
		if err != nil {
			return nil, fmt.Errorf("can't read file (path: %s): %w", fieldsPath, err)
		}

		var f fields
		if err := yaml.Unmarshal(content, &f); err != nil {
			return nil, fmt.Errorf("yaml.Unmarshal failed (path: %s): %w", fieldsPath, err)
		}
		return f, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(fields), nil
}

func listDataStreams(fsys PackageFS) ([]string, error) {
	entries, err := fsys.Files(dataStreamDir + "/*")
	if err != nil {
		return nil, fmt.Errorf("can't list data streams directory: %w", err)
	}

	var list []string
	for _, entry := range entries {
		if entry.IsDir() {
			list = append(list, entry.Name())
		}
	}
	return list, nil
}

func listTransforms(fsys PackageFS) ([]string, error) {
	transformDirectory := path.Join("elasticsearch", "transform")
	entries, err := fsys.Files(transformDirectory + "/*")
	if err != nil {
		return nil, fmt.Errorf("can't list transforms directory: %w", err)
	}

	var list []string
	for _, entry := range entries {
		if entry.IsDir() {
			list = append(list, entry.Name())
		}
	}
	return list, nil
}

func listPipelineFiles(fsys PackageFS) ([]pipelineFileMetadata, error) {
	var pipelineFileMetadatas []pipelineFileMetadata

	type pipelineDirMetadata struct {
		dir        string
		dataStream string
	}

	// Empty directory is used here to read ingest pipelines defined in the package root.
	dirs := []pipelineDirMetadata{{dir: ""}}

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return nil, specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}
	for _, dataStream := range dataStreams {
		dirs = append(dirs, pipelineDirMetadata{
			dir:        path.Join(dataStreamDir, dataStream),
			dataStream: dataStream,
		})
	}

	for _, d := range dirs {
		pipelinePath := path.Join(d.dir, "elasticsearch", "ingest_pipeline")
		pipelineFiles, err := readPipelinesFolder(fsys, pipelinePath)
		if err != nil {
			return nil, fmt.Errorf("cannot read pipeline files from integration package: %w", err)
		}
		for _, file := range pipelineFiles {
			pipelineFileMetadatas = append(pipelineFileMetadatas, pipelineFileMetadata{
				filePath:     file,
				fullFilePath: fsys.Path(file),
				dataStream:   d.dataStream,
			})
		}
	}

	return pipelineFileMetadatas, nil
}
