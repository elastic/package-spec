// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/elastic/gojsonschema"

	"github.com/xeipuuv/gojsonreference"
	"gopkg.in/yaml.v3"
)

type yamlReferenceLoader struct {
	fs      fs.FS
	source  string
	version semver.Version
}

var _ gojsonschema.JSONLoader = new(yamlReferenceLoader)

// NewReferenceLoaderFileSystem method creates new instance of `yamlReferenceLoader`.
func NewReferenceLoaderFileSystem(source string, fs fs.FS, version semver.Version) gojsonschema.JSONLoader {
	return &yamlReferenceLoader{
		fs:      fs,
		source:  source,
		version: version,
	}
}

func (l *yamlReferenceLoader) JsonSource() any { // golint:ignore
	return l.source
}

func (l *yamlReferenceLoader) LoadJSON() (any, error) {
	parsed, err := url.Parse(l.source)
	if err != nil {
		return nil, fmt.Errorf("parsing source failed (source: %s): %w", l.source, err)
	}
	resourcePath := strings.TrimPrefix(parsed.Path, "/")

	itemSchemaData, err := fs.ReadFile(l.fs, resourcePath)
	if err != nil {
		return nil, fmt.Errorf("reading schema file failed: %w", err)
	}
	if len(itemSchemaData) == 0 {
		return nil, errors.New("schema file is empty")
	}

	var schema itemSchemaSpec
	err = yaml.Unmarshal(itemSchemaData, &schema)
	if err != nil {
		return nil, fmt.Errorf("schema unmarshalling failed (path: %s): %w", l.source, err)
	}

	// fixJSONNumbers ensures that the numbers in the resulting spec are of type `json.Number`, that is
	// what the gojsonschema library expects. Without this, gojsonschema complains about some integer types
	// not being integers.
	// TODO: This shouldn't probably be needed, we could try to fix gojsonschema to accept other integers, or
	// look for a YAML parser that can be customized to use `json.Number`.
	schema.Spec, err = fixJSONNumbers(schema.Spec)
	if err != nil {
		return nil, fmt.Errorf("fixing numbers in parsed schema failed (path %s): %w", l.source, err)
	}

	return schema.resolve(l.version)
}

// fixJSONNumbers converts number types to `json.Number` by converting the struct to JSON and decoding it again.
func fixJSONNumbers[T any](v T) (result T, err error) {
	d, err := json.Marshal(v)
	if err != nil {
		return result, err
	}

	dec := json.NewDecoder(bytes.NewReader(d))
	dec.UseNumber()
	return result, dec.Decode(&result)
}

func (l *yamlReferenceLoader) JsonReference() (gojsonreference.JsonReference, error) {
	r, err := gojsonreference.NewJsonReference(l.JsonSource().(string))
	if err != nil {
		return r, err
	}

	// gojsonreference uses filepath to decide if the reference has a full file path,
	// and in Windows it has additional special handling.
	// Here we are operating on a fs.FS, where '/' is always used as separator, also on
	// Windows. Override the value with the result of `path.IsAbs`.
	r.HasFullFilePath = path.IsAbs(r.GetUrl().Path)

	return r, nil
}

func (l *yamlReferenceLoader) LoaderFactory() gojsonschema.JSONLoaderFactory {
	return &fileSystemYAMLLoaderFactory{
		fs:      l.fs,
		version: l.version,
	}
}

type fileSystemYAMLLoaderFactory struct {
	fs      fs.FS
	version semver.Version
}

var _ gojsonschema.JSONLoaderFactory = new(fileSystemYAMLLoaderFactory)

func (f *fileSystemYAMLLoaderFactory) New(source string) gojsonschema.JSONLoader {
	return &yamlReferenceLoader{
		fs:      f.fs,
		source:  source,
		version: f.version,
	}
}
