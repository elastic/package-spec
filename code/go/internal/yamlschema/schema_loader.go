// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"io/fs"
	"io/ioutil"
	"net/url"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/elastic/gojsonschema"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonreference"
	"gopkg.in/yaml.v3"
)

type yamlReferenceLoader struct {
	fs     fs.FS
	source string
}

var _ gojsonschema.JSONLoader = new(yamlReferenceLoader)

type rawReferenceLoader struct {
	fs     fs.FS
	source interface{}
}

// NewReferenceLoaderFileSystem method creates new instance of `yamlReferenceLoader`.
func NewReferenceLoaderFileSystem(source string, fs fs.FS) gojsonschema.JSONLoader {
	return &yamlReferenceLoader{
		fs:     fs,
		source: source,
	}
}

func (l *yamlReferenceLoader) JsonSource() interface{} { // golint:ignore
	return l.source
}

func (l *yamlReferenceLoader) LoadJSON() (interface{}, error) {
	parsed, err := url.Parse(l.source)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing source failed (source: %s)", l.source)
	}
	resourcePath := strings.TrimPrefix(parsed.Path, "/")

	itemSchemaFile, err := l.fs.Open(resourcePath)
	if err != nil {
		return nil, errors.Wrapf(err, "opening schema file failed (path: %s)", resourcePath)
	}
	defer itemSchemaFile.Close()

	itemSchemaData, err := ioutil.ReadAll(itemSchemaFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading schema file failed")
	}

	if len(itemSchemaData) == 0 {
		return nil, errors.New("schema file is empty")
	}

	var schema itemSchemaSpec
	err = yaml.Unmarshal(itemSchemaData, &schema)
	if err != nil {
		return nil, errors.Wrapf(err, "schema unmarshalling failed (path: %s)", l.source)
	}

	v := semver.MustParse("1.0.0") // FIXME: Get an actual version here.
	return schema.resolve(*v)
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
		fs: l.fs,
	}
}

type fileSystemYAMLLoaderFactory struct {
	fs fs.FS
}

var _ gojsonschema.JSONLoaderFactory = new(fileSystemYAMLLoaderFactory)

func (f *fileSystemYAMLLoaderFactory) New(source string) gojsonschema.JSONLoader {
	return &yamlReferenceLoader{
		fs:     f.fs,
		source: source,
	}
}
